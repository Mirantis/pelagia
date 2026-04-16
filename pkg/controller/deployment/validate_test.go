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
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestValidateNetworkSpec(t *testing.T) {
	tests := []struct {
		name           string
		networkSpec    cephv1.NetworkSpec
		expectedIssues []string
	}{
		{
			name:           "no address ranges provided",
			networkSpec:    cephv1.NetworkSpec{},
			expectedIssues: []string{"cluster network addressRanges parameter is not specified"},
		},
		{
			name:        "address ranges are not specified",
			networkSpec: cephv1.NetworkSpec{AddressRanges: &cephv1.AddressRangesSpec{}},
			expectedIssues: []string{
				"cluster network addressRanges public parameter not specified",
				"cluster network addressRanges cluster parameter not specified",
			},
		},
		{
			name: "empty ranges provided",
			networkSpec: cephv1.NetworkSpec{
				AddressRanges: &cephv1.AddressRangesSpec{
					Public:  []cephv1.CIDR{cephv1.CIDR("")},
					Cluster: []cephv1.CIDR{cephv1.CIDR("")},
				},
			},
			expectedIssues: []string{
				"cluster network address ranges public parameter should not be empty or contain range 0.0.0.0",
				"cluster network address ranges cluster parameter should not be empty or contain range 0.0.0.0",
			},
		},
		{
			name: "0.0.0.0 ranges provided",
			networkSpec: cephv1.NetworkSpec{
				AddressRanges: &cephv1.AddressRangesSpec{
					Public:  []cephv1.CIDR{cephv1.CIDR("0.0.0.0/0")},
					Cluster: []cephv1.CIDR{cephv1.CIDR("0.0.0.0/0")},
				},
			},
			expectedIssues: []string{
				"cluster network address ranges public parameter should not be empty or contain range 0.0.0.0",
				"cluster network address ranges cluster parameter should not be empty or contain range 0.0.0.0",
			},
		},
		{
			name: "multus network selector is not provided",
			networkSpec: cephv1.NetworkSpec{
				Provider: "multus",
				AddressRanges: &cephv1.AddressRangesSpec{
					Public:  []cephv1.CIDR{cephv1.CIDR("10.0.0.0/16")},
					Cluster: []cephv1.CIDR{cephv1.CIDR("10.0.0.0/16")},
				},
			},
			expectedIssues: []string{
				"cluster network public/cluster selector parameter(s) should not be empty for 'multus' provider",
			},
		},
		{
			name: "unknown network provider set",
			networkSpec: cephv1.NetworkSpec{
				Provider: "custom",
			},
			expectedIssues: []string{"cluster network provider parameter should be empty or equals 'host' or 'multus'"},
		},
		{
			name: "host network spec ok",
			networkSpec: func() cephv1.NetworkSpec {
				spec, _ := unitinputs.BaseCephDeployment.Spec.Cluster.GetSpec()
				return spec.Network
			}(),
			expectedIssues: []string{},
		},
		{
			name: "multus network spec ok",
			networkSpec: func() cephv1.NetworkSpec {
				spec, _ := unitinputs.BaseCephDeploymentMultus.Spec.Cluster.GetSpec()
				return spec.Network
			}(),
			expectedIssues: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := validateNetworkSpec(test.networkSpec)
			assert.Equal(t, test.expectedIssues, issues)
		})
	}
}

