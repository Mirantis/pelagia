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
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

// usage of controller specific global vars across different controllers
func (c *cephDeploymentConfig) buildExpandedNodeList() ([]cephlcmv1alpha1.CephDeploymentNode, error) {
	expandedList := make([]cephlcmv1alpha1.CephDeploymentNode, 0)
	for idx, node := range c.cdConfig.cephDpl.Spec.Nodes {
		if node.Name == "" {
			msg := fmt.Sprintf("name missed for node (group) item #%d", idx)
			return expandedList, errors.New(msg)
		}
		if len(node.NodeGroup) > 0 && node.NodesByLabel != "" {
			msg := fmt.Sprintf("labels and node groups used simultaneously for node (group) %s", node.Name)
			return expandedList, errors.New(msg)
		}
		if len(node.NodeGroup) > 0 {
			for _, nodeName := range node.NodeGroup {
				newNode := node.DeepCopy()
				newNode.Name = nodeName
				newNode.NodeGroup = nil
				if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && c.cdConfig.cephDpl.Spec.ExtraOpts.DeviceLabels != nil {
					newNode.Devices = getExpandedDevicesList(newNode.Devices, c.cdConfig.cephDpl.Spec.ExtraOpts.DeviceLabels[nodeName])
				}
				expandedList = append(expandedList, *newNode)
			}
			continue
		}
		if node.NodesByLabel != "" {
			selector, _ := labels.Parse(node.NodesByLabel)
			label := &client.ListOptions{LabelSelector: selector}
			nodesByLabel := &v1.NodeList{}
			err := c.api.Client.List(c.context, nodesByLabel, label)
			if err != nil {
				return expandedList, errors.Wrapf(err, "failed to get nodes with label: %s", node.NodesByLabel)
			}
			for _, nodeByLabel := range nodesByLabel.Items {
				newNode := node.DeepCopy()
				newNode.Name = nodeByLabel.Name
				newNode.NodesByLabel = node.NodesByLabel
				if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && c.cdConfig.cephDpl.Spec.ExtraOpts.DeviceLabels != nil {
					newNode.Devices = getExpandedDevicesList(newNode.Devices, c.cdConfig.cephDpl.Spec.ExtraOpts.DeviceLabels[nodeByLabel.Name])
				}
				expandedList = append(expandedList, *newNode)
			}
			continue
		}
		if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && c.cdConfig.cephDpl.Spec.ExtraOpts.DeviceLabels != nil {
			node.Devices = getExpandedDevicesList(node.Devices, c.cdConfig.cephDpl.Spec.ExtraOpts.DeviceLabels[node.Name])
		}
		expandedList = append(expandedList, node)
	}
	return expandedList, nil
}

func getExpandedDevicesList(nodeDevices []cephv1.Device, nodeDeviceLabels cephlcmv1alpha1.LabeledDevices) []cephv1.Device {
	if len(nodeDeviceLabels) == 0 {
		return nodeDevices
	}
	for idx, device := range nodeDevices {
		if realDevLink, ok := nodeDeviceLabels[device.Name]; ok {
			if strings.HasPrefix(realDevLink, "/dev/disk/by-path/") {
				device.Name = ""
				device.FullPath = realDevLink
			} else {
				device.Name = realDevLink
				device.FullPath = ""
			}
			nodeDevices[idx] = device
		}
		if metadataDev, ok := device.Config["metadataDevice"]; ok {
			if metaLink, ok := nodeDeviceLabels[metadataDev]; ok {
				device.Config["metadataDevice"] = metaLink
				nodeDevices[idx] = device
			}
		}
	}
	return nodeDevices
}
