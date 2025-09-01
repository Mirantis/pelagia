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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentInfraConfig) checkLabelsAndOwnerRefs(cur *metav1.ObjectMeta, expected *metav1.ObjectMeta) bool {
	update := false
	if !reflect.DeepEqual(cur.Labels, expected.Labels) {
		if cur.Labels == nil {
			cur.Labels = expected.Labels
			c.log.Debug().Msg("updating labels")
		} else {
			for k, v := range expected.Labels {
				if _, ok := cur.Labels[k]; !ok {
					update = true
					cur.Labels[k] = v
					c.log.Debug().Msgf("update label '%s=%s'", k, v)
				}
			}
		}
	}
	if !reflect.DeepEqual(cur.OwnerReferences, expected.OwnerReferences) {
		update = true
		c.log.Debug().Msg("updating ownerReferences")
		cur.OwnerReferences = expected.OwnerReferences
	}
	return update
}

func (c *cephDeploymentInfraConfig) ensureDiskDaemon() error {
	if c.infraConfig.externalCeph {
		return nil
	}
	if c.infraConfig.cephImage == "" {
		c.log.Error().Msgf("related CephCluster has no image provided in status yet, skipping %s reconcile", lcmcommon.PelagiaDiskDaemon)
		return nil
	}
	diskDaemonNew := c.generateDiskDaemon()
	diskDaemonCur, err := c.api.Kubeclientset.AppsV1().DaemonSets(c.infraConfig.namespace).Get(c.context, lcmcommon.PelagiaDiskDaemon, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.log.Info().Msgf("create disk daemon daemonset '%s/%s'", diskDaemonNew.Namespace, diskDaemonNew.Name)
			_, err = c.api.Kubeclientset.AppsV1().DaemonSets(c.infraConfig.namespace).Create(c.context, diskDaemonNew, metav1.CreateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msg("")
				return errors.Wrapf(err, "failed to create disk-daemon daemonset '%s/%s'", diskDaemonNew.Namespace, diskDaemonNew.Name)
			}
			return nil
		}
		c.log.Error().Err(err).Msg("")
		return errors.Wrapf(err, "failed to check disk-daemon daemonset '%s/%s'", c.infraConfig.namespace, lcmcommon.PelagiaDiskDaemon)
	}
	// we can't predict current default scheduler name - so just take it from present deployment
	diskDaemonNew.Spec.Template.Spec.SchedulerName = diskDaemonCur.Spec.Template.Spec.SchedulerName
	if !reflect.DeepEqual(diskDaemonCur.Spec, diskDaemonNew.Spec) || c.checkLabelsAndOwnerRefs(&diskDaemonCur.ObjectMeta, &diskDaemonNew.ObjectMeta) {
		c.log.Info().Msgf("update disk daemon daemonset '%s/%s'", diskDaemonCur.Namespace, diskDaemonCur.Name)
		lcmcommon.ShowObjectDiff(*c.log, diskDaemonCur.Spec, diskDaemonNew.Spec)
		diskDaemonCur.Spec = diskDaemonNew.Spec
		_, err = c.api.Kubeclientset.AppsV1().DaemonSets(c.infraConfig.namespace).Update(c.context, diskDaemonCur, metav1.UpdateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return errors.Wrapf(err, "failed to update disk-daemon daemonset '%s/%s'", diskDaemonCur.Namespace, diskDaemonCur.Name)
		}
		return nil
	}
	if !lcmcommon.IsDaemonSetReady(diskDaemonCur) {
		msg := fmt.Sprintf("desired: %d, ready: %d, updated: %d",
			diskDaemonCur.Status.DesiredNumberScheduled, diskDaemonCur.Status.NumberReady, diskDaemonCur.Status.UpdatedNumberScheduled)
		c.log.Warn().Msgf("disk-daemon daemonset '%s/%s' is not ready yet (%s)", diskDaemonCur.Namespace, diskDaemonCur.Name, msg)
	}
	return nil
}

