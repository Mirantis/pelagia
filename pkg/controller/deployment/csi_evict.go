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

package deployment

import (
	"context"
	"strings"
	"time"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	detachCSIVolumesTimeout           = 30 * time.Minute
	verifyRBDVolumesMountsTimeout     = 5 * time.Minute
	verifyRBDVolumeAttachmentsTimeout = 10 * time.Minute
	verifyCSIPodEvictedTimeout        = 5 * time.Minute
	waitForDaemonsetsPodsTimeout      = 5 * time.Minute
	csiPollInterval                   = 15 * time.Second
	waitForDaemonsetsPodsInterval     = 5 * time.Second
)

// TODO: labels are used from kaas maintenance - ?
func (c *cephDeploymentConfig) ensureDaemonsetLabels() {
	c.log.Debug().Msg("run check daemonsets label ensure")
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("ceph daemonsets label ensure failed: failed to list nodes")
		return
	}
	for _, node := range nodes.Items {
		if node.Annotations != nil && node.Annotations[cephDaemonsetDrainRequest] == "true" {
			c.log.Info().Msgf("ceph daemonset ensure: found lcm drain request '%s' for node %s", cephDaemonsetDrainRequest, node.Name)
			if node.Labels[cephDaemonsetLabel] != "" {
				c.log.Info().Msgf("ceph daemonset ensure: found label '%s' to delete", cephDaemonsetLabel)
				err = c.detachCSIVolumes(node.Name)
				if err != nil {
					c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to detach CSI volumes from node %s", node.Name)
					continue
				}
				nodeLabelDelete, err := c.api.Kubeclientset.CoreV1().Nodes().Get(c.context, node.Name, metav1.GetOptions{})
				if err != nil {
					c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to get node %s to delete label", node.Name)
					continue
				}
				delete(nodeLabelDelete.Labels, cephDaemonsetLabel)
				c.log.Info().Msgf("ceph daemonset ensure: removing label '%s' from a node %s", cephDaemonsetLabel, node.Name)
				_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, nodeLabelDelete, metav1.UpdateOptions{})
				if err != nil {
					c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to remove %s label from node %s", cephDaemonsetLabel, node.Name)
					continue
				}
			}
			err = c.verifyCSIPodEvicted(node.Name)
			if err != nil {
				c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: daemonset pods not verified in evict from %s node", node.Name)
				continue
			}
			err = c.waitForDaemonsetsPods(node.Name, false)
			if err != nil {
				c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: daemonset pods not evicted from %s node", node.Name)
				continue
			}
			c.log.Info().Msgf("ceph daemonset ensure: setting annotation '%s' for %s node", cephDaemonsetDrainReady, node.Name)
			nodeAnnotationsCsi, err := c.api.Kubeclientset.CoreV1().Nodes().Get(c.context, node.Name, metav1.GetOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to get node %s", node.Name)
				continue
			}
			if nodeAnnotationsCsi.Annotations[cephDaemonsetDrainReady] != "true" {
				nodeAnnotationsCsi.Annotations[cephDaemonsetDrainReady] = "true"
				_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, nodeAnnotationsCsi, metav1.UpdateOptions{})
				if err != nil {
					c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to add drain ready annotation %s to node %s", cephDaemonsetDrainReady, node.Name)
					continue
				}
			}
			continue
		}
		if node.Labels[cephDaemonsetLabel] != "true" {
			c.log.Info().Msgf("ceph daemonset ensure: label '%s' not found on node '%s'", cephDaemonsetLabel, node.Name)
			nodeForLabel, err := c.api.Kubeclientset.CoreV1().Nodes().Get(c.context, node.Name, metav1.GetOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to get node %s to add label %s", node.Name, cephDaemonsetLabel)
				continue
			}
			nodeForLabel.Labels[cephDaemonsetLabel] = "true"
			c.log.Info().Msgf("adding label '%s' for a node %s", cephDaemonsetLabel, node.Name)
			_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, nodeForLabel, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to add %s label from node %s", cephDaemonsetLabel, node.Name)
				continue
			}
		}
	}
}