func TestValidateClusterNodes(t *testing.T) {
	tests := []struct {
		name                string
		inputResources      map[string]runtime.Object
		cephDpl             *cephlcmv1alpha1.CephDeployment
		expectedErrorOutput string
	}{
		{
			name:                "failed to get node list",
			inputResources:      map[string]runtime.Object{},
			expectedErrorOutput: "failed to list nodes",
		},
		{
			name:                "some nodes from cephdeployment spec not present in k8s cluster",
			cephDpl:             &unitinputs.BaseCephDeployment,
			inputResources:      map[string]runtime.Object{"nodes": &v1.NodeList{}},
			expectedErrorOutput: "found nodes present in spec, but not exist among k8s cluster nodes: node-1,node-2,node-3",
		},
		{
			name:    "nodes from cephdeployment spec present in k8s cluster",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"nodes": unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, test.inputResources, nil)

			if _, ok := test.inputResources["nodes"]; ok {
				expanded, err := lcmcommon.GetExpandedCephDeploymentNodeList(c.context, c.api.Client, test.cephDpl.Spec)
				assert.Nil(t, err)
				c.cdConfig.nodesListExpanded = expanded
			}
			err := c.validateClusterNodes()
			if test.expectedErrorOutput != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedErrorOutput, err.Error())
			} else {
				assert.Nil(t, err)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestValidateNodesSpec(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		expectedIssues []string
	}{
		{
			name: "validate incorrect nodes spec #1",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: unitinputs.BaseCephDeployment.Spec.Cluster.DeepCopy(),
					Nodes: []cephlcmv1alpha1.CephDeploymentNode{
						{
							Node: cephv1.Node{
								Name:      "node-1",
								Selection: cephv1.Selection{UseAllDevices: &[]bool{true}[0]},
							},
						},
						{
							Node: cephv1.Node{
								Name:      "node-2",
								Selection: cephv1.Selection{VolumeClaimTemplates: []cephv1.VolumeClaimTemplate{{}}},
							},
						},
						{
							NodeGroup: []string{"node-3", "node-4"},
							Crush:     map[string]string{"fake": "value"},
							Node:      cephv1.Node{Name: "node-group-1"},
						},
						{
							Node: cephv1.Node{
								Name:      "node-5",
								Selection: cephv1.Selection{DeviceFilter: "vdf"},
								Config:    map[string]string{"deviceClass": "unknownclass"},
							},
							Crush: map[string]string{"rack": "value"},
						},
						{
							Node: cephv1.Node{
								Name:      "node-6",
								Selection: cephv1.Selection{DeviceFilter: "vdf"},
								Config:    map[string]string{"osdsPerDevice": "aas"},
							},
							Crush: map[string]string{"rack": "value"},
						},
						{
							Node: cephv1.Node{
								Name:      "node-7",
								Selection: cephv1.Selection{DeviceFilter: "vdf"},
							},
							Crush: map[string]string{"rack": "value"},
						},
					},
				},
			},
			expectedIssues: []string{
				"found 'useAllDevices' field for nodes item node 'node-1', which is not supported, remove field",
				"found 'volumeClaimTemplates' field for nodes item node 'node-2', which is not supported, remove field",
				"nodes item nodeGroup 'node-group-1' contains invalid crush topology key 'fake'. Valid are: chassis, datacenter, pdu, rack, region, room, row, zone",
				"nodes item node 'node-5' config has unknown deviceClass 'unknownclass' (default valid options are: [hdd nvme ssd], either specify custom classes)",
				"failed to parse config parameter 'osdsPerDevice' from nodes item node 'node-6': strconv.Atoi: parsing \"aas\": invalid syntax",
				"config parameter 'deviceClass' is not specified for nodes item node 'node-7', but it is required",
				"no nodes with 'mon' roles specified",
				"no nodes with 'mgr' roles specified, required at least one",
			},
		},
		{
			name: "validate incorrect nodes spec #2",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: unitinputs.BaseCephDeployment.Spec.Cluster.DeepCopy(),
					ExtraOpts: &cephlcmv1alpha1.CephDeploymentExtraOpts{
						CustomDeviceClasses: []string{"custom"},
					},
					Nodes: []cephlcmv1alpha1.CephDeploymentNode{
						{
							Node: cephv1.Node{
								Name: "node-1",
								Selection: cephv1.Selection{
									Devices: []cephv1.Device{
										{
											Name:   "sdb",
											Config: map[string]string{"osdsPerDevice": "ss"},
										},
									},
								},
							},
							Roles: []string{"mon", "mgr"},
						},
						{
							Node: cephv1.Node{
								Name: "node-2",
								Selection: cephv1.Selection{
									Devices: []cephv1.Device{
										{
											FullPath: "/dev/sdb",
											Config:   map[string]string{"deviceClass": "fake"},
										},
									},
								},
							},
							Roles: []string{"mon"},
						},
						{
							Node: cephv1.Node{
								Name:   "node-3",
								Config: map[string]string{"deviceClass": "custom"},
								Selection: cephv1.Selection{
									Devices: []cephv1.Device{{Name: "sdb"}},
								},
							},
							Crush: map[string]string{"rack": "value"},
						},
						{
							NodesByLabel: "some-label=some-value",
							Node: cephv1.Node{
								Name: "labeled-nodes",
								Selection: cephv1.Selection{
									Devices: []cephv1.Device{{Config: map[string]string{"deviceClass": "custom"}}},
								},
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"failed to parse config parameter 'osdsPerDevice' for device 'sdb' from node 'node-1': strconv.Atoi: parsing \"ss\": invalid syntax",
				"config parameter 'deviceClass' is not specified for device 'sdb' from nodes item node 'node-1', but it is required",
				"device '/dev/sdb' from nodes item node 'node-2' has unknown deviceClass 'fake' (default valid options are: [hdd nvme ssd custom], either specify custom classes)",
				"nodes item nodeGroup 'labeled-nodes' has device without name or fullpath specified",
				"monitor nodes in spec (with roles 'mon') count is 2, but should be odd for a healthy quorum",
			},
		},
		{
			name:           "validate correct nodes #1",
			cephDpl:        unitinputs.BaseCephDeployment.DeepCopy(),
			expectedIssues: []string{},
		},
		{
			name: "validate correct nodes #2",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: unitinputs.BaseCephDeployment.Spec.Cluster.DeepCopy(),
					ExtraOpts: &cephlcmv1alpha1.CephDeploymentExtraOpts{
						CustomDeviceClasses: []string{"custom"},
					},
					Nodes: []cephlcmv1alpha1.CephDeploymentNode{
						{
							Node: cephv1.Node{
								Name:      "node-1",
								Selection: cephv1.Selection{DeviceFilter: "vdf"},
								Config:    map[string]string{"osdsPerDevice": "1", "deviceClass": "hdd"},
							},
							Crush: map[string]string{"rack": "value"},
							Roles: []string{"mon", "mgr"},
						},
						{
							NodeGroup: []string{"node-2", "node-3"},
							Node: cephv1.Node{
								Name: "node-2",
								Selection: cephv1.Selection{
									Devices: []cephv1.Device{
										{
											FullPath: "/dev/sdb",
											Config:   map[string]string{"deviceClass": "custom", "osdsPerDevice": "2"},
										},
									},
								},
							},
							Roles: []string{"mon"},
						},
					},
				},
			},
			expectedIssues: []string{},
		},
		{
			name: "validate correct single node",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: unitinputs.BaseCephDeployment.Spec.Cluster.DeepCopy(),
					Nodes: []cephlcmv1alpha1.CephDeploymentNode{
						{
							Node: cephv1.Node{
								Name:      "node-1",
								Selection: cephv1.Selection{DeviceFilter: "vdf"},
								Config:    map[string]string{"osdsPerDevice": "1", "deviceClass": "hdd"},
							},
							Crush: map[string]string{"rack": "value"},
							Roles: []string{"mon", "mgr"},
						},
					},
				},
			},
			expectedIssues: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			errs := validateNodesSpec(c.cdConfig.cephDpl, c.cdConfig.nodesListExpanded)
			assert.Equal(t, test.expectedIssues, errs)
		})
	}
}

