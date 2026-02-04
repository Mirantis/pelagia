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
	"k8s.io/apimachinery/pkg/util/intstr"
)

var DaemonSetListEmpty = &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{}}

var DaemonSetListReady = &appsv1.DaemonSetList{
	Items: []appsv1.DaemonSet{
		*DaemonSetWithStatus(RookNamespace, "rook-ceph.rbd.csi.ceph.com-nodeplugin", 3, 3), *DaemonSetWithStatus(RookNamespace, "rook-ceph.cephfs.csi.ceph.com-nodeplugin", 3, 3),
		*DaemonSetWithStatus(LcmObjectMeta.Namespace, "pelagia-disk-daemon", 2, 2),
	},
}

var DaemonSetListNotReady = &appsv1.DaemonSetList{
	Items: []appsv1.DaemonSet{
		*DaemonSetWithStatus(RookNamespace, "rook-ceph.rbd.csi.ceph.com-nodeplugin", 3, 1), *DaemonSetWithStatus(RookNamespace, "rook-ceph.cephfs.csi.ceph.com-nodeplugin", 3, 1),
		*DaemonSetWithStatus(LcmObjectMeta.Namespace, "pelagia-disk-daemon", 2, 0),
	},
}

func DaemonSetWithStatus(namespace, name string, desiredReplicas, readyReplicas int32) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: appsv1.DaemonSetStatus{
			NumberReady:            readyReplicas,
			CurrentNumberScheduled: desiredReplicas,
			NumberAvailable:        readyReplicas,
			UpdatedNumberScheduled: desiredReplicas,
			DesiredNumberScheduled: desiredReplicas,
		},
	}
}

var DiskDaemonDaemonset = appsv1.DaemonSet{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pelagia-disk-daemon",
		Namespace: LcmObjectMeta.Namespace,
		Labels: map[string]string{
			"app": "pelagia-disk-daemon",
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: "lcm.mirantis.com/v1alpha1",
				Kind:       "CephDeploymentHealth",
				Name:       LcmObjectMeta.Name,
			},
		},
	},
	Spec: appsv1.DaemonSetSpec{
		MinReadySeconds:      5,
		RevisionHistoryLimit: &[]int32{5}[0],
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &intstr.IntOrString{
					Type:   1,
					StrVal: "30%",
				},
				MaxSurge: &intstr.IntOrString{
					Type:   0,
					IntVal: 0,
				},
			},
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "pelagia-disk-daemon",
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "pelagia-disk-daemon",
				},
			},
			Spec: corev1.PodSpec{
				DNSPolicy: "ClusterFirstWithHostNet",
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser:  &[]int64{0}[0],
					RunAsGroup: &[]int64{0}[0],
				},
				RestartPolicy:                 corev1.RestartPolicyAlways,
				TerminationGracePeriodSeconds: &[]int64{10}[0],
				InitContainers: []corev1.Container{
					{
						Name:  "bin-downloader",
						Image: "some-registry/lcm-controller:v1",
						Command: []string{
							"cp",
						},
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: "File",
						Args: []string{
							"/usr/local/bin/pelagia-disk-daemon",
							"/usr/local/bin/tini",
							"/tmp/bin/",
						},
						ImagePullPolicy: "IfNotPresent",
						SecurityContext: &corev1.SecurityContext{
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "pelagia-disk-daemon-bin",
								MountPath: "/tmp/bin",
							},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:  "pelagia-disk-daemon",
						Image: cephClusterImage,
						Command: []string{
							"/usr/local/bin/tini", "--",
						},
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: "File",
						Args: []string{
							"/usr/local/bin/pelagia-disk-daemon", "--daemon", "--port", "9999",
						},
						ImagePullPolicy: "IfNotPresent",
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "pelagia-disk-daemon-bin",
								MountPath: "/usr/local/bin",
							},
							{
								Name:      "devices",
								MountPath: "/dev",
								ReadOnly:  true,
							},
							{
								Name:      "run-udev",
								MountPath: "/run/udev",
								ReadOnly:  true,
							},
						},
						Env: []corev1.EnvVar{
							{
								Name:  "DM_DISABLE_UDEV",
								Value: "0",
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &[]bool{true}[0],
							RunAsUser:  &[]int64{0}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"/usr/local/bin/pelagia-disk-daemon",
										"--api-check", "--port", "9999",
									},
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       10,
							FailureThreshold:    3,
							TimeoutSeconds:      1,
							SuccessThreshold:    1,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"/usr/local/bin/pelagia-disk-daemon",
										"--api-check", "--port", "9999",
									},
								},
							},
							PeriodSeconds:    10,
							FailureThreshold: 3,
							TimeoutSeconds:   1,
							SuccessThreshold: 1,
						},
					},
				},
				NodeSelector: map[string]string{"pelagia-disk-daemon": "true"},
				Volumes: []corev1.Volume{
					{
						Name: "pelagia-disk-daemon-bin",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "devices",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/dev",
								Type: &[]corev1.HostPathType{corev1.HostPathDirectory}[0],
							},
						},
					},
					{
						Name: "run-udev",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/run/udev",
								Type: &[]corev1.HostPathType{corev1.HostPathDirectory}[0],
							},
						},
					},
				},
			},
		},
	},
}

var DiskDaemonDaemonsetWithOsdTolerations = func() *appsv1.DaemonSet {
	ds := DiskDaemonDaemonset.DeepCopy()
	ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
		{
			Key:      "test.kubernetes.io/testkey",
			Effect:   "Schedule",
			Operator: "Exists",
		},
	}
	return ds
}()

var RookDiscover = appsv1.DaemonSet{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rook-discover",
	},
	Spec: appsv1.DaemonSetSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"],
					},
				},
			},
		},
	},
	Status: appsv1.DaemonSetStatus{
		NumberReady:            1,
		CurrentNumberScheduled: 1,
		DesiredNumberScheduled: 1,
		NumberAvailable:        1,
		UpdatedNumberScheduled: 1,
	},
}
