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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
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
	excludeNodes := []string{}
	for nodeName, ip := range nodeMonitorIPs {
		changed, err := c.annotateNodes(map[string]string{monIPAnnotation: ip}, nodeName)
		if err != nil {
			c.log.Error().Err(err).Msg("failed to annotate node with monitor ip address")
			errCollector++
		}
		changedNodes = changedNodes || changed
		excludeNodes = append(excludeNodes, nodeName)
	}
	if errCollector > 0 {
		return false, errors.New("failed to set rook annotations for some node(s)")
	}
	// Remove annotations from other obsolete nodes
	noChanges, err := c.deleteNodesAnnotations(excludeNodes...)
	if err != nil {
		return false, err
	}
	changedNodes = changedNodes || !noChanges
	return changedNodes, nil
}

func (c *cephDeploymentConfig) ensureLabelNodes() (bool, error) {
	c.log.Debug().Msg("ensure nodes labels (ceph roles) and topology")
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	errCollector := 0
	nodeRoles := map[string][]string{}
	osdDeploymentExists := map[string]bool{}
	// check all nodes with osd role label does it have or not any osd deployment right now
	// this will help to determine keep or not keep osd role label if node is not specified
	// in spec, but may continue running osd pods (even if they are in crashed)
	for _, node := range nodes.Items {
		if _, osdLabelPresent := node.Labels[fmt.Sprintf(cephNodeLabelTemplate, "osd")]; osdLabelPresent {
			labelSelector := fmt.Sprintf(nodeWithOSDSelectorTemplate, node.Name)
			osdDeployments, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{LabelSelector: labelSelector})
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to check node '%s' for present osd deployments", node.Name)
				errCollector++
				continue
			}
			if len(osdDeployments.Items) > 0 {
				osdDeploymentExists[node.Name] = true
				// put role right now, in case if current node is not in spec
				nodeRoles[node.Name] = []string{"osd"}
			}
		}
	}
	if errCollector > 0 {
		return false, errors.New("failed to check osd deployments for some node(s) with osd role")
	}
	changedNodes := false
	for _, node := range c.cdConfig.nodesListExpanded {
		roles := node.Roles
		// if node has storage configuration - it may have crush topology as well
		if isCephOsdNode(node.Node) {
			roles = append(roles, "osd")
			changed, err := c.addTopology(node.Name, node.Crush)
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to set crush topology labels for node %q", node.Name)
				errCollector++
			}
			changedNodes = changedNodes || changed
		} else if osdDeploymentExists[node.Name] {
			roles = append(roles, "osd")
		}
		nodeRoles[node.Name] = roles
	}
	excludeNodes := []string{}
	for nodeName, roles := range nodeRoles {
		changed, err := c.labelNodes(roles, nodeName)
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to set role labels for node %q", nodeName)
			errCollector++
		}
		changedNodes = changedNodes || changed
		excludeNodes = append(excludeNodes, nodeName)
	}
	if errCollector > 0 {
		return false, errors.New("failed to set role or crush topology labels for some node(s)")
	}
	// Remove roles from other obsolete nodes
	noChanges, err := c.deleteLabelNodes(excludeNodes...)
	if err != nil {
		return false, err
	}
	changedNodes = changedNodes && noChanges
	return changedNodes, nil
}

func (c *cephDeploymentConfig) deleteLabelNodes(excludeNode ...string) (bool, error) {
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	errCollector := 0
	changed := false
	for _, node := range nodes.Items {
		if lcmcommon.Contains(excludeNode, node.Name) {
			continue
		}
		updated, err := c.labelNodes([]string{}, node.Name)
		changed = changed || updated
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to unlabel node '%s' from needless ceph role labels", node.Name)
			errCollector++
		}
		updated, err = c.addTopology(node.Name, map[string]string{})
		changed = changed || updated
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to unlabel node '%s' from crush topology labels", node.Name)
			errCollector++
		}
	}
	if errCollector > 0 {
		return false, errors.New("failed to delete ceph role or crush topology labels from obsolete node(s)")
	}
	return !changed, nil
}

func (c *cephDeploymentConfig) deleteNodesAnnotations(excludeNode ...string) (bool, error) {
	nodes, err := c.api.Kubeclientset.CoreV1().Nodes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}
	errCollector := 0
	changed := false
	for _, node := range nodes.Items {
		if lcmcommon.Contains(excludeNode, node.Name) {
			continue
		}
		updated, err := c.annotateNodes(map[string]string{}, node.Name)
		changed = changed || updated
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to cleanup node '%s' from redundant annotations", node.Name)
			errCollector++
		}
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
			c.log.Info().Msgf("remove node '%s' label %s", node.Name, cephDaemonsetLabel)
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

