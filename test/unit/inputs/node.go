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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNodeWithLabels(nodeName string, labels map[string]string, annotations map[string]string) corev1.Node {
	return corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName, Labels: labels, Annotations: annotations}}
}

var EmptyLabelsNode = GetNodeWithLabels("node1", map[string]string{}, nil)

var EmptyAnnotationsNode = GetNodeWithLabels("node1", map[string]string{}, map[string]string{})

var RoleLabelsNode = GetNodeWithLabels("node1", map[string]string{
	"ceph_role_mon": "true",
	"ceph_role_mgr": "true",
	"ceph_role_osd": "true",
	"ceph_role_rgw": "true",
	"ceph_role_mds": "true",
}, nil)

var TopologyLabelsNode = GetNodeWithLabels("node1", map[string]string{
	"topology.kubernetes.io/region":              "region1",
	"cephdpl-prev-topology.kubernetes.io/region": "region1",
	"topology.kubernetes.io/zone":                "zone1",
	"cephdpl-prev-topology.kubernetes.io/zone":   "zone1",
	"topology.rook.io/rack":                      "rack1",
}, nil)

var TopologyLabelsNodeNoRoles = GetNodeWithLabels("node1", map[string]string{
	"topology.kubernetes.io/region": "region1",
	"topology.kubernetes.io/zone":   "zone1",
	"topology.rook.io/rack":         "rack1",
}, nil)

var TopologyLabelsNodeOrigRoles = GetNodeWithLabels("node1", map[string]string{
	"topology.kubernetes.io/region":              "region1",
	"cephdpl-prev-topology.kubernetes.io/region": "orig-region",
	"topology.kubernetes.io/zone":                "zone1",
	"cephdpl-prev-topology.kubernetes.io/zone":   "orig-zone",
	"topology.rook.io/rack":                      "rack1",
}, nil)

var NotAllTopologyLabelsNode = GetNodeWithLabels("node1", map[string]string{
	"topology.kubernetes.io/region":              "region1",
	"cephdpl-prev-topology.kubernetes.io/region": "region1",
	"topology.rook.io/rack":                      "rack1",
}, nil)

var RolesTopologyLabelsNode = GetNodeWithLabels("node1", map[string]string{
	"ceph_role_mon":                 "true",
	"ceph_role_mgr":                 "true",
	"ceph_role_osd":                 "true",
	"ceph_role_rgw":                 "true",
	"ceph_role_mds":                 "true",
	"topology.kubernetes.io/region": "region1",
	"cephdpl-prev-topology.kubernetes.io/region": "region1",
	"topology.kubernetes.io/zone":                "zone1",
	"cephdpl-prev-topology.kubernetes.io/zone":   "zone1",
	"topology.rook.io/rack":                      "rack1",
}, nil)

var NodeMonitorIPAnnotation = GetNodeWithLabels("node1", map[string]string{}, map[string]string{"network.rook.io/mon-ip": "127.0.0.1"})

var DaemonSetLabelsNode = GetNodeWithLabels("node1", map[string]string{"ceph-daemonset-available-node": "true"}, nil)

var LcmDrainedAnnotationsNode = GetNodeWithLabels("node-1", map[string]string{"ceph-daemonset-available-node": "true"},
	map[string]string{"kaas.mirantis.com/lcm-drained": "true"})

var CsiDrainedAnnotationsNode = GetNodeWithLabels("node-1", map[string]string{},
	map[string]string{"kaas.mirantis.com/lcm-drained": "true", "kaas.mirantis.com/csi-drained": "true"})

var LcmDrainedNoCsiLabelNode = GetNodeWithLabels("node-1", map[string]string{},
	map[string]string{"kaas.mirantis.com/lcm-drained": "true"})

func GetAvailableNode(name string) corev1.Node {
	return GetNodeWithLabels(name, map[string]string{"ceph-daemonset-available-node": "true"}, nil)
}

func GetOsdNodesList(names []string) *corev1.NodeList {
	nodesArray := make([]corev1.Node, 0)
	for _, name := range names {
		nodesArray = append(nodesArray, corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"ceph_role_osd": "true"},
			},
		})
	}
	return &corev1.NodeList{Items: nodesArray}
}

type NodeAttrs struct {
	Name        string
	Labeled     bool
	Unreachable bool
}

func GetNodesList(nodes []NodeAttrs) corev1.NodeList {
	list := make([]corev1.Node, len(nodes))
	for idx, attrs := range nodes {
		node := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: attrs.Name}}
		if attrs.Labeled {
			node.Labels = map[string]string{"pelagia-disk-daemon": "true"}
		}
		if attrs.Unreachable {
			node.Spec.Taints = []corev1.Taint{{Key: "node.kubernetes.io/unreachable"}}
		}
		list[idx] = node
	}
	return corev1.NodeList{Items: list}
}
