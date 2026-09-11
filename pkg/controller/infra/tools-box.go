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

	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
)

// toolbox script version: Rook v1.20.7
const toolBoxScript = `#!/bin/bash -e
# Replicate the script from toolbox.sh inline so the ceph image
# can be run directly, instead of requiring the rook toolbox
ENDPOINT_MOUNT="/etc/rook/mon-endpoints"
KEYRING_MOUNT="/var/lib/rook-ceph-mon/secret.keyring"
CONFIG_OVERRIDE_MOUNT="/etc/rook-config-override/config"

# outputs
CEPH_CONFIG="/etc/ceph/ceph.conf"
KEYRING_FILE="/etc/ceph/keyring"

config_override_exists() {
  [ -f "${CONFIG_OVERRIDE_MOUNT}" ] && [ -s "${CONFIG_OVERRIDE_MOUNT}" ]
}

# create/update the ceph keyring file
write_keyring() {
  DATE=$(date)

  # read the secret from an env var (for backward compatibility), or from the secret file
  ceph_secret=${ROOK_CEPH_SECRET}
  if [[ "$ceph_secret" == "" ]]; then
    ceph_secret=$(cat ${KEYRING_MOUNT})
  fi

  echo "$DATE writing keyring to ${KEYRING_FILE}"
  cat <<EOF > ${KEYRING_FILE}
[${ROOK_CEPH_USERNAME}]
key = ${ceph_secret}
EOF

} # write_keyring()

# create/update the ceph config file in its default location so ceph/rados tools can be used
# without specifying any arguments
write_conf() {
  DATE=$(date)

  # filter out the mon names
  # external cluster can have numbers or hyphens in mon names, handling them in regex
  # shellcheck disable=SC2001
  mon_endpoints=$(cat ${ENDPOINT_MOUNT}| sed 's/[a-z0-9_-]\+=//g')

  config_override=""
  if config_override_exists; then
    echo "$DATE merging config override from ${CONFIG_OVERRIDE_MOUNT}"
    config_override=$(cat "${CONFIG_OVERRIDE_MOUNT}")
  fi

  echo "$DATE writing mon endpoints to ${CEPH_CONFIG}: ${mon_endpoints}"
  cat << EOF > ${CEPH_CONFIG}
[global]
mon_host = ${mon_endpoints}

[client.admin]
keyring = ${KEYRING_FILE}

${config_override}
EOF

} # write_conf()

# watch the ceph config directory and update if the mon endpoints or keyring change
watch_etc_ceph() {
  # get the timestamp for the targets of the soft links
  keyring_mount_path=$(realpath "${KEYRING_MOUNT}")
  keyring_init_time=$(stat -c %Z "${keyring_mount_path}")

  endpoint_mount_path=$(realpath "${ENDPOINT_MOUNT}")
  endpoint_init_time=$(stat -c %Z "${endpoint_mount_path}")

  override_mount_path=""
  override_init_time=""
  if config_override_exists; then
    override_mount_path=$(realpath "${CONFIG_OVERRIDE_MOUNT}")
    override_init_time=$(stat -c %Z "${override_mount_path}")
  fi

  while true; do
    sleep 10
    DATE=$(date)

    # keyring file
    keyring_mount_path=$(realpath "${KEYRING_MOUNT}")
    keyring_latest_time="$(stat -c %Z "${keyring_mount_path}")"

    if [ "${keyring_latest_time}" != "${keyring_init_time}" ]; then
      echo "$DATE keyring changed"
      write_keyring
      keyring_init_time="${keyring_latest_time}"
    fi

    # ceph.conf file / config override
    endpoint_mount_path=$(realpath "${ENDPOINT_MOUNT}")
    endpoint_latest_time=$(stat -c %Z "${endpoint_mount_path}")

    override_mount_path=""
    override_latest_time=""
    if config_override_exists; then
      override_mount_path=$(realpath "${CONFIG_OVERRIDE_MOUNT}")
      override_latest_time=$(stat -c %Z "${override_mount_path}")
    fi

    need_conf_update=false

    if [ "${endpoint_latest_time}" != "${endpoint_init_time}" ]; then
      echo "$DATE mon endpoints changed"
      need_conf_update=true
    fi

    if [ "${override_latest_time}" != "${override_init_time}" ]; then
      echo "$DATE config override changed"
      need_conf_update=true
    fi

    if $need_conf_update; then
      write_conf
      endpoint_init_time="${endpoint_latest_time}"
      override_init_time="${override_latest_time}"
    fi
  done
}

# write the initial config files
write_keyring
write_conf

# continuously update the config files for mon failover and/or cephx key rotation
if [ "$1" != "--skip-watch" ]; then
  watch_etc_ceph
fi
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
				return errors.Wrapf(err, "failed to create toolbox deployment '%s/%s'", cephToolsGenerated.Namespace, cephToolsGenerated.Name)
			}
			return nil
		}
		c.log.Error().Err(err).Msg("")
		return errors.Wrapf(err, "failed to check toolbox deployment '%s/%s'", cephToolsGenerated.Namespace, cephToolsGenerated.Name)
	}
	if !reflect.DeepEqual(cephTools.Spec, cephToolsGenerated.Spec) || c.checkLabelsAndOwnerRefs(&cephTools.ObjectMeta, &cephToolsGenerated.ObjectMeta, "deployment") {
		c.log.Info().Msgf("update toolbox deployment %s/%s", cephTools.Namespace, cephTools.Name)
		lcmcommon.ShowObjectDiff(*c.log, cephTools.Spec, cephToolsGenerated.Spec)
		cephTools.Spec = cephToolsGenerated.Spec
		_, err = c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Update(c.context, cephTools, metav1.UpdateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return errors.Wrapf(err, "failed to update toolbox deployment '%s/%s'", cephToolsGenerated.Namespace, cephToolsGenerated.Name)
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
			Labels:          lcmcommon.ExtendLabels(map[string]string{"app": lcmcommon.PelagiaToolBox}, baseResourceLabels),
			OwnerReferences: c.infraConfig.cephOwnerRefs,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": lcmcommon.PelagiaToolBox},
			},
			Replicas:                &toolBoxReplicas,
			RevisionHistoryLimit:    &revisionHistoryLimit,
			ProgressDeadlineSeconds: lcmcommon.PtrTo(int32(60)),
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
					DefaultMode:          lcmcommon.PtrTo(int32(420)),
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
			keyName := fmt.Sprintf("%s/sha256", store.Spec.Gateway.CaBundleRef)
			if _, ok := toolBox.Spec.Template.Annotations[keyName]; !ok {
				secrets = append(secrets, store.Spec.Gateway.CaBundleRef)
				toolBox.Spec.Template.Annotations[keyName] = lcmcommon.GetStringSha256(string(secret.Data["cabundle"]))
			}
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
				Args:    []string{"/usr/bin/update-ca-trust extract -o /tmp/new-ca-bundle/"},
				SecurityContext: &v1.SecurityContext{
					Capabilities:             &v1.Capabilities{Drop: []v1.Capability{"ALL"}},
					RunAsUser:                &rookUserID,
					RunAsGroup:               &rookUserID,
					RunAsNonRoot:             &trueVar,
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
							DefaultMode: lcmcommon.PtrTo(int32(420)),
							Items: []v1.KeyToPath{
								{
									Key:  "cabundle",
									Path: fmt.Sprintf("%s.crt", secrets[0]),
									Mode: lcmcommon.PtrTo(int32(256)),
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
								Mode: lcmcommon.PtrTo(int32(256)),
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
