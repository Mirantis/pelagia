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
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/v2/pkg/common"
)

func (c *cephDeploymentConfig) ensureNodesAnnotation() (bool, error) {
	c.log.Debug().Msg("ensure nodes annotations with rook keys")
	errCollector := 0
	changedNodes := false

	nodeMonitorIPs := map[string]string{}
	for _, node := range c.cdConfig.nodesListExpanded {
		if node.MonitorIP != "" {
			nodeMonitorIPs[node.Name] = node.MonitorIP
		}
	}
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	for _, node := range nodes.Items {
		annotations := map[string]string{}
		if ip, ok := nodeMonitorIPs[node.Name]; ok {
			annotations[monIPAnnotation] = ip
		}
		changed, err := c.annotateNode(annotations, node)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			errCollector++
		}
		changedNodes = changedNodes || changed
	}
	if errCollector > 0 {
		return false, errors.New("failed to verify node(s) annotations")
	}
	return changedNodes, nil
}

func (c *cephDeploymentConfig) ensureLabelNodes() (bool, error) {
	c.log.Debug().Msg("ensure nodes labels for ceph roles and topology")
	nodesRoles := map[string][]string{}
	nodesCrushTopology := map[string]map[string]string{}
	for _, node := range c.cdConfig.nodesListExpanded {
		roles := node.Roles
		if lcmcommon.IsCephOsdNode(node.Node) {
			roles = append(roles, "osd")
			nodesCrushTopology[node.Name] = node.Crush
		}
		nodesRoles[node.Name] = roles
	}
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	errCollector := 0
	labelsUpdated := false
	for _, node := range nodes.Items {
		nodeRoles, nodeInSpec := nodesRoles[node.Name]
		nodeTopology, storageNodeInSpec := nodesCrushTopology[node.Name]
		// check all nodes with osd role label does it have or not any osd deployment right now
		// this will help to determine keep or not keep osd role label if node is not specified
		// in spec or specified w/o devices, but may continue running osd pods (even if they are in crashed)
		// ceph osd label allows running ceph disk daemon, so if some pods are running, but node not in spec
		// means it is getting removed and we need to keep label until removed
		if _, osdLabelPresent := node.Labels[fmt.Sprintf(lcmcommon.CephNodeLabelTemplate, "osd")]; osdLabelPresent {
			if !storageNodeInSpec {
				labelSelector := fmt.Sprintf(nodeWithOSDSelectorTemplate, node.Name)
				osdDeployments, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{LabelSelector: labelSelector})
				if err != nil {
					c.log.Error().Err(err).Msgf("failed to check node '%s' for present osd deployments", node.Name)
					errCollector++
					continue
				}
				if len(osdDeployments.Items) > 0 {
					if nodeInSpec {
						nodeRoles = append(nodeRoles, "osd")
					} else {
						nodeRoles = []string{"osd"}
					}
				}
			}
		}
		newLabels, updated := buildNodeLabels(node.Labels, nodeRoles, nodeTopology)
		if updated {
			c.log.Info().Msgf("update node '%s' labels", node.Name)
			lcmcommon.ShowObjectDiff(*c.log, node.Labels, newLabels)
			node.Labels = newLabels
			_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, &node, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to update '%s' node labels", node.Name)
				errCollector++
			}
			labelsUpdated = true
		}
	}
	if errCollector > 0 {
		return false, errors.New("failed to verify node(s) labels")
	}
	return labelsUpdated, nil
}

func (c *cephDeploymentConfig) deleteLabelNodes() (bool, error) {
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	errCollector := 0
	changed := false
	for _, node := range nodes.Items {
		if newLabels, updated := buildNodeLabels(node.Labels, nil, nil); updated {
			c.log.Info().Msgf("removing ceph related labels from node '%s'", node.Name)
			lcmcommon.ShowObjectDiff(*c.log, node.Labels, newLabels)
			node.Labels = newLabels
			_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, &node, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to update '%s' node labels", node.Name)
				errCollector++
			}
			changed = true
		}
	}
	if errCollector > 0 {
		return false, errors.New("failed to delete ceph role or crush topology labels from obsolete node(s)")
	}
	return !changed, nil
}