func TestValidatePoolsSpec(t *testing.T) {
	tests := []struct {
		name            string
		cephDpl         *cephlcmv1alpha1.CephDeployment
		externalCluster bool
		expectedIssues  []string
	}{
		{
			name:           "no block storage section for local",
			cephDpl:        &cephlcmv1alpha1.CephDeployment{},
			expectedIssues: []string{"no block storage pools provided, required at least one"},
		},
		{
			name:            "no block storage section for external",
			cephDpl:         &cephlcmv1alpha1.CephDeployment{},
			externalCluster: true,
		},
		{
			name: "incorrect pools spec for local #1",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
						Pools: []cephlcmv1alpha1.CephPool{
							{
								Name:             "pool-1",
								StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{Default: true},
								PoolSpec: runtime.RawExtension{
									Raw: unitinputs.ConvertStructToRaw(cephv1.PoolSpec{}),
								},
							},
							{
								Name: "pool-2",
								StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{
									Default:       true,
									ReclaimPolicy: "unknown",
								},
								Role:     "volumes",
								PoolSpec: unitinputs.CephDeployPoolReplicated.PoolSpec,
							},
							{
								Name:     "pool-3",
								Role:     "volumes",
								PoolSpec: unitinputs.CephDeployPoolErasureCoded.PoolSpec,
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"pool-1 pool should be either replicated or erasureCoded",
				"multiple default pools specified",
				"pool pool-2 contains invalid reclaimPolicy 'unknown', valid are: [Retain Delete]",
				"found pools with Openstack roles, but missed pools with next roles: [backup images vms] - required to be specified for Openstack",
			},
		},
		{
			name: "incorrect pools spec for local #2",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
						Pools: []cephlcmv1alpha1.CephPool{
							{
								Name:     "pool-1",
								Role:     "images",
								PoolSpec: unitinputs.CephDeployPoolReplicated.PoolSpec,
							},
							{
								Name:     "pool-2",
								Role:     "images",
								PoolSpec: unitinputs.CephDeployPoolErasureCoded.PoolSpec,
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"no default pool specified",
				"found pools with Openstack roles, but missed pools with next roles: [backup vms volumes] - required to be specified for Openstack",
				"found pools with Openstack roles, but pools with roles [images] allowed to be specified only once",
			},
		},
		{
			name: "incorrect pools spec for external",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
						Pools: []cephlcmv1alpha1.CephPool{
							{
								Name: "pool-1",
								Role: "images",
								PoolSpec: runtime.RawExtension{
									Raw: unitinputs.ConvertStructToRaw(cephv1.PoolSpec{}),
								},
							},
						},
					},
				},
			},
			externalCluster: true,
			expectedIssues:  []string{"pool 'pool-1' has no device class specified"},
		},
		{
			name:           "block storage ok for mosk",
			cephDpl:        &unitinputs.CephDeployMosk,
			expectedIssues: []string{},
		},
		{
			name:           "block storage ok for non-mosk",
			cephDpl:        &unitinputs.CephDeployNonMosk,
			expectedIssues: []string{},
		},
		{
			name:            "block storage ok external",
			cephDpl:         &unitinputs.CephDeployExternal,
			externalCluster: true,
			expectedIssues:  []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validatePoolsSpec(test.cephDpl, test.externalCluster, false)
			assert.Equal(t, test.expectedIssues, errs)
		})
	}
}

