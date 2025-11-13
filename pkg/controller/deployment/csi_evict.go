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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
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
				_, err = c.checkCSIVolumes(node.Name, true)
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

		updateNode := func(nodeName string, setLabel bool) {
			nodeForLabel, err := c.api.Kubeclientset.CoreV1().Nodes().Get(c.context, nodeName, metav1.GetOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to get node %s to add label %s", nodeName, cephDaemonsetLabel)
				return
			}
			if setLabel {
				c.log.Info().Msgf("adding label '%s' for a node %s", cephDaemonsetLabel, nodeName)
				nodeForLabel.Labels[cephDaemonsetLabel] = "true"
			} else {
				c.log.Info().Msgf("dropping label '%s' from a node %s", cephDaemonsetLabel, nodeName)
				delete(nodeForLabel.Labels, cephDaemonsetLabel)
			}
			_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, nodeForLabel, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to add %s label from node %s", cephDaemonsetLabel, nodeName)
			}
		}

		if c.lcmConfig.DeployParams.CephDaemonsetPlacementLabelExclude != "" {
			// since we are validating it in config - will be ok for sure
			selector, _ := labels.Parse(c.lcmConfig.DeployParams.CephDaemonsetPlacementLabelExclude)
			if selector.Matches(labels.Set(node.GetLabels())) {
				if _, ok := node.Labels[cephDaemonsetLabel]; ok {
					c.log.Info().Msgf("ceph daemonset ensure: label '%s' is found on node '%s', which has excluding label(s) '%s', trying to remove...",
						cephDaemonsetLabel, node.Name, c.lcmConfig.DeployParams.CephDaemonsetPlacementLabelExclude)
					found, err := c.checkCSIVolumes(node.Name, false)
					if err != nil {
						c.log.Error().Err(err).Msgf("ceph daemonsets label ensure failed: failed to check CSI volumes/volumeattachments for node '%s', can't remove '%s' label", node.Name, cephDaemonsetLabel)
					} else if found {
						c.log.Info().Msgf("ceph daemonsets label ensure: found volumes/volumeattachments mounts for a node '%s', can't remove '%s' label", node.Name, cephDaemonsetLabel)
					} else {
						updateNode(node.Name, false)
					}
				} else {
					c.log.Debug().Msgf("ceph daemonset ensure: node '%s' has excluding label(s) '%s', skip adding '%s' label",
						node.Name, c.lcmConfig.DeployParams.CephDaemonsetPlacementLabelExclude, cephDaemonsetLabel)
				}
				continue
			}
		}
		if node.Labels[cephDaemonsetLabel] != "true" {
			updateNode(node.Name, true)
		}
	}
}

func (c *cephDeploymentConfig) checkCSIVolumes(nodeName string, umountFound bool) (bool, error) {
	foundMounted := false
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
		foundVolumes, err := c.verifyRBDVolumesMounts(nodeName, csiPod, umountFound)
		if err != nil {
			c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to verify rbd volumes mounts on node '%s'", nodeName)
			return false, nil
		}
		foundAttachments, err := c.verifyRBDVolumeAttachments(nodeName, umountFound)
		if err != nil {
			c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to verify rbd volumeattachments for node '%s'", nodeName)
			return false, nil
		}
		foundMounted = foundVolumes || foundAttachments
		return true, nil
	})
	if err != nil {
		return false, errors.Wrapf(err, "timeout to check volumes for node '%s'", nodeName)
	}
	return foundMounted, nil
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
		err = c.api.Kubeclientset.CoreV1().Pods(c.lcmConfig.RookNamespace).Delete(c.context, csiPod.Name, metav1.DeleteOptions{})
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

func (c *cephDeploymentConfig) verifyRBDVolumesMounts(nodeName string, csiPod *corev1.Pod, umountFound bool) (bool, error) {
	mountsPresent := false
	err := wait.PollUntilContextTimeout(c.context, csiPollInterval, verifyRBDVolumesMountsTimeout, true, func(_ context.Context) (bool, error) {
		c.log.Info().Msgf("ceph daemonset ensure: verify rbd volume mounts on node '%s'", nodeName)
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
		foundMount := false
		for _, line := range mountLines {
			rbdVolume := strings.Split(line, " ")[0]
			if strings.HasPrefix(rbdVolume, "/dev/rbd") {
				foundMount = true
				if umountFound {
					c.log.Info().Msgf("ceph daemonset ensure: found rbd volume mount '%s' on node '%s', cleanup", rbdVolume, nodeName)
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
					}
				} else {
					c.log.Info().Msgf("ceph daemonset ensure: found rbd volume mount '%s' on node '%s', cleanup related resources manually", rbdVolume, nodeName)
				}
			}
		}
		if umountFound && foundMount {
			return false, nil
		}
		if !foundMount {
			c.log.Info().Msgf("ceph daemonset ensure: no rbd volumes mounts found on a node '%s'", nodeName)
		}
		mountsPresent = foundMount
		return true, nil
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to wait verifying mounts for rbd volumes on node '%s'", nodeName)
	}
	return mountsPresent, nil
}

func (c *cephDeploymentConfig) verifyRBDVolumeAttachments(nodeName string, umountFound bool) (bool, error) {
	c.log.Info().Msgf("ceph daemonset ensure: verifying RBD volumeattachments for node %s", nodeName)
	volumeAttachmentsFound := false
	err := wait.PollUntilContextTimeout(c.context, csiPollInterval, verifyRBDVolumeAttachmentsTimeout, true, func(_ context.Context) (done bool, err error) {
		vas, err := c.api.Kubeclientset.StorageV1().VolumeAttachments().List(c.context, metav1.ListOptions{})
		if err != nil {
			c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to list volumeattachments for csi-evict")
			return false, nil
		}
		found := false
		for _, va := range vas.Items {
			if va.Spec.Attacher == cephVolumeAttachmentType && va.Spec.NodeName == nodeName {
				found = true
				if umountFound {
					c.log.Info().Msgf("ceph daemonset ensure: volumeattachment found for node %s: %s, removing", nodeName, va.Name)
					err = c.api.Kubeclientset.StorageV1().VolumeAttachments().Delete(c.context, va.Name, metav1.DeleteOptions{})
					if err != nil {
						c.log.Error().Err(err).Msgf("ceph daemonset ensure: failed to delete volumeattachment %s", va.Name)
					}
				} else {
					c.log.Info().Msgf("ceph daemonset ensure: volumeattachment found for node %s: %s, remove it manually", nodeName, va.Name)
				}
			}
		}
		if umountFound && found {
			return false, nil
		}
		if !found {
			c.log.Info().Msgf("ceph daemonset ensure: volumeattachment are not found for node %s", nodeName)
		}
		volumeAttachmentsFound = found
		return true, nil
	})
	if err != nil {
		return false, errors.Wrapf(err, "timeout to check volumeattachments for node '%s'", nodeName)
	}
	return volumeAttachmentsFound, nil
}

func (c *cephDeploymentConfig) waitForDaemonsetsPods(nodeName string, shouldBeFound bool) error {
	c.log.Info().Msg("ceph daemonset ensure: Wait for daemonset pods consistence")
	err := wait.PollUntilContextTimeout(c.context, waitForDaemonsetsPodsInterval, waitForDaemonsetsPodsTimeout, true, func(_ context.Context) (bool, error) {
		pods, err := c.api.Kubeclientset.CoreV1().Pods(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{
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
