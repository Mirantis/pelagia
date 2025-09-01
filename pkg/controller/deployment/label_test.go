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
	"reflect"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestLabelNodes(t *testing.T) {
	tests := []struct {
		name          string
		labels        []string
		nodes         *v1.NodeList
		expectedNodes *v1.NodeList
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:          "label nodes - get node failed",
			labels:        []string{},
			nodes:         &v1.NodeList{},
			expectedNodes: &v1.NodeList{},
			expectedError: "failed to get 'node1' node: nodes \"node1\" not found",
		},
		{
			name:          "label nodes - no labels, update skipped",
			labels:        []string{},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			apiErrors:     map[string]error{"update-nodes": errors.New("unexpected update call")},
		},
		{
			name:          "label nodes - all labels, update skipped",
			labels:        []string{"mon", "mgr", "osd", "rgw", "mds"},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RoleLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.RoleLabelsNode.DeepCopy()}},
			apiErrors:     map[string]error{"update-nodes": errors.New("unexpected update call")},
		},
		{
			name:          "label nodes - remove all labels, update success",
			labels:        []string{},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RoleLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
		},
		{
			name:          "label nodes - add all labels, update success",
			labels:        []string{"mon", "mgr", "osd", "rgw", "mds"},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.RoleLabelsNode.DeepCopy()}},
		},
		{
			name:          "label nodes - update failed",
			labels:        []string{"mon", "mgr", "osd", "rgw", "mds"},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			apiErrors:     map[string]error{"update-nodes": errors.New("update failed")},
			expectedError: "failed to update 'node1' node labels: update failed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			changeExpected := !reflect.DeepEqual(test.nodes, test.expectedNodes)
			res := map[string]runtime.Object{"nodes": test.nodes}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, res, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"nodes"}, res, test.apiErrors)

			changed, err := c.labelNodes(test.labels, "node1")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedNodes, test.nodes)
			assert.Equal(t, changeExpected, changed)
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestAddTopology(t *testing.T) {
	tests := []struct {
		name          string
		topology      map[string]string
		nodes         *v1.NodeList
		expectedNodes *v1.NodeList
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:          "add topology to nodes - get node failed",
			topology:      map[string]string{},
			nodes:         &v1.NodeList{},
			expectedNodes: &v1.NodeList{},
			expectedError: "failed to get 'node1' node: nodes \"node1\" not found",
		},
		{
			name:          "add topology to nodes - no topology, update skipped",
			topology:      map[string]string{},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			apiErrors:     map[string]error{"update-nodes": errors.New("unexpected update call")},
		},
		{
			name:          "add topology to nodes - add all topology, update success",
			topology:      map[string]string{"region": "region1", "zone": "zone1", "rack": "rack1"},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.TopologyLabelsNode.DeepCopy()}},
		},
		{
			name:          "add topology to nodes - remove all topology, update success",
			topology:      map[string]string{},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.TopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
		},
		{
			name:     "add topology to nodes - remove all topology, update region and zone skipped",
			topology: map[string]string{},
			nodes:    &v1.NodeList{Items: []v1.Node{*unitinputs.TopologyLabelsNodeNoRoles.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{
				func() v1.Node {
					expected := unitinputs.TopologyLabelsNodeNoRoles.DeepCopy()
					delete(expected.Labels, "topology.rook.io/rack")
					return *expected
				}(),
			}},
		},
		{
			name:     "add topology to nodes - remove all topology, return region and zone to original values",
			topology: map[string]string{},
			nodes:    &v1.NodeList{Items: []v1.Node{*unitinputs.TopologyLabelsNodeOrigRoles.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{
				func() v1.Node {
					expected := unitinputs.TopologyLabelsNodeOrigRoles.DeepCopy()
					delete(expected.Labels, "topology.rook.io/rack")
					delete(expected.Labels, "cephdpl-prev-topology.kubernetes.io/region")
					delete(expected.Labels, "cephdpl-prev-topology.kubernetes.io/zone")
					expected.Labels["topology.kubernetes.io/region"] = "orig-region"
					expected.Labels["topology.kubernetes.io/zone"] = "orig-zone"
					return *expected
				}(),
			}},
		},
		{
			name:          "add topology to nodes - update failed",
			topology:      map[string]string{},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.TopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.TopologyLabelsNode.DeepCopy()}},
			apiErrors:     map[string]error{"update-nodes": errors.New("failed update")},
			expectedError: "failed to update 'node1' node crush topology labels: failed update",
		},
		{
			name:     "add topology to nodes - change topology, update success",
			topology: map[string]string{"zone": "zone1", "rack": "rack-new"},
			nodes:    &v1.NodeList{Items: []v1.Node{*unitinputs.NotAllTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{
				func() v1.Node {
					expected := unitinputs.TopologyLabelsNode.DeepCopy()
					delete(expected.Labels, "topology.kubernetes.io/region")
					delete(expected.Labels, "cephdpl-prev-topology.kubernetes.io/region")
					expected.Labels["topology.rook.io/rack"] = "rack-new"
					return *expected
				}(),
			}},
		},
		{
			name:          "add topology to nodes - invalid topology failed",
			topology:      map[string]string{"fake": "fake"},
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedError: "crush topology labels do not changed due to error(s) found in node 'node1' crush section",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			changeExpected := !reflect.DeepEqual(test.nodes, test.expectedNodes)
			res := map[string]runtime.Object{"nodes": test.nodes}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, res, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"nodes"}, res, test.apiErrors)

			changed, err := c.addTopology("node1", test.topology)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedNodes, test.nodes)
			assert.Equal(t, changeExpected, changed)
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestDeleteLabelNodes(t *testing.T) {
	tests := []struct {
		name          string
		excludeNode   string
		nodes         *v1.NodeList
		expectedNodes *v1.NodeList
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:          "delete node ceph labels - node excluded, skipped",
			excludeNode:   "node1",
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
		},
		{
			name:          "delete node ceph labels - node list failed",
			expectedError: "failed to list nodes: failed to list nodes",
		},
		{
			name:          "delete node ceph labels - ceph roles removing failed",
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RoleLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RoleLabelsNode}},
			apiErrors:     map[string]error{"update-nodes": errors.New("update failed")},
			expectedError: "failed to delete ceph role or crush topology labels from obsolete node(s)",
		},
		{
			name:          "delete node ceph labels - ceph topology labels removing failed",
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.TopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.TopologyLabelsNode}},
			apiErrors:     map[string]error{"update-nodes": errors.New("update failed")},
			expectedError: "failed to delete ceph role or crush topology labels from obsolete node(s)",
		},
		{
			name:          "delete node ceph labels - ceph both types labels removing failed",
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			apiErrors:     map[string]error{"update-nodes": errors.New("update failed")},
			expectedError: "failed to delete ceph role or crush topology labels from obsolete node(s)",
		},
		{
			name:          "delete node ceph labels - success",
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy(), unitinputs.GetAvailableNode("node-a")}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode, unitinputs.GetAvailableNode("node-a")}},
		},
		{
			name:          "delete node ceph labels - nothing to remove",
			nodes:         &v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode, unitinputs.GetAvailableNode("node-a")}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode, unitinputs.GetAvailableNode("node-a")}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			allRemoved := reflect.DeepEqual(test.nodes, test.expectedNodes)
			res := map[string]runtime.Object{}
			if test.nodes != nil {
				res["nodes"] = test.nodes
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, res, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, res, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"nodes"}, res, test.apiErrors)

			deleted, err := c.deleteLabelNodes(test.excludeNode)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, deleted)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, allRemoved, deleted)
			}
			if test.nodes != nil {
				assert.Equal(t, test.expectedNodes, test.nodes)
			}
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestDeleteDaemonSetLabels(t *testing.T) {
	tests := []struct {
		name          string
		nodes         *v1.NodeList
		expectedNodes *v1.NodeList
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name: "delete daemon set labels - success",
			nodes: &v1.NodeList{Items: []v1.Node{
				unitinputs.GetAvailableNode("node-a"), unitinputs.GetAvailableNode("node-b"),
				unitinputs.GetNodeWithLabels("node-c", nil, nil), *unitinputs.RolesTopologyLabelsNode.DeepCopy(),
			}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{
				unitinputs.GetNodeWithLabels("node-a", map[string]string{}, nil), unitinputs.GetNodeWithLabels("node-b", map[string]string{}, nil),
				unitinputs.GetNodeWithLabels("node-c", nil, nil), *unitinputs.RolesTopologyLabelsNode.DeepCopy(),
			}},
		},
		{
			name: "delete daemon set labels - nothing to delete",
			nodes: &v1.NodeList{Items: []v1.Node{
				unitinputs.GetNodeWithLabels("node-a", nil, nil), *unitinputs.RolesTopologyLabelsNode.DeepCopy(),
			}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{
				unitinputs.GetNodeWithLabels("node-a", nil, nil), *unitinputs.RolesTopologyLabelsNode.DeepCopy(),
			}},
		},
		{
			name:          "delete daemon set labels - node list failed",
			expectedError: "failed to list nodes: failed to list nodes",
		},
		{
			name:          "delete daemon set labels - daemons set roles removing failed for all",
			nodes:         &v1.NodeList{Items: []v1.Node{unitinputs.GetAvailableNode("node-a")}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.GetAvailableNode("node-a")}},
			apiErrors:     map[string]error{"update-nodes": errors.New("node update failed")},
			expectedError: "failed to delete daemonset labels from some nodes",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			allRemoved := reflect.DeepEqual(test.nodes, test.expectedNodes)
			res := map[string]runtime.Object{}
			if test.nodes != nil {
				res["nodes"] = test.nodes
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, res, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"nodes"}, res, test.apiErrors)

			deleted, err := c.deleteDaemonSetLabels()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, deleted)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, allRemoved, deleted)
			}
			if test.nodes != nil {
				assert.Equal(t, test.expectedNodes, test.nodes)
			}
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestEnsureLabelNodes(t *testing.T) {
	tests := []struct {
		name          string
		cephDpl       *cephlcmv1alpha1.CephDeployment
		nodes         *v1.NodeList
		deployments   *appsv1.DeploymentList
		expectedNodes *v1.NodeList
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:          "ensure node ceph labels - failed to list nodes",
			cephDpl:       unitinputs.CephDeployEnsureRolesCrush.DeepCopy(),
			expectedError: "failed to list nodes: failed to list nodes",
		},
		{
			name:          "ensure node ceph labels - failed to check deployments on a node",
			cephDpl:       unitinputs.CephDeployEnsureRolesCrush.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			expectedError: "failed to check osd deployments for some node(s) with osd role",
		},
		{
			name:          "ensure node ceph labels - role and topology labels update error",
			cephDpl:       unitinputs.CephDeployEnsureRolesCrush.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode}},
			deployments:   &appsv1.DeploymentList{},
			apiErrors:     map[string]error{"update-nodes": errors.New("node update failed")},
			expectedError: "failed to set role or crush topology labels for some node(s)",
		},
		{
			name:          "ensure node ceph labels - cleanup obsolete ceph labels error",
			cephDpl:       &unitinputs.CephDeployExternal,
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			deployments:   &appsv1.DeploymentList{},
			apiErrors:     map[string]error{"update-nodes": errors.New("node update failed")},
			expectedError: "failed to delete ceph role or crush topology labels from obsolete node(s)",
		},
		{
			name:          "ensure node ceph labels  - no changes, success",
			cephDpl:       unitinputs.CephDeployEnsureRolesCrush.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			deployments:   &appsv1.DeploymentList{},
			apiErrors:     map[string]error{"update-nodes": errors.New("unexpected update failed")},
		},
		{
			name:          "ensure node ceph labels  - roles, topology updated, success",
			cephDpl:       unitinputs.CephDeployEnsureRolesCrush.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			deployments:   &appsv1.DeploymentList{},
		},
		{
			name: "ensure node ceph labels  - roles only updated, success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRolesCrush.DeepCopy()
				mc.Spec.Nodes = []cephlcmv1alpha1.CephDeploymentNode{
					{
						Node: cephv1.Node{
							Name: "node1",
						},
						Roles: []string{"mon", "mgr", "rgw", "mds", "osd"},
					},
				}
				return mc
			}(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RoleLabelsNode}},
			deployments:   &appsv1.DeploymentList{},
		},
		{
			name:          "ensure node ceph labels  - topology updated, success",
			cephDpl:       unitinputs.CephDeployEnsureRolesCrush.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RoleLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			deployments:   &appsv1.DeploymentList{},
		},
		{
			name: "ensure node ceph labels when no osd role but configuration - success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRolesCrush.DeepCopy()
				mc.Spec.Nodes = []cephlcmv1alpha1.CephDeploymentNode{
					{
						Node: cephv1.Node{
							Name: "node1",
							Selection: cephv1.Selection{
								DeviceFilter: "sda",
							},
						},
						Roles: []string{"mon", "mgr", "rgw", "mds"},
						Crush: map[string]string{
							"region": "region1",
							"zone":   "zone1",
							"rack":   "rack1",
						},
					},
				}
				return mc
			}(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RoleLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			deployments:   &appsv1.DeploymentList{},
		},
		{
			name: "ensure node ceph labels no osd configuration, but osd deployment is not removed - success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRolesCrush.DeepCopy()
				mc.Spec.Nodes = []cephlcmv1alpha1.CephDeploymentNode{
					{
						Node: cephv1.Node{
							Name: "node1",
						},
						Roles: []string{"mon", "mgr", "rgw", "mds"},
					},
				}
				return mc
			}(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.RolesTopologyLabelsNode}},
			deployments: &appsv1.DeploymentList{Items: []appsv1.Deployment{
				*unitinputs.GetDeployment("rook-ceph-osd-1", "rook-ceph", map[string]string{"app": "rook-ceph-osd", "failure-domain": "node1"}, nil)}},
			apiErrors: map[string]error{"update-nodes": errors.New("unexpected update failed")},
		},
		{
			name: "ensure node ceph labels when it is removed from spec, but osd deployment is not removed - success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRolesCrush.DeepCopy()
				mc.Spec.Nodes = []cephlcmv1alpha1.CephDeploymentNode{}
				return mc
			}(),
			nodes: &v1.NodeList{Items: []v1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.GetNodeWithLabels(unitinputs.RolesTopologyLabelsNode.Name,
				map[string]string{
					"ceph_role_osd":                              "true",
					"topology.kubernetes.io/region":              "region1",
					"cephdpl-prev-topology.kubernetes.io/region": "region1",
					"topology.kubernetes.io/zone":                "zone1",
					"cephdpl-prev-topology.kubernetes.io/zone":   "zone1",
					"topology.rook.io/rack":                      "rack1",
				}, nil)}},
			deployments: &appsv1.DeploymentList{Items: []appsv1.Deployment{
				*unitinputs.GetDeployment("rook-ceph-osd-1", "rook-ceph", map[string]string{"app": "rook-ceph-osd", "failure-domain": "node1"}, nil)}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			c.cdConfig.nodesListExpanded = test.cephDpl.Spec.Nodes
			changeExpected := !reflect.DeepEqual(test.nodes, test.expectedNodes)

			res := map[string]runtime.Object{}
			if test.nodes != nil {
				res["nodes"] = test.nodes
			}
			if test.deployments != nil {
				res["deployments"] = test.deployments
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "list", []string{"deployments"}, res, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, res, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, res, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"nodes"}, res, test.apiErrors)

			changed, err := c.ensureLabelNodes()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, changed)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, changeExpected, changed)
			}
			if test.nodes != nil {
				assert.Equal(t, test.expectedNodes, test.nodes)
			}
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
}