func TestValidatePoolSpec(t *testing.T) {
	tests := []struct {
		name           string
		poolName       string
		poolSpec       cephv1.PoolSpec
		metadataPool   bool
		singleNode     bool
		specExtraOpts  *cephlcmv1alpha1.CephDeploymentExtraOpts
		expectedIssues []string
	}{
		{
			name:           "metadata pool has no replicated spec",
			poolSpec:       cephv1.PoolSpec{},
			poolName:       "metadata pool-1",
			metadataPool:   true,
			expectedIssues: []string{"metadata pool-1 pool must be only replicated"},
		},
		{
			name:           "no pool type specified",
			poolSpec:       cephv1.PoolSpec{},
			poolName:       "test-pool",
			expectedIssues: []string{"test-pool pool should be either replicated or erasureCoded"},
		},
		{
			name: "both pool types specified",
			poolSpec: cephv1.PoolSpec{
				Replicated:   cephv1.ReplicatedSpec{Size: 2},
				ErasureCoded: cephv1.ErasureCodedSpec{CodingChunks: 1, DataChunks: 1},
			},
			poolName:       "test-pool",
			expectedIssues: []string{"test-pool pool should be either replicated or erasureCoded"},
		},
		{
			name: "ec pool has wrong spec",
			poolSpec: cephv1.PoolSpec{
				ErasureCoded: cephv1.ErasureCodedSpec{CodingChunks: 0, DataChunks: 1},
			},
			poolName: "test-pool",
			expectedIssues: []string{
				"erasureCoded test-pool pool needs dataChunks set to at least 2",
				"erasureCoded test-pool pool needs codingChunks set to at least 1",
				"test-pool pool has no deviceClass specified (default valid options are: [hdd nvme ssd], either specify custom classes)",
			},
		},
		{
			name: "replicated pool has wrong spec",
			poolSpec: cephv1.PoolSpec{
				FailureDomain: "osd",
				DeviceClass:   "fake",
				Replicated:    cephv1.ReplicatedSpec{Size: 2},
			},
			poolName: "test-pool",
			expectedIssues: []string{
				"test-pool pool has unknown deviceClass 'fake' (default valid options are: [hdd nvme ssd], either specify custom classes)",
				"test-pool pool contains prohibited 'osd' failureDomain",
			},
		},
		{
			name: "replicated pool ok",
			poolSpec: func() cephv1.PoolSpec {
				spec, _ := unitinputs.CephDeployPoolReplicated.GetSpec()
				return spec
			}(),
			expectedIssues: []string{},
		},
		{
			name: "replicated pool with custom deviceClass for single node ok",
			poolSpec: cephv1.PoolSpec{
				FailureDomain: "osd",
				DeviceClass:   "custom",
				Replicated:    cephv1.ReplicatedSpec{Size: 1},
			},
			poolName:       "test-pool",
			singleNode:     true,
			specExtraOpts:  &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"custom"}},
			expectedIssues: []string{},
		},
		{
			name: "ec pool ok",
			poolSpec: func() cephv1.PoolSpec {
				spec, _ := unitinputs.CephDeployPoolErasureCoded.GetSpec()
				return spec
			}(),
			expectedIssues: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validatePoolSpec(test.poolSpec, test.metadataPool, test.poolName, test.singleNode, test.specExtraOpts)
			assert.Equal(t, test.expectedIssues, errs)
		})
	}
}