func (c *cephDeploymentConfig) detachCSIVolumes(nodeName string) error {
	err := wait.PollUntilContextTimeout(c.context, csiPollInterval, detachCSIVolumesTimeout, true, func(_ context.Context) (bool, error) {
		csiPod, err := c.getCSIPodForNode(nodeName)
		if err != nil {
			c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to get CSI pod for node '%s'", nodeName)
			return false, nil
		}
		if csiPod == nil {
			c.log.Info().Msgf("ceph daemonset ensure: CSI pod not found for node '%s'", nodeName)
			return true, nil
		}
		err = c.verifyRBDVolumesMounts(nodeName, csiPod)
		if err != nil {
			c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to verify hanging rbd volumes mounts on node '%s'", nodeName)
			return false, nil
		}
		err = c.verifyRBDVolumeAttachments(nodeName)
		if err != nil {
			c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to verify rbd volumeattachments for node '%s'", nodeName)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "timeout to detach volumes from node '%s'", nodeName)
	}
	return nil
}

func (c *cephDeploymentConfig) verifyCSIPodEvicted(nodeName string) error {
	csiPod, err := c.getCSIPodForNode(nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to get CSI pod for node '%s'", nodeName)
	}
	if csiPod == nil {
		return nil
	}
	err = wait.PollUntilContextTimeout(c.context, csiPollInterval, verifyCSIPodEvictedTimeout, true, func(_ context.Context) (bool, error) {
		c.log.Info().Msgf("ceph daemonset ensure: Removing existent CSI pod %s/%s...", csiPod.Namespace, csiPod.Name)
		err = c.api.Kubeclientset.CoreV1().Pods(c.lcmConfig.RookNamespace).Delete(context.Background(), csiPod.Name, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			c.log.Info().Msgf("ceph daemonset ensure: CSI pod %s/%s successfully removed", csiPod.Namespace, csiPod.Name)
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		c.log.Error().Err(err).Msgf("ceph daemonset ensure: timeout to delete CSI pod %s/%s", csiPod.Namespace, csiPod.Name)
		return errors.Wrapf(err, "timeout to delete CSI pod %s/%s", csiPod.Namespace, csiPod.Name)
	}
	return nil
}

func (c *cephDeploymentConfig) getCSIPodForNode(nodeName string) (*corev1.Pod, error) {
	pods, err := c.api.Kubeclientset.CoreV1().Pods(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{
		LabelSelector: "app=csi-rbdplugin",
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list pods in %q namespace with app=csi-rbdplugin label", c.lcmConfig.RookNamespace)
	}
	found := false
	csiPod := corev1.Pod{}
	for _, pod := range pods.Items {
		if pod.Spec.NodeName == nodeName {
			c.log.Info().Msgf("ceph daemonset ensure: Pod found '%v' for node '%s'", pod.Name, nodeName)
			csiPod = pod
			found = true
			break
		}
	}
	if !found {
		c.log.Info().Msgf("ceph daemonset ensure: Requested CSI pod from node %s not found", nodeName)
		return nil, nil
	}
	return &csiPod, nil
}

func (c *cephDeploymentConfig) verifyRBDVolumesMounts(nodeName string, csiPod *corev1.Pod) error {
	err := wait.PollUntilContextTimeout(c.context, csiPollInterval, verifyRBDVolumesMountsTimeout, true, func(_ context.Context) (bool, error) {
		c.log.Info().Msgf("ceph daemonset ensure: verify there is no hanging rbd volume mounts on node '%s'", nodeName)
		e := lcmcommon.ExecConfig{
			Context:       c.context,
			Kubeclient:    c.api.Kubeclientset,
			Namespace:     c.lcmConfig.RookNamespace,
			Pod:           csiPod,
			ContainerName: "csi-rbdplugin",
			Command:       "mount",
		}
		stdout, _, err := lcmcommon.RunPodCmdAndCheckError(e)
		if err != nil {
			c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to exec \"mount\" on csi pod %s/%s", c.lcmConfig.RookNamespace, csiPod.Name)
			return false, nil
		}
		mountLines := strings.Split(stdout, "\n")
		foundHanging := false
		for _, line := range mountLines {
			rbdVolume := strings.Split(line, " ")[0]
			if strings.HasPrefix(rbdVolume, "/dev/rbd") {
				c.log.Info().Msgf("ceph daemonset ensure: found hanging rbd volume mount '%s' on node '%s', cleanup", rbdVolume, nodeName)
				foundHanging = true
				e = lcmcommon.ExecConfig{
					Context:       c.context,
					Kubeclient:    c.api.Kubeclientset,
					Namespace:     c.lcmConfig.RookNamespace,
					Pod:           csiPod,
					ContainerName: "csi-rbdplugin",
					Command:       "umount " + rbdVolume,
				}
				_, _, err = lcmcommon.RunPodCmdAndCheckError(e)
				if err != nil {
					c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to exec 'umount %s' on csi pod %s/%s", rbdVolume, c.lcmConfig.RookNamespace, csiPod.Name)
					return false, nil
				}
			}
		}
		if !foundHanging {
			c.log.Info().Msgf("ceph daemonset ensure: there is no hanging rbd volumes mounts on node '%s'", nodeName)
		}
		return !foundHanging, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to wait verifying mounts for rbd volumes on node '%s'", nodeName)
	}
	return nil
}

func (c *cephDeploymentConfig) verifyRBDVolumeAttachments(nodeName string) error {
	c.log.Info().Msgf("ceph daemonset ensure: verifying RBD volumeattachments for node %s", nodeName)
	vas, err := c.api.Kubeclientset.StorageV1().VolumeAttachments().List(c.context, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list volumeattachments for csi-evict")
	}

	vaNames := []string{}
	for _, va := range vas.Items {
		if va.Spec.Attacher == cephVolumeAttachmentType && va.Spec.NodeName == nodeName {
			vaNames = append(vaNames, va.Name)
		}
	}

	c.log.Info().Msgf("ceph daemonset ensure: volumeattachments are found for node %s: %s, removing", nodeName, strings.Join(vaNames, ","))
	err = wait.PollUntilContextTimeout(c.context, csiPollInterval, verifyRBDVolumeAttachmentsTimeout, true, func(_ context.Context) (done bool, err error) {
		vaFailedDelete := []string{}
		for _, va := range vaNames {
			err = c.api.Kubeclientset.StorageV1().VolumeAttachments().Delete(context.TODO(), va, metav1.DeleteOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to delete volumeattachment %s", va)
				vaFailedDelete = append(vaFailedDelete, va)
			}
			c.log.Info().Msgf("ceph daemonset ensure: delete volumeattachment %s in progress: %v", va, err)
			vaFailedDelete = append(vaFailedDelete, va)
		}
		return len(vaFailedDelete) == 0, nil
	})
	if err != nil {
		c.log.Error().Err(err).Msgf("ceph daemonset ensure: volumeattachments removing failed with timeout for node %s", nodeName)
		return errors.Wrapf(err, "volumeattachments removing failed with timeout for node '%s'", nodeName)
	}
	c.log.Info().Msgf("ceph daemonset ensure: volumeattachments removing successfully finished for node %s", nodeName)
	return nil
}

func (c *cephDeploymentConfig) waitForDaemonsetsPods(nodeName string, shouldBeFound bool) error {
	c.log.Info().Msg("ceph daemonset ensure: Wait for daemonset pods consistence")
	err := wait.PollUntilContextTimeout(c.context, waitForDaemonsetsPodsInterval, waitForDaemonsetsPodsTimeout, true, func(_ context.Context) (bool, error) {
		pods, err := c.api.Kubeclientset.CoreV1().Pods(c.lcmConfig.RookNamespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "app=csi-rbdplugin",
		})
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to list pods in %q namespace with app=csi-rbdplugin label", c.lcmConfig.RookNamespace)
			return false, nil
		}
		found := false
		for _, pod := range pods.Items {
			if pod.Spec.NodeName == nodeName {
				found = true
				break
			}
		}
		return shouldBeFound == found, nil
	})
	if err != nil {
		return errors.Wrapf(err, "timeout to wait daemonset pods in %q namespace updated", c.lcmConfig.RookNamespace)
	}
	return nil
}
