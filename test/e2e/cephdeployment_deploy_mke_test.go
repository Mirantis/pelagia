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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestDeployCephDeploymentMKE(t *testing.T) {
	t.Log("#### e2e test: deploy basic CephDeployment on top of MKE")
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

	f.Step(t, "Checking CephDeployment is not present")
	mcList, err := f.TF.ManagedCluster.ListCephDeployment()
	if err != nil {
		t.Fatalf("failed to get CephDeployment list for MKE cluster: %v", err)
	}

	if len(mcList) != 0 {
		t.Skip("CephDeployment is already deployed on MKE cluster, skipping")
	}

	mkeNodes, err := f.TF.ManagedCluster.ListNodes()
	if err != nil {
		t.Fatalf("failed to get nodes list for MKE cluster: %v", err)
	}

	mcNodes := []cephlcmv1alpha1.CephDeploymentNode{}
	for _, node := range mkeNodes {
		if role, ok := node.Labels["mke-ceph-node"]; ok {
			mcNode := cephlcmv1alpha1.CephDeploymentNode{
				Node:  cephv1.Node{Name: node.Name},
				Roles: []string{"mon", "mgr"},
			}
			if role == "storage" || role == "storage-dedicated" {
				mcNode.Devices = []cephv1.Device{
					{
						Name:   "vdc",
						Config: map[string]string{"deviceClass": "hdd"},
					},
				}
				if role == "storage-dedicated" {
					mcNode.Roles = []string{}
				}
			}
			mcNodes = append(mcNodes, mcNode)
		}
	}

	mkeCD := &cephlcmv1alpha1.CephDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mke-cephcluster",
			Namespace: f.TF.ManagedCluster.LcmNamespace,
		},
		Spec: cephlcmv1alpha1.CephDeploymentSpec{
			Network: cephlcmv1alpha1.CephNetworkSpec{
				ClusterNet: testConfig["clusterNet"],
				PublicNet:  testConfig["publicNet"],
			},
			ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
				Rgw: cephlcmv1alpha1.CephRGW{
					Name: "rgw-store",
					Gateway: cephlcmv1alpha1.CephRGWGateway{
						Instances:  1,
						Port:       80,
						SecurePort: 8443,
					},
					MetadataPool: &cephlcmv1alpha1.CephPoolSpec{
						DeviceClass: "hdd",
						Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
							Size: 3,
						},
					},
					DataPool: &cephlcmv1alpha1.CephPoolSpec{
						DeviceClass: "hdd",
						ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
							CodingChunks: 1,
							DataChunks:   2,
						},
					},
				},
			},
			Pools: []cephlcmv1alpha1.CephPool{
				{
					Name:             "kubernetes",
					StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{Default: true},
					Role:             "kubernetes",
					CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass: "hdd",
						Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
							Size: 3,
						},
					},
				},
			},
			Nodes: mcNodes,
		},
	}

	f.Step(t, "Creating CephDeployment")
	err = f.TF.ManagedCluster.CreateCephDeployment(mkeCD)
	if err != nil {
		t.Fatalf("failed to create CephDeployment %s/%s for MKE cluster: %v", mkeCD.Namespace, mkeCD.Name, err)
	}

	f.Step(t, "Checking CephDeployment and CephDeploymentHealth are fully healthy")
	err = f.WaitForStatusReady(mkeCD.Name)
	if err != nil {
		t.Fatalf("failed to wait readiness: %v", err)
	}
}