func (c *cephDeploymentInfraConfig) generateDiskDaemon() *apps.DaemonSet {
	// init vars with pathes
	diskDaemonGoPathMountName := "pelagia-disk-daemon-bin"
	diskDaemonGoPath := "/usr/local/bin"
	initContainerDiskDaemonGoPath := "/tmp/bin"
	diskDaemonDevPathMountName := "devices"
	diskDaemonDevPath := "/dev"
	diskDaemonUdevPathMountName := "run-udev"
	diskDaemonUdevPath := "/run/udev"
	hostPathSourceType := v1.HostPathDirectory
	// we dont need to much time to terminate - stop api server, stop go threads
	// nothing that can take to much time or affect on other services
	terminationPeriod := int64(10)
	nodeSelector, _ := labels.ConvertSelectorToLabelsMap(c.lcmConfig.DiskDaemonPlacementLabel)

	diskDaemon := &apps.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            lcmcommon.PelagiaDiskDaemon,
			Namespace:       c.infraConfig.namespace,
			Labels:          map[string]string{"app": lcmcommon.PelagiaDiskDaemon},
			OwnerReferences: c.infraConfig.lcmOwnerRefs,
		},
		Spec: apps.DaemonSetSpec{
			// we dont need to much time to realize container is up and running
			// since it will crush on any error, so just 5 secs is enough to init base thread
			MinReadySeconds:      5,
			RevisionHistoryLimit: &revisionHistoryLimit,
			UpdateStrategy: apps.DaemonSetUpdateStrategy{
				Type: apps.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &apps.RollingUpdateDaemonSet{
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
				MatchLabels: map[string]string{"app": lcmcommon.PelagiaDiskDaemon},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": lcmcommon.PelagiaDiskDaemon},
				},
				Spec: v1.PodSpec{
					DNSPolicy: "ClusterFirstWithHostNet",
					SecurityContext: &v1.PodSecurityContext{
						RunAsUser:  &rootUserID,
						RunAsGroup: &rootUserID,
					},
					RestartPolicy:                 v1.RestartPolicyAlways,
					TerminationGracePeriodSeconds: &terminationPeriod,
					InitContainers: []v1.Container{
						{
							Name:                     "bin-downloader",
							Image:                    c.infraConfig.controllerImage,
							Command:                  []string{"cp"},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: "File",
							Args: []string{
								fmt.Sprintf("%s/%s", diskDaemonGoPath, lcmcommon.PelagiaDiskDaemon),
								fmt.Sprintf("%s/tini", diskDaemonGoPath),
								fmt.Sprintf("%s/", initContainerDiskDaemonGoPath),
							},
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &v1.SecurityContext{Capabilities: &v1.Capabilities{Drop: []v1.Capability{"ALL"}}},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      diskDaemonGoPathMountName,
									MountPath: initContainerDiskDaemonGoPath,
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:                     lcmcommon.PelagiaDiskDaemon,
							Image:                    c.infraConfig.cephImage,
							Command:                  []string{fmt.Sprintf("%s/tini", diskDaemonGoPath), "--"},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: "File",
							Args: []string{
								fmt.Sprintf("%s/%s", diskDaemonGoPath, lcmcommon.PelagiaDiskDaemon),
								"--daemon",
								"--port",
								fmt.Sprintf("%d", c.lcmConfig.DiskDaemonPort),
							},
							ImagePullPolicy: "IfNotPresent",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      diskDaemonGoPathMountName,
									MountPath: diskDaemonGoPath,
								},
								{
									Name:      diskDaemonDevPathMountName,
									MountPath: diskDaemonDevPath,
									ReadOnly:  true,
								},
								{
									Name:      diskDaemonUdevPathMountName,
									MountPath: diskDaemonUdevPath,
									ReadOnly:  true,
								},
							},
							Env: []v1.EnvVar{
								{
									// do not change value, since it is affecting udev-lvm sync
									// https://listman.redhat.com/archives/lvm-devel/2012-November/msg00069.html
									Name:  "DM_DISABLE_UDEV",
									Value: "0",
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged:   &trueVar,
								RunAsUser:    &rootUserID,
								Capabilities: &v1.Capabilities{Drop: []v1.Capability{"ALL"}},
							},
							LivenessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{
									Exec: &v1.ExecAction{
										Command: []string{
											fmt.Sprintf("%s/%s", diskDaemonGoPath, lcmcommon.PelagiaDiskDaemon),
											"--api-check",
											"--port",
											fmt.Sprintf("%d", c.lcmConfig.DiskDaemonPort),
										},
									},
								},
								TimeoutSeconds:      1,
								SuccessThreshold:    1,
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
								FailureThreshold:    3,
							},
							ReadinessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{
									Exec: &v1.ExecAction{
										Command: []string{
											fmt.Sprintf("%s/%s", diskDaemonGoPath, lcmcommon.PelagiaDiskDaemon),
											"--api-check",
											"--port",
											fmt.Sprintf("%d", c.lcmConfig.DiskDaemonPort),
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
					NodeSelector: nodeSelector,
					Volumes: []v1.Volume{
						{
							Name:         diskDaemonGoPathMountName,
							VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
						},
						{
							Name: diskDaemonDevPathMountName,
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: diskDaemonDevPath,
									Type: &hostPathSourceType,
								},
							},
						},
						{
							Name: diskDaemonUdevPathMountName,
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: diskDaemonUdevPath,
									Type: &hostPathSourceType,
								},
							},
						},
					},
				},
			},
		},
	}
	if len(c.infraConfig.osdPlacement.Tolerations) > 0 {
		diskDaemon.Spec.Template.Spec.Tolerations = c.infraConfig.osdPlacement.Tolerations
	}
	return diskDaemon
}
