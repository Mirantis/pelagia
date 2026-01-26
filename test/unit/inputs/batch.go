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
	"fmt"
	"sort"
	"strings"

	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetCleanupJobOnlyStatus(jobName, namespace string, active, failed, succeeded int32) *batch.Job {
	return &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Status: batch.JobStatus{
			Active:    active,
			Failed:    failed,
			Succeeded: succeeded,
		},
	}
}

func GetCleanupJob(host, osd, longName string, devices map[string]string) *batch.Job {
	jobName := fmt.Sprintf("device-cleanup-job-%s-%s", host, osd)
	osdLabel := strings.ReplaceAll(osd, "__", "")
	if longName != "" {
		jobName = longName
	}
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: LcmObjectMeta.Namespace,
			Labels: map[string]string{
				"app":          "pelagia-lcm-cleanup-disks",
				"rook-cluster": LcmObjectMeta.Name,
				"host":         host,
				"osd":          osdLabel,
				"task":         "osdremove-task",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "lcm.mirantis.com/v1alpha1",
					Kind:       "CephOsdRemoveTask",
					Name:       "osdremove-task",
				},
			},
		},
		Spec: batch.JobSpec{
			BackoffLimit:          &[]int32{0}[0],
			ActiveDeadlineSeconds: &[]int64{3600}[0],
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: "pelagia-lcm-cleanup-disks"},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{RunAsUser: &[]int64{0}[0]},
					NodeSelector:    map[string]string{corev1.LabelHostname: host},
					Containers: func() []corev1.Container {
						containers := []corev1.Container{}
						idx := 1
						deviceList := []string{}
						for dev := range devices {
							deviceList = append(deviceList, dev)
						}
						// to avoid misordering since map is not ordered
						sort.Strings(deviceList)
						for _, dev := range deviceList {
							script := devices[dev]
							container := corev1.Container{
								Name:  fmt.Sprintf("cleanup-run-%d", idx),
								Image: cephClusterImage,
								SecurityContext: &corev1.SecurityContext{
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"NET_RAW"},
									},
									RunAsUser:  &[]int64{0}[0],
									Privileged: &[]bool{true}[0],
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "host-dev",
										MountPath: "/dev",
									},
									{
										Name:      "run-udev",
										MountPath: "/run/udev",
										ReadOnly:  true,
									},
									{
										Name:      "host-rook",
										MountPath: "/var/lib/rook",
									},
								},
								Command: []string{"/bin/bash", "-c", script},
								Env: []corev1.EnvVar{
									{
										Name:  "DEVICE_NAME",
										Value: "/dev/" + dev,
									},
									{
										Name:  "DM_DISABLE_UDEV",
										Value: "1",
									},
								},
							}
							containers = append(containers, container)
							idx++
						}
						return containers
					}(),
					Volumes: []corev1.Volume{
						{
							Name: "host-dev",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/dev",
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
						{
							Name: "host-rook",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/rook",
								},
							},
						},
					},
					// it allows to keep failed pods and do not remove them
					// since we dont need restart it allows to inspect failed container logs
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	for dev := range devices {
		job.Labels[dev] = "true"
	}
	return job
}
