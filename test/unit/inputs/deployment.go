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

package input

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DeploymentListEmpty = &appsv1.DeploymentList{Items: []appsv1.Deployment{}}
var DeploymentList = &appsv1.DeploymentList{Items: []appsv1.Deployment{*RookDeploymentLatestVersion}}
var DeploymentListWithCSIReady = &appsv1.DeploymentList{
	Items: []appsv1.Deployment{
		*RookDeploymentLatestVersion, *GetDeploymentWithStatus("ceph-csi-controller-manager", "rook-ceph", nil, 1, 1),
		*GetDeploymentWithStatus("rook-ceph.cephfs.csi.ceph.com-ctrlplugin", "rook-ceph", nil, 2, 2), *GetDeploymentWithStatus("rook-ceph.rbd.csi.ceph.com-ctrlplugin", "rook-ceph", nil, 2, 2),
	},
}
var DeploymentListWithCSINotReady = &appsv1.DeploymentList{
	Items: []appsv1.Deployment{
		*RookDeploymentLatestVersion, *GetDeploymentWithStatus("ceph-csi-controller-manager", "rook-ceph", nil, 1, 0),
		*GetDeploymentWithStatus("rook-ceph.cephfs.csi.ceph.com-ctrlplugin", "rook-ceph", nil, 2, 0), *GetDeploymentWithStatus("rook-ceph.rbd.csi.ceph.com-ctrlplugin", "rook-ceph", nil, 2, 0),
	},
}

func GetDeployment(name, namespace string, labels map[string]string, replicas *int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
		},
	}
}

func GetDeploymentWithStatus(name, namespace string, labels map[string]string, desiredReplicas, readyReplicas int32) *appsv1.Deployment {
	deploy := GetDeployment(name, namespace, labels, &desiredReplicas)
	deploy.Status = appsv1.DeploymentStatus{
		Replicas:          desiredReplicas,
		UpdatedReplicas:   readyReplicas,
		ReadyReplicas:     readyReplicas,
		AvailableReplicas: readyReplicas,
	}
	return deploy
}

func GetRookDeployment(image string, desiredReplicas, readyReplicas int32) *appsv1.Deployment {
	operator := GetDeployment("rook-ceph-operator", RookNamespace, nil, nil)
	operator.Status = appsv1.DeploymentStatus{
		Replicas:          desiredReplicas,
		UpdatedReplicas:   desiredReplicas,
		ReadyReplicas:     readyReplicas,
		AvailableReplicas: readyReplicas,
	}
	operator.Spec.Replicas = &desiredReplicas
	operator.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "rook-ceph-operator",
				Image: image,
			},
		},
	}
	return operator
}