func (c *cephDeploymentConfig) deleteNodesAnnotations() (bool, error) {
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	errCollector := 0
	changed := false
	for _, node := range nodes.Items {
		updated, err := c.annotateNode(map[string]string{}, node)
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to cleanup node '%s' from redundant annotations", node.Name)
			errCollector++
		}
		changed = changed || updated
	}
	if errCollector > 0 {
		return false, errors.New("failed to delete rook annotations from obsolete node(s)")
	}
	return !changed, nil
}

func (c *cephDeploymentConfig) deleteDaemonSetLabels() (bool, error) {
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	errCollector := 0
	noLabels := true
	for _, node := range nodes.Items {
		if _, ok := node.Labels[cephDaemonsetLabel]; ok {
			noLabels = false
			c.log.Info().Msgf("remove cephdeployment label '%s' from node '%s'", cephDaemonsetLabel, node.Name)
			delete(node.Labels, cephDaemonsetLabel)
			_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, &node, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to remove '%s' node label %s", node.Name, cephDaemonsetLabel)
				errCollector++
			}
		}
	}
	if errCollector > 0 {
		return false, errors.New("failed to delete daemonset labels from some nodes")
	}
	return noLabels, nil
}

func (c *cephDeploymentConfig) annotateNode(annotations map[string]string, node v1.Node) (bool, error) {
	newAnnotations, updateAnnotations := buildCephNodeAnnotations(node.Annotations, annotations)
	if updateAnnotations {
		c.log.Info().Msgf("update node '%s' annotations", node.Name)
		lcmcommon.ShowObjectDiff(*c.log, node.Annotations, newAnnotations)
		node.Annotations = newAnnotations
		if _, err := c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, &node, metav1.UpdateOptions{}); err != nil {
			return false, errors.Wrapf(err, "failed to update '%s' node annotations", node.Name)
		}
	}
	return updateAnnotations, nil
}

func buildNodeLabels(currentLables map[string]string, nodeRoles []string, newNodeTopology map[string]string) (map[string]string, bool) {
	newLabels, updatedLabels := lcmcommon.BuildCephNodeLabels(currentLables, nodeRoles)
	for crushroot, crushtopology := range newNodeTopology {
		topology := crushTopologyAllowedKeys[crushroot]
		currentTopology, present := newLabels[topology]
		// for AWS topology.kubernetes.io labels may be used w/o ceph
		// so need to previous value to have an ability to restore it
		// in case of dropping node from ceph cluster
		if !present {
			if isKubeCrush(crushroot) {
				newLabels[fmt.Sprintf(cephKubeTopologyLabelTemplate, topology)] = ""
			}
			newLabels[topology] = crushtopology
			updatedLabels = true
		} else {
			if isKubeCrush(crushroot) {
				_, ok := newLabels[fmt.Sprintf(cephKubeTopologyLabelTemplate, topology)]
				// if we dont have prev topology, topology set initially not by our controller, so keep it
				if !ok {
					newLabels[fmt.Sprintf(cephKubeTopologyLabelTemplate, topology)] = currentTopology
					updatedLabels = true
				}
			}
			if currentTopology != crushtopology {
				newLabels[topology] = crushtopology
				updatedLabels = true
			}
		}
	}
	// remove obsolete crush topology labels
	for allowedKey, topologyPath := range crushTopologyAllowedKeys {
		if _, ok := newNodeTopology[allowedKey]; ok {
			continue
		}
		if _, ok := newLabels[topologyPath]; ok {
			if isKubeCrush(allowedKey) {
				prevKey := fmt.Sprintf(cephKubeTopologyLabelTemplate, topologyPath)
				originalValue, exist := newLabels[prevKey]
				if !exist {
					// if no prev value - means controller not set it ever, keep it
					continue
				}
				// if prev value is empty - it is only ceph topology, otherwise set it back
				if originalValue == "" {
					delete(newLabels, topologyPath)
				} else {
					newLabels[topologyPath] = originalValue
				}
				delete(newLabels, prevKey)
			} else {
				delete(newLabels, topologyPath)
			}
			updatedLabels = true
		}
	}
	return newLabels, updatedLabels
}
