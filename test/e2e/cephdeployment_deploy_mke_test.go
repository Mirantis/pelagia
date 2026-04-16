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

package test

import (
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rook/rook/pkg/operator/k8sutil"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestDeployCephDeployment(t *testing.T) {
	t.Log("#### e2e test: deploy basic CephDeployment")
	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}

	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["clusterNet"]; !ok {
		t.Fatal("test config does not contain 'clusterNet' key")
	}
	if _, ok := testConfig["publicNet"]; !ok {
		t.Fatal("test config does not contain 'publicNet' key")
	}
	monLabelsStr, monOk := testConfig["monNodesLabel"]
	if !monOk {
		t.Fatal("test config does not contain 'monNodesLabel' key")
	}
	osdLabelsStr, osdOk := testConfig["osdNodesLabel"]
	if !osdOk {
		t.Fatal("test config does not contain 'osdNodesLabel' key")
	}
	tolerations := []corev1.Toleration{}
	if v, ok := testConfig["tolerations"]; ok {
		parsed, err := k8sutil.YamlToTolerations(v)
		if err != nil {
			t.Fatalf("failed to parse provided tolerations: %v", err)
		}
		tolerations = parsed
	}

	f.Step(t, "Checking CephDeployment is not present")
	mcList, err := f.TF.ManagedCluster.ListCephDeployment()
	if err != nil {
		t.Fatalf("failed to get CephDeployment list for cluster: %v", err)
	}

	if len(mcList) != 0 {
		t.Skip("CephDeployment is already deployed on cluster, skipping")
	}

	controlNodesList, err := f.TF.ManagedCluster.ListNodes(monLabelsStr)
	if err != nil {
		t.Fatalf("failed to get nodes list for cluster: %v", err)
	}
	storageNodesList, err := f.TF.ManagedCluster.ListNodes(osdLabelsStr)
	if err != nil {
		t.Fatalf("failed to get nodes list for cluster: %v", err)
	}
	controlNodes := map[string]bool{}
	for _, node := range controlNodesList {
		controlNodes[node.Name] = true
	}
	specNodes := []cephlcmv1alpha1.CephDeploymentNode{}
	for _, node := range storageNodesList {
		newNode := cephlcmv1alpha1.CephDeploymentNode{
			Node: cephv1.Node{
				Name: node.Name,
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{Name: testConfig["defaultDevice"], Config: map[string]string{"deviceClass": "hdd"}},
					},
				},
			},
			Roles: []string{},
		}
		if controlNodes[node.Name] {
			newNode.Roles = []string{"mon", "mgr"}
			delete(controlNodes, node.Name)
		}
		specNodes = append(specNodes, newNode)
	}
	for nodeName := range controlNodes {
		newNode := cephlcmv1alpha1.CephDeploymentNode{
			Node:  cephv1.Node{Name: nodeName},
			Roles: []string{"mon", "mgr"},
		}
		specNodes = append(specNodes, newNode)
	}

	rawCluster, err := cephlcmv1alpha1.DecodeStructToRaw(
		cephv1.ClusterSpec{
			Network: cephv1.NetworkSpec{
				AddressRanges: &cephv1.AddressRangesSpec{
					Public:  []cephv1.CIDR{cephv1.CIDR(testConfig["publicNet"])},
					Cluster: []cephv1.CIDR{cephv1.CIDR(testConfig["clusterNet"])},
				},
			},
			Placement: cephv1.PlacementSpec{
				"all": cephv1.Placement{Tolerations: tolerations},
			},
		})

	if err != nil {
		t.Fatal(err)
	}
	rawPool, err := cephlcmv1alpha1.DecodeStructToRaw(
		cephv1.PoolSpec{
			DeviceClass: "hdd",
			Replicated:  cephv1.ReplicatedSpec{Size: 3},
		})

	if err != nil {
		t.Fatal(err)
	}
	rawRgw, err := cephlcmv1alpha1.DecodeStructToRaw(
		cephv1.ObjectStoreSpec{
			Gateway: cephv1.GatewaySpec{
				Instances:  1,
				Port:       80,
				SecurePort: 8443,
				Placement:  cephv1.Placement{Tolerations: tolerations},
			},
			MetadataPool: cephv1.PoolSpec{
				DeviceClass: "hdd",
				Replicated:  cephv1.ReplicatedSpec{Size: 3},
			},
			DataPool: cephv1.PoolSpec{
				DeviceClass: "hdd",
				ErasureCoded: cephv1.ErasureCodedSpec{
					CodingChunks: 1,
					DataChunks:   2,
				},
			},
		})

	if err != nil {
		t.Fatal(err)
	}

	newCD := &cephlcmv1alpha1.CephDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cephcluster",
			Namespace: f.TF.ManagedCluster.LcmNamespace,
		},
		Spec: cephlcmv1alpha1.CephDeploymentSpec{
			Cluster: &cephlcmv1alpha1.CephCluster{
				RawExtension: runtime.RawExtension{Raw: rawCluster},
			},
			BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
				Pools: []cephlcmv1alpha1.CephPool{
					{
						Name:             "kubernetes",
						StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{Default: true},
						Role:             "kubernetes",
						PoolSpec:         runtime.RawExtension{Raw: rawPool},
					},
				},
			},
			ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
				Rgws: []cephlcmv1alpha1.CephObjectStore{
					{
						Name: "rgw-store",
						Spec: runtime.RawExtension{Raw: rawRgw},
					},
				},
			},
			Nodes: specNodes,
		},
	}

	f.Step(t, "Creating CephDeployment")
	err = f.TF.ManagedCluster.CreateCephDeployment(newCD)
	if err != nil {
		t.Fatalf("failed to create CephDeployment %s/%s for cluster: %v", newCD.Namespace, newCD.Name, err)
	}

	f.Step(t, "Checking CephDeployment and CephDeploymentHealth are fully healthy")
	err = f.WaitForStatusReady(newCD.Name)
	if err != nil {
		t.Fatalf("failed to wait readiness: %v", err)
	}
}