func TestValidateFilesystemSpec(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		expectedErrors []string
	}{
		{
			name:    "no filesystems specified",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
		},
		{
			name: "no datapools for cephfs specified",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployNonMosk.DeepCopy()
				spec, _ := cephDpl.Spec.SharedFilesystem.Filesystems[0].GetSpec()
				spec.DataPools = nil
				cephDpl.Spec.SharedFilesystem.Filesystems[0].FsSpec.Raw = unitinputs.ConvertStructToRaw(spec)
				return cephDpl
			}(),
			expectedErrors: []string{"cephfs 'test-cephfs' has no datapools specified, requires at least one"},
		},
		{
			name: "cephfs wrong spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.BaseCephDeployment.DeepCopy()
				cephDpl.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{
					Filesystems: []cephlcmv1alpha1.CephFilesystem{
						{
							Name: "test-cephfs",
							FsSpec: runtime.RawExtension{
								Raw: unitinputs.ConvertStructToRaw(
									cephv1.FilesystemSpec{
										MetadataPool: cephv1.NamedPoolSpec{
											PoolSpec: cephv1.PoolSpec{FailureDomain: "osd"},
										},
										DataPools: []cephv1.NamedPoolSpec{
											{
												Name: "some-pool-name",
												PoolSpec: cephv1.PoolSpec{
													DeviceClass:   "some-custom-device-class",
													FailureDomain: "osd",
												},
											},
											{
												Name: "some-pool-name-2",
											},
										},
										MetadataServer: cephv1.MetadataServerSpec{
											ActiveCount:   1,
											ActiveStandby: true,
										},
									},
								),
							},
						},
					},
				}
				return cephDpl
			}(),
			expectedErrors: []string{
				"cephfs 'test-cephfs' metadata pool must be only replicated",
				"cephfs 'test-cephfs' data some-pool-name will be used as default and must use replication only",
				"cephfs 'test-cephfs' data some-pool-name-2 pool should be either replicated or erasureCoded",
				"not enough 'mds' roles specified in nodes spec, cephfs test-cephfs requires at least 1, found 0",
			},
		},
		{
			name:           "cephfs ok",
			cephDpl:        unitinputs.CephDeployNonMosk.DeepCopy(),
			expectedErrors: []string{},
		},
		{
			name: "cephfs with custom classes ok",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDplOkWithExtraClasses := unitinputs.CephDeployNonMosk.DeepCopy()
				cephDplOkWithExtraClasses.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"some-custom-class"}}
				cephfs := cephDplOkWithExtraClasses.Spec.SharedFilesystem.Filesystems[0]
				castedFsSpec, _ := cephfs.GetSpec()
				castedFsSpec.MetadataPool.DeviceClass = "some-custom-class"
				dataPool := castedFsSpec.DataPools[0].DeepCopy()
				dataPool.DeviceClass = "some-custom-class"
				dataPool.Name = "custom-pool"
				castedFsSpec.DataPools = append(castedFsSpec.DataPools, *dataPool)
				cephDplOkWithExtraClasses.Spec.SharedFilesystem.Filesystems[0].FsSpec.Raw = unitinputs.ConvertStructToRaw(castedFsSpec)
				return cephDplOkWithExtraClasses
			}(),
			expectedErrors: []string{},
		},
		{
			name:           "cephfs ok for external",
			cephDpl:        unitinputs.CephDeployExternalCephFS.DeepCopy(),
			expectedErrors: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			errors := validateFilesystemSpec(c.cdConfig.cephDpl, c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable)
			assert.Equal(t, test.expectedErrors, errors)
		})
	}
}

