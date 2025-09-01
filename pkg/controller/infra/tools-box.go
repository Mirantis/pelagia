/*
Copyright 2025 Mirantis IT.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package infra

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

const toolBoxScript = `#!/bin/bash -e
# Replicate the script from toolbox.sh inline so the ceph image
# can be run directly, instead of requiring the rook toolbox
CEPH_CONFIG="/etc/ceph/ceph.conf"
MON_CONFIG="/etc/rook/mon-endpoints"
KEYRING_FILE="/etc/ceph/keyring"

# create a ceph config file in its default location so ceph/rados tools can be used
# without specifying any arguments
write_endpoints() {
  endpoints=$(cat ${MON_CONFIG})

  # filter out the mon names
  # external cluster can have numbers or hyphens in mon names, handling them in regex
  # shellcheck disable=SC2001
  mon_endpoints=$(echo "${endpoints}"| sed 's/[a-z0-9_-]\+=//g')

  DATE=$(date)
  echo "$DATE writing mon endpoints to ${CEPH_CONFIG}: ${endpoints}"
    cat <<EOF > ${CEPH_CONFIG}
[global]
mon_host = ${mon_endpoints}

[client.admin]
keyring = ${KEYRING_FILE}
EOF
}

# watch the endpoints config file and update if the mon endpoints ever change
watch_endpoints() {
  # get the timestamp for the target of the soft link
  real_path=$(realpath ${MON_CONFIG})
  initial_time=$(stat -c %Z "${real_path}")
  while true; do
    real_path=$(realpath ${MON_CONFIG})
    latest_time=$(stat -c %Z "${real_path}")

    if [[ "${latest_time}" != "${initial_time}" ]]; then
      write_endpoints
      initial_time=${latest_time}
    fi

    sleep 10
  done
}

# read the secret from an env var (for backward compatibility), or from the secret file
ceph_secret=${ROOK_CEPH_SECRET}
if [[ "$ceph_secret" == "" ]]; then
  ceph_secret=$(cat /var/lib/rook-ceph-mon/secret.keyring)
fi

# create the keyring file
cat <<EOF > ${KEYRING_FILE}
[${ROOK_CEPH_USERNAME}]
key = ${ceph_secret}
EOF

# write the initial config file
write_endpoints

# continuously update the mon endpoints if they fail over
watch_endpoints
`

func (c *cephDeploymentInfraConfig) ensureToolBox() error {
	cephToolsGenerated, err := c.generateToolBox()
	if err != nil {
		return errors.Wrapf(err, "failed to generate toolbox deployment '%s/%s'", c.lcmConfig.RookNamespace, lcmcommon.PelagiaToolBox)
	}
	cephTools, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Get(c.context, lcmcommon.PelagiaToolBox, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.log.Info().Msgf("creating toolbox deployment %s/%s", cephToolsGenerated.Namespace, cephToolsGenerated.Name)
			_, err = c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Create(c.context, cephToolsGenerated, metav1.CreateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msg("")
				return errors.Wrapf(err, "failed to create toolbox deployment '%s/%s", cephToolsGenerated.Namespace, cephToolsGenerated.Name)
			}
			return nil
		}
		c.log.Error().Err(err).Msg("")
		return errors.Wrapf(err, "failed to check toolbox deployment '%s/%s", cephToolsGenerated.Namespace, cephToolsGenerated.Name)
	}
	if !reflect.DeepEqual(cephTools.Spec, cephToolsGenerated.Spec) || c.checkLabelsAndOwnerRefs(&cephTools.ObjectMeta, &cephToolsGenerated.ObjectMeta) {
		c.log.Info().Msgf("update toolbox deployment %s/%s", cephTools.Namespace, cephTools.Name)
		lcmcommon.ShowObjectDiff(*c.log, cephTools.Spec, cephToolsGenerated.Spec)
		cephTools.Spec = cephToolsGenerated.Spec
		_, err = c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Update(c.context, cephTools, metav1.UpdateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return errors.Wrapf(err, "failed to update toolbox deployment '%s/%s", cephToolsGenerated.Namespace, cephToolsGenerated.Name)
		}
		return nil
	}
	if !lcmcommon.IsDeploymentReady(cephTools) {
		msg := fmt.Sprintf("replicas desired: %d, ready: %d, updated: %d",
			cephTools.Status.Replicas, cephTools.Status.ReadyReplicas, cephTools.Status.UpdatedReplicas)
		c.log.Warn().Msgf("toolbox deployment '%s/%s' is not ready yet (%s)", cephTools.Namespace, cephTools.Name, msg)
	}
	return nil
}

func (c *cephDeploymentInfraConfig) generateToolBox() (*apps.Deployment, error) {
	rookDeployment, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Get(c.context, lcmcommon.RookCephOperatorName, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, errors.Wrapf(err, "failed to check '%s/%s' deployment", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
	}
	objectStores, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, errors.Wrap(err, "failed to check cephobjectstores")
	}
	imageName := ""
	for _, container := range rookDeployment.Spec.Template.Spec.Containers {
		if container.Name == "rook-ceph-operator" {
			imageName = container.Image
			break
		}
	}

	toolBoxReplicas := int32(1)
	toolBox := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            lcmcommon.PelagiaToolBox,
			Namespace:       c.lcmConfig.RookNamespace,
			Labels:          map[string]string{"app": lcmcommon.PelagiaToolBox},
			OwnerReferences: c.infraConfig.cephOwnerRefs,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": lcmcommon.PelagiaToolBox},
			},
			Replicas:                &toolBoxReplicas,
			RevisionHistoryLimit:    &revisionHistoryLimit,
			ProgressDeadlineSeconds: &[]int32{60}[0],
			Strategy:                rookDeployment.Spec.Strategy,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": lcmcommon.PelagiaToolBox},
				},
			},
		},
	}

	toolBox.Spec.Template.Spec = rookDeployment.Spec.Template.Spec
	toolBox.Spec.Template.Spec.DNSPolicy = "ClusterFirstWithHostNet"
	// unset accounts if any
	toolBox.Spec.Template.Spec.DeprecatedServiceAccount = ""
	toolBox.Spec.Template.Spec.ServiceAccountName = ""
	toolBox.Spec.Template.Spec.InitContainers = nil
	toolBox.Spec.Template.Spec.Containers = []v1.Container{
		{
			Name:    lcmcommon.PelagiaToolBox,
			Image:   imageName,
			Command: []string{"/bin/bash", "-c"},
			Args:    []string{toolBoxScript},
			SecurityContext: &v1.SecurityContext{
				Capabilities:             &v1.Capabilities{Drop: []v1.Capability{"ALL"}},
				RunAsUser:                &rookUserID,
				RunAsGroup:               &rookUserID,
				AllowPrivilegeEscalation: &falseVar,
				RunAsNonRoot:             &trueVar,
			},
			TerminationMessagePath:   "/dev/termination-log",
			TerminationMessagePolicy: "File",
			ImagePullPolicy:          "IfNotPresent",
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "ceph-config",
					MountPath: "/etc/ceph",
				},
				{
					Name:      "mon-endpoint",
					MountPath: "/etc/rook",
				},
			},
			Env: []v1.EnvVar{
				{
					Name: "ROOK_CEPH_USERNAME",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{Name: lcmcommon.RookCephMonSecretName},
							Key:                  "ceph-username",
						},
					},
				},
				{
					Name: "ROOK_CEPH_SECRET",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{Name: lcmcommon.RookCephMonSecretName},
							Key:                  "ceph-secret",
						},
					},
				},
			},
		},
	}
	toolBox.Spec.Template.Spec.Volumes = []v1.Volume{
		{
			Name: "mon-endpoint",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: lcmcommon.MonMapConfigMapName},
					DefaultMode:          &[]int32{420}[0],
					Items: []v1.KeyToPath{
						{
							Key:  "data",
							Path: "mon-endpoints",
						},
					},
				},
			},
		},
		{
			Name:         "ceph-config",
			VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
		},
	}

	// if ceph cluster is external and non-admin then we need to add client name
	// as a default client for toolbox, if it specified
	if c.infraConfig.externalCeph {
		toolBox.Spec.Template.Spec.Containers[0].Env = append(
			toolBox.Spec.Template.Spec.Containers[0].Env,
			v1.EnvVar{
				Name: "CEPH_ARGS",
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: lcmcommon.RookCephMonSecretName},
						Key:                  "ceph-args",
					},
				},
			},
		)
	}

	if len(objectStores.Items) == 0 || c.infraConfig.externalCeph {
		return toolBox, nil
	}
	secrets := []string{}
	for _, store := range objectStores.Items {
		if store.Spec.Gateway.CaBundleRef != "" {
			if toolBox.Spec.Template.Annotations == nil {
				toolBox.Spec.Template.Annotations = map[string]string{}
			}
			secret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, store.Spec.Gateway.CaBundleRef, metav1.GetOptions{})
			if err != nil {
				c.log.Error().Err(err).Msg("")
				return nil, errors.Wrapf(err, "failed to get secret '%s/%s' with cabundle for CephObjectStore '%s/%s'",
					c.lcmConfig.RookNamespace, store.Spec.Gateway.CaBundleRef, store.Namespace, store.Name)
			}
			secrets = append(secrets, store.Spec.Gateway.CaBundleRef)
			toolBox.Spec.Template.Annotations[fmt.Sprintf("%s/sha256", store.Spec.Gateway.CaBundleRef)] = lcmcommon.GetStringSha256(string(secret.Data["cabundle"]))
		}
	}
	// mount rgw related cabundles for rgw api direct access from tools
	if len(secrets) > 0 {
		volumeCaBundleSecret := "cabundle-secret"
		volumeCaBundleUpdated := "cabundle-updated"
		toolBox.Spec.Template.Spec.InitContainers = []v1.Container{
			{
				Name:    "cabundle-update",
				Image:   imageName,
				Command: []string{"/bin/bash", "-c"},
				Args:    []string{"/usr/bin/update-ca-trust extract; cp -rf /etc/pki/ca-trust/extracted//* /tmp/new-ca-bundle/"},
				SecurityContext: &v1.SecurityContext{
					Capabilities:             &v1.Capabilities{Drop: []v1.Capability{"ALL"}},
					RunAsUser:                &rootUserID,
					RunAsGroup:               &rootUserID,
					Privileged:               &falseVar,
					AllowPrivilegeEscalation: &falseVar,
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: "File",
				ImagePullPolicy:          "IfNotPresent",
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      volumeCaBundleSecret,
						MountPath: "/etc/pki/ca-trust/source/anchors/",
						ReadOnly:  true,
					},
					{
						Name:      volumeCaBundleUpdated,
						MountPath: "/tmp/new-ca-bundle/",
					},
				},
			},
		}
		toolBox.Spec.Template.Spec.Containers[0].VolumeMounts = append(toolBox.Spec.Template.Spec.Containers[0].VolumeMounts,
			v1.VolumeMount{
				Name:      volumeCaBundleUpdated,
				MountPath: "/etc/pki/ca-trust/extracted/",
				ReadOnly:  true,
			})
		toolBox.Spec.Template.Spec.Volumes = append(toolBox.Spec.Template.Spec.Volumes,
			v1.Volume{
				Name: volumeCaBundleUpdated,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			})
		if len(secrets) == 1 {
			toolBox.Spec.Template.Spec.Volumes = append(toolBox.Spec.Template.Spec.Volumes,
				v1.Volume{
					Name: volumeCaBundleSecret,
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName:  secrets[0],
							DefaultMode: &[]int32{420}[0],
							Items: []v1.KeyToPath{
								{
									Key:  "cabundle",
									Path: fmt.Sprintf("%s.crt", secrets[0]),
									Mode: &[]int32{256}[0],
								},
							},
						},
					},
				})
		} else {
			// case for multiinstance setup to have access from toolbox to all at the same time
			sources := []v1.VolumeProjection{}
			for _, secret := range secrets {
				vp := v1.VolumeProjection{
					Secret: &v1.SecretProjection{
						LocalObjectReference: v1.LocalObjectReference{Name: secret},
						Items: []v1.KeyToPath{
							{
								Key:  "cabundle",
								Path: fmt.Sprintf("%s.crt", secret),
								Mode: &[]int32{256}[0],
							},
						},
					},
				}
				sources = append(sources, vp)
			}
			toolBox.Spec.Template.Spec.Volumes = append(toolBox.Spec.Template.Spec.Volumes,
				v1.Volume{
					Name: volumeCaBundleSecret,
					VolumeSource: v1.VolumeSource{
						Projected: &v1.ProjectedVolumeSource{Sources: sources},
					},
				})
		}
	}
	return toolBox, nil
}
