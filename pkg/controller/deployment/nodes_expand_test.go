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
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
)

func TestBuildExpandedNodeList(t *testing.T) {
	node1 := v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1", Labels: map[string]string{"app-key": "value"}}}
	node2 := v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-2", Labels: map[string]string{"app-key": "value"}}}
	node3 := v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-3"}}
	node4 := v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-4", Labels: map[string]string{"app-key": "value2"}}}
	nodeList := &v1.NodeList{Items: []v1.Node{node1, node2, node3, node4}}

	client := faketestclients.GetClientBuilder().WithLists(nodeList).Build()
	baseNodes := []cephlcmv1alpha1.CephDeploymentNode{
		{
			Node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							Name:   "sda",
							Config: map[string]string{"deviceClass": "hdd"},
						},
					},
				},
			},
		},
		{
			Node: cephv1.Node{
				Name: "node-2",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							Name:   "sda",
							Config: map[string]string{"deviceClass": "hdd"},
						},
					},
				},
			},
		},
	}
	baseNodes2 := make([]cephlcmv1alpha1.CephDeploymentNode, 2)
	labeledNodes := make([]cephlcmv1alpha1.CephDeploymentNode, 2)
	labeledNodes2 := make([]cephlcmv1alpha1.CephDeploymentNode, 2)
	for idx, node := range baseNodes {
		newNode := node.DeepCopy()
		newNode.NodesByLabel = "app-key=value"
		labeledNodes[idx] = *newNode
		labeledNodes2[idx] = *newNode.DeepCopy()
		baseNodes2[idx] = *node.DeepCopy()
	}
	baseNodes2[1].Devices[0].Name = "sdb"
	labeledNodes2[1].Devices[0].Name = "sdb"

	tests := []struct {
		name          string
		nodes         []cephlcmv1alpha1.CephDeploymentNode
		extraOpts     *cephlcmv1alpha1.CephDeploymentExtraOpts
		expectedError string
		expectedNodes []cephlcmv1alpha1.CephDeploymentNode
	}{
		{
			name: "name missed for node item",
			nodes: []cephlcmv1alpha1.CephDeploymentNode{
				{
					NodeGroup: []string{"node-1", "node-2"},
				},
			},
			expectedError: "name missed for node (group) item #0",
		},
		{
			name: "label and node group specified",
			nodes: []cephlcmv1alpha1.CephDeploymentNode{
				{
					Node: cephv1.Node{
						Name: "nodegroup-1",
					},
					NodeGroup:    []string{"node-1", "node-2"},
					NodesByLabel: "test=app",
				},
			},
			expectedError: "labels and node groups used simultaneously for node (group) nodegroup-1",
		},
		{
			name:          "no node transforms, no device labels",
			nodes:         baseNodes,
			expectedNodes: baseNodes,
		},
		{
			name: "node group transforms, no device labels",
			nodes: []cephlcmv1alpha1.CephDeploymentNode{
				{
					Node: cephv1.Node{
						Name: "nodegroup-1",
						Selection: cephv1.Selection{
							Devices: []cephv1.Device{
								{
									Name:   "sda",
									Config: map[string]string{"deviceClass": "hdd"},
								},
							},
						},
					},
					NodeGroup: []string{"node-1", "node-2"},
				},
			},
			expectedNodes: baseNodes,
		},
		{
			name: "node label transforms, no device labels",
			nodes: []cephlcmv1alpha1.CephDeploymentNode{
				{
					Node: cephv1.Node{
						Name: "labelgroup-1",
						Selection: cephv1.Selection{
							Devices: []cephv1.Device{
								{
									Name:   "sda",
									Config: map[string]string{"deviceClass": "hdd"},
								},
							},
						},
					},
					NodesByLabel: "app-key=value",
				},
			},
			expectedNodes: labeledNodes,
		},
		{
			name: "no node transforms, device labels present",
			nodes: []cephlcmv1alpha1.CephDeploymentNode{
				{
					Node: cephv1.Node{
						Name: "node-1",
						Selection: cephv1.Selection{
							Devices: []cephv1.Device{
								{
									Name:   "dev-name-1",
									Config: map[string]string{"deviceClass": "hdd"},
								},
							},
						},
					},
				},
				baseNodes[1],
			},
			extraOpts: &cephlcmv1alpha1.CephDeploymentExtraOpts{
				DeviceLabels: map[string]cephlcmv1alpha1.LabeledDevices{
					"node-1": {
						"dev-name-1": "sda",
					},
				},
			},
			expectedNodes: baseNodes,
		},
		{
			name: "node group transforms, device labels present",
			nodes: []cephlcmv1alpha1.CephDeploymentNode{
				{
					Node: cephv1.Node{
						Name: "nodegroup-1",
						Selection: cephv1.Selection{
							Devices: []cephv1.Device{
								{
									Name:   "dev-name",
									Config: map[string]string{"deviceClass": "hdd"},
								},
							},
						},
					},
					NodeGroup: []string{"node-1", "node-2"},
				},
			},
			extraOpts: &cephlcmv1alpha1.CephDeploymentExtraOpts{
				DeviceLabels: map[string]cephlcmv1alpha1.LabeledDevices{
					"node-1": {
						"dev-name": "sda",
					},
					"node-2": {
						"dev-name": "sdb",
					},
				},
			},
			expectedNodes: baseNodes2,
		},
		{
			name: "node transforms, device labels present",
			nodes: []cephlcmv1alpha1.CephDeploymentNode{
				{
					Node: cephv1.Node{
						Name: "labelgroup-1",
						Selection: cephv1.Selection{
							Devices: []cephv1.Device{
								{
									Name:   "dev-name",
									Config: map[string]string{"deviceClass": "hdd"},
								},
							},
						},
					},
					NodesByLabel: "app-key=value",
				},
			},
			extraOpts: &cephlcmv1alpha1.CephDeploymentExtraOpts{
				DeviceLabels: map[string]cephlcmv1alpha1.LabeledDevices{
					"node-1": {
						"dev-name": "sda",
					},
					"node-2": {
						"dev-name": "sdb",
					},
				},
			},
			expectedNodes: labeledNodes2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			c.api.Client = client
			c.cdConfig.cephDpl.Spec.Nodes = test.nodes
			c.cdConfig.cephDpl.Spec.ExtraOpts = test.extraOpts

			resultNodes, err := c.buildExpandedNodeList()
			if test.expectedError == "" {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedNodes, resultNodes)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			}
		})
	}
}
func TestGetExpandedDevicesList(t *testing.T) {
	tests := []struct {
		name             string
		nodeDevices      []cephv1.Device
		nodeDeviceLabels cephlcmv1alpha1.LabeledDevices
		expectedDevices  []cephv1.Device
	}{
		{
			name: "base devices spec, no labels",
			nodeDevices: []cephv1.Device{
				{
					Name:   "sda",
					Config: map[string]string{"deviceClass": "hdd"},
				},
			},
			expectedDevices: []cephv1.Device{
				{
					Name:   "sda",
					Config: map[string]string{"deviceClass": "hdd"},
				},
			},
		},
		{
			name: "base devices spec, labels specified not used",
			nodeDevices: []cephv1.Device{
				{
					Name:   "sda",
					Config: map[string]string{"deviceClass": "hdd"},
				},
			},
			nodeDeviceLabels: cephlcmv1alpha1.LabeledDevices{
				"some-label": "some-key",
			},
			expectedDevices: []cephv1.Device{
				{
					Name:   "sda",
					Config: map[string]string{"deviceClass": "hdd"},
				},
			},
		},
		{
			name: "labels specified and used",
			nodeDevices: []cephv1.Device{
				{
					Name:   "sdx",
					Config: map[string]string{"deviceClass": "hdd"},
				},
				{
					Name:   "dev-name-label",
					Config: map[string]string{"deviceClass": "hdd"},
				},
				{
					Name:   "dev-id-label",
					Config: map[string]string{"deviceClass": "hdd", "metadataDevice": "sdx"},
				},
				{
					Name:   "dev-path-label",
					Config: map[string]string{"deviceClass": "hdd", "metadataDevice": "metadata-dev-1"},
				},
			},
			nodeDeviceLabels: cephlcmv1alpha1.LabeledDevices{
				"dev-name-label": "sda",
				"dev-id-label":   "/dev/disk/by-id/seagate-123fdg1",
				"dev-path-label": "/dev/disk/by-path/pci-0000.1",
				"metadata-dev-1": "/dev/ceph-meta/part-1",
			},
			expectedDevices: []cephv1.Device{
				{
					Name:   "sdx",
					Config: map[string]string{"deviceClass": "hdd"},
				},
				{
					Name:   "sda",
					Config: map[string]string{"deviceClass": "hdd"},
				},
				{
					Name:   "/dev/disk/by-id/seagate-123fdg1",
					Config: map[string]string{"deviceClass": "hdd", "metadataDevice": "sdx"},
				},
				{
					FullPath: "/dev/disk/by-path/pci-0000.1",
					Config:   map[string]string{"deviceClass": "hdd", "metadataDevice": "/dev/ceph-meta/part-1"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultDevices := getExpandedDevicesList(test.nodeDevices, test.nodeDeviceLabels)
			assert.Equal(t, test.expectedDevices, resultDevices)
		})
	}
}