func TestValidateObjectStorageSpec(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		expectedIssues []string
	}{
		{
			name:           "no object storage section",
			cephDpl:        unitinputs.BaseCephDeployment.DeepCopy(),
			expectedIssues: []string{},
		},
		{
			name: "rgw external cluster wrong spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployExternal.DeepCopy()
				cephDpl.Spec.ObjectStorage = unitinputs.CephDeployMultisiteRgw.Spec.ObjectStorage.DeepCopy()
				cephDpl.Spec.ObjectStorage.Rgws = []cephlcmv1alpha1.CephObjectStore{
					{
						Name: "external-rgw",
						Spec: runtime.RawExtension{
							Raw: unitinputs.ConvertStructToRaw(
								cephv1.ObjectStoreSpec{
									MetadataPool: unitinputs.CephObjectStoreBase.Spec.MetadataPool,
									DataPool:     unitinputs.CephObjectStoreBase.Spec.DataPool,
								},
							),
						},
					},
				}
				return cephDpl
			}(),
			expectedIssues: []string{
				"cluster is external, rgw realms can't be created",
				"cluster is external, rgw zonegroups can't be created",
				"cluster is external, rgw zones can't be created",
				"rgw 'external-rgw' has no port specified",
				"cluster is external, rgw 'external-rgw' pools (metadata and data) specification is not allowed",
				"external endpoints for rgw 'external-rgw' are not provided",
			},
		},
		{
			name: "multiple zones, realms, zonegroups",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zone2 := cd.Spec.ObjectStorage.Zones[0].DeepCopy()
				zone2.Name = "zone2"
				zonegroup2 := cd.Spec.ObjectStorage.Zonegroups[0].DeepCopy()
				zonegroup2.Name = "zonegroup2"
				realm2 := cd.Spec.ObjectStorage.Realms[0].DeepCopy()
				realm2.Name = "realm2"
				cd.Spec.ObjectStorage.Zones = append(cd.Spec.ObjectStorage.Zones, *zone2)
				cd.Spec.ObjectStorage.Zonegroups = append(cd.Spec.ObjectStorage.Zonegroups, *zonegroup2)
				cd.Spec.ObjectStorage.Realms = append(cd.Spec.ObjectStorage.Realms, *realm2)
				return cd
			}(),
			expectedIssues: []string{
				"more than one realm specified, but currently supported only one realm per cluster",
				"more than one zonegroup specified, but currently supported only one zonegroup per cluster",
				"more than one zone specified, but currently supported only one zone per cluster",
			},
		},
		{
			name: "incorrect spec zones, relams, zonegroups",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				rgwCasted, _ := cd.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Zone.Name = "fake"
				cd.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				zoneCasted, _ := cd.Spec.ObjectStorage.Zones[0].GetSpec()
				zoneCasted.ZoneGroup = "fake"
				zoneCasted.DataPool.ErasureCoded.CodingChunks = 0
				zoneCasted.DataPool.ErasureCoded.DataChunks = 0
				zoneCasted.MetadataPool.Replicated.Size = 0
				cd.Spec.ObjectStorage.Zones[0].Spec.Raw = unitinputs.ConvertStructToRaw(zoneCasted)
				zoneGroupCasted, _ := cd.Spec.ObjectStorage.Zonegroups[0].GetSpec()
				zoneGroupCasted.Realm = "fake"
				cd.Spec.ObjectStorage.Zonegroups[0].Spec.Raw = unitinputs.ConvertStructToRaw(zoneGroupCasted)
				return cd
			}(),
			expectedIssues: []string{
				"zonegroup 'zonegroup1' has specified realm 'fake', which is not specified in spec",
				"zone 'secondary-zone1' has specified zonegroup 'fake', which is not specified in spec",
				"zone 'secondary-zone1' metadata pool must be only replicated",
				"zone 'secondary-zone1' data pool should be either replicated or erasureCoded",
				"incorrect rgw 'rgw-store' configuration, specified zone 'fake' is not found",
			},
		},
		{
			name: "incorrect base rgw and user spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				rgwCasted, _ := cd.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.DataPool.FailureDomain = "osd"
				rgwCasted.MetadataPool.FailureDomain = "osd"
				rgwCasted.Gateway.Instances = 20
				cd.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				userCasted, _ := cd.Spec.ObjectStorage.Users[0].GetSpec()
				userCasted.Store = ""
				cd.Spec.ObjectStorage.Users[0].Spec.Raw = unitinputs.ConvertStructToRaw(userCasted)
				userCasted2, _ := cd.Spec.ObjectStorage.Users[1].GetSpec()
				userCasted2.Store = "fake"
				cd.Spec.ObjectStorage.Users[1].Spec.Raw = unitinputs.ConvertStructToRaw(userCasted2)
				return cd
			}(),
			expectedIssues: []string{
				"rgw 'rgw-store' metadata pool contains prohibited 'osd' failureDomain",
				"rgw 'rgw-store' data pool contains prohibited 'osd' failureDomain",
				"not enough 'rgw' roles specified in nodes spec, rgw 'rgw-store' requires at least 20, found 3",
				"object store user 'fake-user-1' has no related rgw set ('store' field)",
				"object store user 'fake-user-2' has unknown rgw set ('store' field)",
			},
		},
		{
			name:           "no object storage issues",
			cephDpl:        unitinputs.CephDeployNonMosk.DeepCopy(),
			expectedIssues: []string{},
		},
		{
			name: "no object storage issues with custom classes and roles",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDplWithExtraOpts := unitinputs.CephDeployNonMosk.DeepCopy()
				rgwCastedCustomClass, _ := cephDplWithExtraOpts.Spec.ObjectStorage.Rgws[0].GetSpec()
				cephDplWithExtraOpts.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"some-custom-class"}}
				rgwCastedCustomClass.DataPool.DeviceClass = "some-custom-class"
				rgwCastedCustomClass.MetadataPool.DeviceClass = "some-custom-class"
				cephDplWithExtraOpts.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCastedCustomClass)
				node := cephDplWithExtraOpts.Spec.Nodes[0]
				node.Roles = append(node.Roles, "rgw")
				cephDplWithExtraOpts.Spec.Nodes[0] = node
				node = cephDplWithExtraOpts.Spec.Nodes[1]
				node.Roles = append(node.Roles, "rgw")
				cephDplWithExtraOpts.Spec.Nodes[1] = node
				return cephDplWithExtraOpts
			}(),
			expectedIssues: []string{},
		},
		{
			name:           "multisite rgw correct",
			cephDpl:        unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			expectedIssues: []string{},
		},
		{
			name:           "rgw external cluster ok",
			cephDpl:        unitinputs.CephDeployExternal.DeepCopy(),
			expectedIssues: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			errs := validateObjectStorageSpec(c.cdConfig.cephDpl, c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable)
			assert.Equal(t, test.expectedIssues, errs)
		})
	}
}

