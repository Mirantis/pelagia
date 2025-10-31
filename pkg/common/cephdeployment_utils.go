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

package lcmcommon

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmclient "github.com/Mirantis/pelagia/pkg/client/clientset/versioned"
)

func IsClusterMaintenanceActing(ctx context.Context, cephLcmclientset lcmclient.Interface, namespace, name string) (bool, error) {
	cdm, err := cephLcmclientset.LcmV1alpha1().CephDeploymentMaintenances(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get CephDeploymentMaintenance %s/%s", namespace, name)
	}
	return cdm.Status != nil && (cdm.Status.State == cephlcmv1alpha1.MaintenanceActing || cdm.Status.State == cephlcmv1alpha1.MaintenanceFailing), nil
}

func IsOpenStackPoolsPresent(pools []cephlcmv1alpha1.CephPool) bool {
	// since on validation stage we are checking that pools section is correct
	// we can just simply check any openstack pool existence now
	expectedRoles := []string{"images", "vms", "backup", "volumes"}
	for _, pool := range pools {
		if Contains(expectedRoles, pool.Role) {
			return true
		}
	}
	return false
}

func IsCephOsdNode(node cephv1.Node) bool {
	return len(node.Devices) > 0 || node.DeviceFilter != "" || node.DevicePathFilter != ""
}

func BuildCephNodeLabels(currentNodeLabels map[string]string, roles []string) (map[string]string, bool) {
	newLabels := map[string]string{}
	for k, v := range currentNodeLabels {
		newLabels[k] = v
	}
	changed := false
	for _, baseRole := range CephDaemonKeys {
		if !Contains(roles, baseRole) {
			if _, ok := newLabels[CephNodeLabels[baseRole]]; ok {
				delete(newLabels, CephNodeLabels[baseRole])
				changed = true
			}
		} else if newLabels[CephNodeLabels[baseRole]] != "true" {
			newLabels[CephNodeLabels[baseRole]] = "true"
			changed = true
		}
	}
	return newLabels, changed
}

func GetExpandedCephDeploymentNodeList(ctx context.Context, crclient client.Client, cephDeploymentSpec cephlcmv1alpha1.CephDeploymentSpec) ([]cephlcmv1alpha1.CephDeploymentNode, error) {
	expandedList := make([]cephlcmv1alpha1.CephDeploymentNode, 0)
	for idx, node := range cephDeploymentSpec.Nodes {
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
				if cephDeploymentSpec.ExtraOpts != nil && cephDeploymentSpec.ExtraOpts.DeviceLabels != nil {
					newNode.Devices = GetExpandedCephNodeDevicesList(newNode.Devices, cephDeploymentSpec.ExtraOpts.DeviceLabels[nodeName])
				}
				expandedList = append(expandedList, *newNode)
			}
			continue
		}
		if node.NodesByLabel != "" {
			selector, _ := labels.Parse(node.NodesByLabel)
			label := &client.ListOptions{LabelSelector: selector}
			nodesByLabel := &v1.NodeList{}
			err := crclient.List(ctx, nodesByLabel, label)
			if err != nil {
				return expandedList, errors.Wrapf(err, "failed to get nodes with label: %s", node.NodesByLabel)
			}
			for _, nodeByLabel := range nodesByLabel.Items {
				newNode := node.DeepCopy()
				newNode.Name = nodeByLabel.Name
				newNode.NodesByLabel = node.NodesByLabel
				if cephDeploymentSpec.ExtraOpts != nil && cephDeploymentSpec.ExtraOpts.DeviceLabels != nil {
					newNode.Devices = GetExpandedCephNodeDevicesList(newNode.Devices, cephDeploymentSpec.ExtraOpts.DeviceLabels[nodeByLabel.Name])
				}
				expandedList = append(expandedList, *newNode)
			}
			continue
		}
		if cephDeploymentSpec.ExtraOpts != nil && cephDeploymentSpec.ExtraOpts.DeviceLabels != nil {
			node.Devices = GetExpandedCephNodeDevicesList(node.Devices, cephDeploymentSpec.ExtraOpts.DeviceLabels[node.Name])
		}
		expandedList = append(expandedList, node)
	}
	return expandedList, nil
}

func GetExpandedCephNodeDevicesList(nodeDevices []cephv1.Device, nodeDeviceLabels cephlcmv1alpha1.LabeledDevices) []cephv1.Device {
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