var RookDeploymentPrevVersion = GetRookDeployment(PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_ROOK_IMAGE"], 1, 1)
var RookDeploymentLatestVersion = GetRookDeployment(PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"], 1, 1)
var RookDeploymentNotScaled = GetRookDeployment(PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"], 0, 0)

func GetToolBoxDeployment(external bool) *appsv1.Deployment {
	deploy := GetDeployment("pelagia-ceph-toolbox", RookNamespace, map[string]string{"app": "pelagia-ceph-toolbox"}, &[]int32{1}[0])
	deploy.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "ceph.rook.io/v1",
			Kind:       "CephCluster",
			Name:       LcmObjectMeta.Name,
		},
	}
	envVars := []corev1.EnvVar{
		{
			Name: "ROOK_CEPH_USERNAME",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "rook-ceph-mon",
					},
					Key: "ceph-username",
				},
			},
		},
		{
			Name: "ROOK_CEPH_SECRET",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "rook-ceph-mon",
					},
					Key: "ceph-secret",
				},
			},
		},
	}
	if external {
		envVars = append(envVars, corev1.EnvVar{
			Name: "CEPH_ARGS",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "rook-ceph-mon"},
					Key:                  "ceph-args",
				},
			},
		})
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "pelagia-ceph-toolbox"},
		},
		Replicas:                &[]int32{1}[0],
		RevisionHistoryLimit:    &[]int32{5}[0],
		ProgressDeadlineSeconds: &[]int32{60}[0],
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "pelagia-ceph-toolbox"}},
			Spec: corev1.PodSpec{
				DNSPolicy:                "ClusterFirstWithHostNet",
				DeprecatedServiceAccount: "",
				ServiceAccountName:       "",
				Containers: []corev1.Container{
					{
						Name:    "pelagia-ceph-toolbox",
						Image:   PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"],
						Command: []string{"/bin/bash", "-c"},
						Args: []string{
							`#!/bin/bash -e
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
`,
						},
						SecurityContext: &corev1.SecurityContext{
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							RunAsUser:                &[]int64{2016}[0],
							RunAsGroup:               &[]int64{2016}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							RunAsNonRoot:             &[]bool{true}[0],
						},
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: "File",
						ImagePullPolicy:          "IfNotPresent",
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "ceph-config",
								MountPath: "/etc/ceph",
							},
							{
								Name:      "mon-endpoint",
								MountPath: "/etc/rook",
							},
						},
						Env: envVars,
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "mon-endpoint",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "rook-ceph-mon-endpoints",
								},
								DefaultMode: &[]int32{420}[0],
								Items: []corev1.KeyToPath{
									{
										Key:  "data",
										Path: "mon-endpoints",
									},
								},
							},
						},
					},
					{
						Name: "ceph-config",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}
	return deploy
}

var ToolBoxDeploymentBase = GetToolBoxDeployment(false)
var ToolBoxDeploymentExternal = GetToolBoxDeployment(true)
var ToolBoxDeploymentWithRgwSecret = func() *appsv1.Deployment {
	deploy := GetToolBoxDeployment(false)
	deploy.Spec.Template.Annotations = map[string]string{"rgw-ssl-certificate/sha256": "c448d82eeaebb5ab538f49a14a57ec788abffd242b43f8eba7b757a22c555005"}
	deploy.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:    "cabundle-update",
			Image:   deploy.Spec.Template.Spec.Containers[0].Image,
			Command: []string{"/bin/bash", "-c"},
			Args:    []string{"/usr/bin/update-ca-trust extract; cp -rf /etc/pki/ca-trust/extracted//* /tmp/new-ca-bundle/"},
			SecurityContext: &corev1.SecurityContext{
				Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				RunAsUser:                &[]int64{0}[0],
				RunAsGroup:               &[]int64{0}[0],
				Privileged:               &[]bool{false}[0],
				AllowPrivilegeEscalation: &[]bool{false}[0],
			},
			TerminationMessagePath:   "/dev/termination-log",
			TerminationMessagePolicy: "File",
			ImagePullPolicy:          "IfNotPresent",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "cabundle-secret",
					MountPath: "/etc/pki/ca-trust/source/anchors/",
					ReadOnly:  true,
				},
				{
					Name:      "cabundle-updated",
					MountPath: "/tmp/new-ca-bundle/",
				},
			},
		},
	}
	deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(deploy.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "cabundle-updated",
			MountPath: "/etc/pki/ca-trust/extracted/",
			ReadOnly:  true,
		})
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: "cabundle-updated",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: "cabundle-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  "rgw-ssl-certificate",
					DefaultMode: &[]int32{420}[0],
					Items: []corev1.KeyToPath{
						{
							Key:  "cabundle",
							Path: "rgw-ssl-certificate.crt",
							Mode: &[]int32{256}[0],
						},
					},
				},
			},
		})
	return deploy
}()

var ToolBoxDeploymentReady = func() *appsv1.Deployment {
	tb := ToolBoxDeploymentBase.DeepCopy()
	tb.Status = appsv1.DeploymentStatus{
		Replicas:          1,
		UpdatedReplicas:   1,
		ReadyReplicas:     1,
		AvailableReplicas: 1,
	}
	return tb
}()