func TestValidateSpec(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		nodeList       *v1.NodeList
		expectedStatus cephlcmv1alpha1.CephDeploymentValidation
	}{
		{
			name:     "failed to validate nodes in cluster",
			cephDpl:  unitinputs.CephDeployMosk.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 0,
				Messages:                []string{"found nodes present in spec, but not exist among k8s cluster nodes: node-1,node-2,node-3"},
			},
		},
		{
			name: "validate incorrect cephdeployment, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				clusterSpec, _ := cd.Spec.Cluster.GetSpec()
				clusterSpec.Network.AddressRanges = nil
				cd.Spec.Cluster.Raw = unitinputs.ConvertStructToRaw(clusterSpec)
				cd.Spec.BlockStorage.Pools[0].StorageClassOpts.Default = false
				cd.Spec.Nodes[1].Roles = nil
				cd.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
				cd.Spec.ObjectStorage.Zones[0].Name = "fake"
				cd.Spec.RBDMirror = unitinputs.CephDeployEnsureRbdMirror.Spec.RBDMirror.DeepCopy()
				cd.Spec.RBDMirror.Peers = append(cd.Spec.RBDMirror.Peers, cephlcmv1alpha1.CephRBDMirrorSecret{
					Site:  "mirror2",
					Token: "fake-token",
					Pools: []string{"pool-1", "pool-2"},
				})
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3", "node-4", "node-5", "node-6"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 0,
				Messages: []string{
					"cluster network addressRanges parameter is not specified",
					"monitor nodes in spec (with roles 'mon') count is 2, but should be odd for a healthy quorum",
					"multiple RBD Peers aren't supported yet",
					"no default pool specified",
					"not enough 'mds' roles specified in nodes spec, cephfs test-cephfs requires at least 1, found 0",
					"incorrect rgw 'rgw-store' configuration, specified zone 'secondary-zone1' is not found",
				},
			},
		},
		{
			name:     "validate non-mosk cephdeployment, success",
			cephDpl:  unitinputs.CephDeployNonMosk.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 10,
			},
		},
		{
			name:     "validate mosk cephdeployment, success",
			cephDpl:  unitinputs.CephDeployMosk.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
		{
			name:     "validate rbdmirror cephdeployment, success",
			cephDpl:  unitinputs.CephDeployEnsureRbdMirror.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
		{
			name:     "validate external cephdeployment, success",
			cephDpl:  unitinputs.CephDeployExternal.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
		{
			name:     "validate single node cephdeployment, success",
			cephDpl:  unitinputs.CephDeploySingleNode.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 10,
			},
		},
		{
			name:     "validate multisite master cephdeployment, success",
			cephDpl:  unitinputs.CephDeployMultisiteMasterRgw.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
		{
			name:     "validate multisite pull cephdeployment, success",
			cephDpl:  unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expectedStatus: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, map[string]runtime.Object{"nodes": test.nodeList}, nil)

			err := c.castExtensions()
			assert.Nil(t, err)

			actualStatus := c.validateSpec()
			assert.Equal(t, test.expectedStatus, actualStatus)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}