func TestEnsureNodesAnnotation(t *testing.T) {
	tests := []struct {
		name          string
		cephDpl       *cephlcmv1alpha1.CephDeployment
		nodes         *v1.NodeList
		expectedNodes *v1.NodeList
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:          "ensure node ceph annotations - failed to list nodes",
			cephDpl:       unitinputs.CephDeployEnsureRolesCrush.DeepCopy(),
			expectedError: "failed to list nodes: failed to list nodes",
		},
		{
			name:          "ensure node ceph annotations - monitor ip annotation update error",
			cephDpl:       unitinputs.CephDeployEnsureMonitorIP.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode}},
			apiErrors:     map[string]error{"update-nodes": errors.New("node update failed")},
			expectedError: "failed to set rook annotations for some node(s)",
		},
		{
			name:          "ensure node ceph annotations - cleanup obsolete ceph annotations error",
			cephDpl:       &unitinputs.CephDeployExternal,
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.NodeMonitorIPAnnotation.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.NodeMonitorIPAnnotation}},
			apiErrors:     map[string]error{"update-nodes": errors.New("node update failed")},
			expectedError: "failed to delete rook annotations from obsolete node(s)",
		},
		{
			name:          "ensure node ceph annotations - cleanup obsolete ceph annotations, success",
			cephDpl:       &unitinputs.CephDeployExternal,
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.NodeMonitorIPAnnotation.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.EmptyAnnotationsNode}},
		},
		{
			name:          "ensure node ceph annotations  - no changes, success",
			cephDpl:       unitinputs.CephDeployEnsureMonitorIP.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.NodeMonitorIPAnnotation.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.NodeMonitorIPAnnotation}},
			apiErrors:     map[string]error{"update-nodes": errors.New("unexpected update failed")},
		},
		{
			name:          "ensure node ceph annotations  - updated, success",
			cephDpl:       unitinputs.CephDeployEnsureMonitorIP.DeepCopy(),
			nodes:         &v1.NodeList{Items: []v1.Node{*unitinputs.EmptyLabelsNode.DeepCopy()}},
			expectedNodes: &v1.NodeList{Items: []v1.Node{unitinputs.NodeMonitorIPAnnotation}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			c.cdConfig.nodesListExpanded = test.cephDpl.Spec.Nodes
			changeExpected := !reflect.DeepEqual(test.nodes, test.expectedNodes)

			res := map[string]runtime.Object{}
			if test.nodes != nil {
				res["nodes"] = test.nodes
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, res, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, res, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"nodes"}, res, test.apiErrors)

			changed, err := c.ensureNodesAnnotation()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, changed)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, changeExpected, changed)
			}
			if test.nodes != nil {
				assert.Equal(t, test.expectedNodes, test.nodes)
			}
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}