func (c *cephDeploymentConfig) labelNodes(roles []string, nodeName string) (bool, error) {
	node, err := c.api.Kubeclientset.CoreV1().Nodes().Get(c.context, nodeName, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get '%s' node", nodeName)
	}
	newLabels, updateLabels := buildCephNodeLabels(node.Labels, roles)
	if updateLabels {
		c.log.Info().Msgf("update node '%s' labels (ceph roles)", nodeName)
		lcmcommon.ShowObjectDiff(*c.log, node.Labels, newLabels)
		node.Labels = newLabels
		_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, node, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update '%s' node labels", nodeName)
		}
	}
	return updateLabels, nil
}

func (c *cephDeploymentConfig) annotateNodes(annotations map[string]string, nodeName string) (bool, error) {
	node, err := c.api.Kubeclientset.CoreV1().Nodes().Get(c.context, nodeName, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get '%s' node", nodeName)
	}
	newAnnotations, updateAnnotations := buildCephNodeAnnotations(node.Annotations, annotations)
	if updateAnnotations {
		c.log.Info().Msgf("update node '%s' annotations", nodeName)
		lcmcommon.ShowObjectDiff(*c.log, node.Annotations, newAnnotations)
		node.Annotations = newAnnotations
		_, err = c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, node, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update '%s' node annotations", nodeName)
		}
	}
	return updateAnnotations, nil
}

func (c *cephDeploymentConfig) addTopology(nodeName string, crush map[string]string) (bool, error) {
	node, err := c.api.Kubeclientset.CoreV1().Nodes().Get(c.context, nodeName, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get '%s' node", nodeName)
	}
	updateLabels := false
	oldLabels := map[string]string{}
	if len(node.Labels) > 0 { // len check to avoid nil for map
		for k, v := range node.Labels {
			oldLabels[k] = v
		}
	}
	var key string

	isError := false
	for crushroot, crushtopology := range crush {
		if key = crushTopologyAllowedKeys[crushroot]; key == "" {
			c.log.Error().Msgf("crush topology label not specified for '%s' node: crushroot '%s' is invalid", nodeName, crushroot)
			isError = true
			continue
		}
		if _, ok := node.Labels[key]; !ok {
			updateLabels = true
			if isKubeCrush(crushroot) {
				node.Labels[fmt.Sprintf(cephKubeTopologyLabelTemplate, key)] = crushtopology
			}
			node.Labels[key] = crushtopology
			c.log.Info().Msgf("add label '%s=%s' on %s", key, crushtopology, node.Name)
		} else if node.Labels[key] != crushtopology {
			updateLabels = true
			origLabel := node.Labels[key]
			if isKubeCrush(crushroot) {
				node.Labels[fmt.Sprintf(cephKubeTopologyLabelTemplate, key)] = origLabel
			}
			node.Labels[key] = crushtopology
			c.log.Info().Msgf("change label '%s' from '%s' to '%s' on %s", key, origLabel, crushtopology, node.Name)
		} else if _, ok = node.Labels[fmt.Sprintf(cephKubeTopologyLabelTemplate, key)]; !ok && isKubeCrush(crushroot) {
			node.Labels[fmt.Sprintf(cephKubeTopologyLabelTemplate, key)] = crushtopology
		}
	}
	if isError {
		return false, errors.Errorf("crush topology labels do not changed due to error(s) found in node '%s' crush section", nodeName)
	}
	// remove obsolete crush topology labels
	for allowedKey, topologyPath := range crushTopologyAllowedKeys {
		if v, ok := crush[allowedKey]; ok && v != "" {
			continue
		}
		actualValue, found := node.Labels[topologyPath]

		var originalValue string
		allowed := true
		if isKubeCrush(allowedKey) {
			originalValue, allowed = node.Labels[fmt.Sprintf(cephKubeTopologyLabelTemplate, topologyPath)]
		}
		if found && allowed {
			if isKubeCrush(allowedKey) {
				if actualValue != originalValue {
					node.Labels[topologyPath] = originalValue
				} else {
					delete(node.Labels, topologyPath)
				}
				delete(node.Labels, fmt.Sprintf(cephKubeTopologyLabelTemplate, topologyPath))
			} else {
				delete(node.Labels, topologyPath)
			}
			updateLabels = true
		}
	}
	if updateLabels {
		c.log.Info().Msgf("update node '%s' crush topology labels", nodeName)
		lcmcommon.ShowObjectDiff(*c.log, oldLabels, node.Labels)
		_, err := c.api.Kubeclientset.CoreV1().Nodes().Update(c.context, node, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update '%s' node crush topology labels", nodeName)
		}
	}
	return updateLabels, nil
}
