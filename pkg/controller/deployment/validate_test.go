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

func TestCephDeploymentNodesValidate(t *testing.T) {
	tests := []struct {
		name                string
		inputResources      map[string]runtime.Object
		cephDpl             *cephlcmv1alpha1.CephDeployment
		expectedErrorOutput string
	}{
		{
			name:                "failed to get node list",
			inputResources:      map[string]runtime.Object{},
			expectedErrorOutput: "failed to get node list: failed to list nodes",
		},
		{
			name:    "nodes from cephdeployment spec present in k8s cluster",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"nodes": unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			},
		},
		{
			name:                "some nodes from cephdeployment spec not present in k8s cluster",
			cephDpl:             &unitinputs.BaseCephDeployment,
			inputResources:      map[string]runtime.Object{"nodes": &v1.NodeList{}},
			expectedErrorOutput: "The following nodes are present in CephDeployment spec but not present in k8s cluster node list: node-1,node-2,node-3",
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
			err := c.cephDeploymentNodesValidate()
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

func TestCephSharedFilesystemValidate(t *testing.T) {
	cephDplOk := unitinputs.BaseCephDeployment.DeepCopy()
	cephDplOk.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
	cephDplExtneralOk := unitinputs.CephDeployExternal.DeepCopy()
	cephDplExtneralOk.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
	node := cephDplOk.Spec.Nodes[0]
	node.Roles = append(node.Roles, "mds")
	cephDplOk.Spec.Nodes[0] = node
	cephDplOkMultipleCephFs := cephDplOk.DeepCopy()
	cephDplOkMultipleCephFs.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemMultiple
	cephDplNotOk1 := unitinputs.BaseCephDeployment.DeepCopy()
	cephDplNotOk1.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{
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
	cephDplNotOk2 := cephDplOk.DeepCopy()
	cephDplNotOk2.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{
		Filesystems: []cephlcmv1alpha1.CephFilesystem{
			{
				Name: "test-cephfs",
				FsSpec: runtime.RawExtension{
					Raw: []byte(string("{}")),
				},
			},
		},
	}
	cephDplOkWithExtraClasses := cephDplOk.DeepCopy()
	cephDplOkWithExtraClasses.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"some-custom-class"}}
	cephfs := cephDplOkWithExtraClasses.Spec.SharedFilesystem.Filesystems[0]
	castedFsSpec, _ := cephfs.GetSpec()
	dataPool := castedFsSpec.DataPools[0].DeepCopy()
	dataPool.DeviceClass = "some-custom-class"
	dataPool.Name = "custom-pool"
	castedFsSpec.DataPools = append(castedFsSpec.DataPools, *dataPool)
	cephDplOkWithExtraClasses.Spec.SharedFilesystem.Filesystems[0].FsSpec.Raw = unitinputs.ConvertStructToRaw(castedFsSpec)

	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		expectedErrors []string
	}{
		{
			name:           "validation ok",
			cephDpl:        cephDplOk,
			expectedErrors: make([]string, 0),
		},
		{
			name:           "validation ok with extra device class",
			cephDpl:        cephDplOkWithExtraClasses,
			expectedErrors: make([]string, 0),
		},
		{
			name:           "validation ok - multiple cephfs specified",
			cephDpl:        cephDplOkMultipleCephFs,
			expectedErrors: make([]string, 0),
		},
		{
			name:    "validation failed - wrong spec",
			cephDpl: cephDplNotOk1,
			expectedErrors: []string{
				"metadataPool for CephFS rook-ceph/test-cephfs must use replication only",
				"metadataPool for CephFS rook-ceph/test-cephfs has no deviceClass specified (valid options are: [hdd nvme ssd])",
				"metadataPool for CephFS rook-ceph/test-cephfs contains prohibited 'osd' failureDomain",
				"dataPool some-pool-name for CephFS rook-ceph/test-cephfs has unknown deviceClass 'some-custom-device-class' (valid options are: [hdd nvme ssd])",
				"dataPool some-pool-name for CephFS rook-ceph/test-cephfs contains prohibited 'osd' failureDomain",
				"dataPool some-pool-name will be used as default for CephFS rook-ceph/test-cephfs and must use replication only",
				"dataPool some-pool-name-2 for CephFS rook-ceph/test-cephfs has no deviceClass specified (valid options are: [hdd nvme ssd])",
				"dataPool some-pool-name-2 for CephFS rook-ceph/test-cephfs has no neither replication or erasureCoded sections specified",
				"not enough 'mds' roles specified in nodes spec, CephFS rook-ceph/test-cephfs requires at least 1",
			},
		},
		{
			name:    "validation failed - pools in spec",
			cephDpl: cephDplNotOk2,
			expectedErrors: []string{
				"metadataPool for CephFS rook-ceph/test-cephfs must use replication only",
				"dataPools sections for CephFS rook-ceph/test-cephfs has no data pools defined",
			},
		},
		{
			name:           "validation ok with insufficient number of 'mds' roles for external cluster",
			cephDpl:        cephDplExtneralOk,
			expectedErrors: make([]string, 0),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			errors := cephSharedFilesystemValidate(test.cephDpl, "rook-ceph", c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable)
			assert.Equal(t, test.expectedErrors, errors)
		})
	}
}

func TestOpenstackPoolsValidate(t *testing.T) {
	cephDplMissedPools := unitinputs.CephDeployMosk.DeepCopy()
	newPools := cephDplMissedPools.Spec.BlockStorage.Pools[:len(cephDplMissedPools.Spec.BlockStorage.Pools)-1]
	cephDplMissedPools.Spec.BlockStorage.Pools = newPools
	cephDplExtraPools := unitinputs.CephDeployMosk.DeepCopy()
	cephDplExtraPools.Spec.BlockStorage.Pools = append(cephDplExtraPools.Spec.BlockStorage.Pools, cephDplExtraPools.Spec.BlockStorage.Pools...)

	tests := []struct {
		name          string
		cephDpl       *cephlcmv1alpha1.CephDeployment
		expectedError string
	}{
		{
			name:    "no openstack pools",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
		},
		{
			name:    "openstack pools present",
			cephDpl: unitinputs.CephDeployMosk.DeepCopy(),
		},
		{
			name:          "some openstack pools missed",
			cephDpl:       cephDplMissedPools,
			expectedError: "Not all Openstack required pools was found: missed [backup]. Or it should not be Openstack pools at all",
		},
		{
			name:          "extra openstack pools specified",
			cephDpl:       cephDplExtraPools,
			expectedError: "Detected incorrent number of OpenStack Pools with roles: [vms images backup] - allowed to be specified only once",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := openstackPoolsValidate(test.cephDpl)
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, test.expectedError, err.Error())
			}
		})
	}
}

func TestValidateObjectStorage(t *testing.T) {
	cephDplRgwEmptyPoolTypes := unitinputs.CephDeployNonMosk.DeepCopy()
	rgwCastedEmpty, _ := cephDplRgwEmptyPoolTypes.Spec.ObjectStorage.Rgws[0].GetSpec()
	rgwCastedEmpty.DataPool.Replicated.Size = 0
	rgwCastedEmpty.DataPool.ErasureCoded.CodingChunks = 0
	rgwCastedEmpty.DataPool.ErasureCoded.DataChunks = 0
	rgwCastedEmpty.MetadataPool.Replicated.Size = 0
	rgwCastedEmpty.MetadataPool.ErasureCoded.CodingChunks = 0
	rgwCastedEmpty.MetadataPool.ErasureCoded.DataChunks = 0
	cephDplRgwEmptyPoolTypes.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCastedEmpty)

	cephDplRgwWrongPoolTypes := unitinputs.CephDeployNonMosk.DeepCopy()
	rgwCastedWrong, _ := cephDplRgwWrongPoolTypes.Spec.ObjectStorage.Rgws[0].GetSpec()
	rgwCastedWrong.DataPool.ErasureCoded.CodingChunks = 0
	rgwCastedWrong.DataPool.ErasureCoded.DataChunks = 1
	rgwCastedWrong.MetadataPool.Replicated.Size = 1
	rgwCastedWrong.MetadataPool.DeviceClass = ""
	cephDplRgwWrongPoolTypes.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCastedWrong)

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

	cephDplWithRgwRoles := unitinputs.CephDeployNonMosk.DeepCopy()
	node = cephDplWithRgwRoles.Spec.Nodes[0]
	node.Roles = append(node.Roles, "rgw")
	cephDplWithRgwRoles.Spec.Nodes[0] = node

	tests := []struct {
		name          string
		cephDpl       *cephlcmv1alpha1.CephDeployment
		expectedError string
	}{
		{
			name:    "no object storage issues",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
		},
		{
			name:    "no object storage issues with custom classes and roles",
			cephDpl: cephDplWithExtraOpts,
		},
		{
			name:          "no pool type for data pool and for metadata pool",
			cephDpl:       cephDplRgwEmptyPoolTypes,
			expectedError: "ObjectStorage section is incorrect: rgw metadata pool must be only replicated,rgw data pool should be either replicated or erasureCoded",
		},
		{
			name:          "incorrect ec params for data pool and metadata pool has no device class provided",
			cephDpl:       cephDplRgwWrongPoolTypes,
			expectedError: "ObjectStorage section is incorrect: rgw metadata pool has no deviceClass specified (valid options are: [hdd nvme ssd]),erasureCoded rgw data pool needs dataChunks set to at least 2,erasureCoded rgw data pool needs dataChunks set to at least 1",
		},
		{
			name:          "not enough rgw roles",
			cephDpl:       cephDplWithRgwRoles,
			expectedError: "not enough 'rgw' roles specified in nodes spec, ObjectStorage section requires at least 2",
		},
		{
			name: "osd failure domain, failure",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				rgwCasted, _ := cd.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.DataPool.FailureDomain = "osd"
				rgwCasted.MetadataPool.FailureDomain = "osd"
				cd.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: rgw metadata pool contains prohibited 'osd' failureDomain,rgw data pool contains prohibited 'osd' failureDomain",
		},
		{
			name:    "multisite rgw correct",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
		},
		{
			name: "rgw has a wrong zone",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				rgwCasted, _ := cd.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Zone.Name = "fake"
				cd.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: incorrect rgw configuration, specified zone 'fake' is not found",
		},
		{
			name: "rgw zone has a wrong zonegroup",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zoneCasted, _ := cd.Spec.ObjectStorage.Zones[0].GetSpec()
				zoneCasted.ZoneGroup = "fake"
				cd.Spec.ObjectStorage.Zones[0].Spec.Raw = unitinputs.ConvertStructToRaw(zoneCasted)
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: incorrect zone configuration, specified zonegroup 'fake' is not found",
		},
		{
			name: "rgw zonegroup has a wrong realm",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zoneGroupCasted, _ := cd.Spec.ObjectStorage.Zonegroups[0].GetSpec()
				zoneGroupCasted.Realm = "fake"
				cd.Spec.ObjectStorage.Zonegroups[0].Spec.Raw = unitinputs.ConvertStructToRaw(zoneGroupCasted)
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: incorrect zonegroup configuration, specified realm 'fake' is not found",
		},
		{
			name: "rgw has a wrong zone pools config",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zoneCasted, _ := cd.Spec.ObjectStorage.Zones[0].GetSpec()
				zoneCasted.DataPool.ErasureCoded.CodingChunks = 0
				zoneCasted.DataPool.ErasureCoded.DataChunks = 0
				zoneCasted.MetadataPool.Replicated.Size = 0
				cd.Spec.ObjectStorage.Zones[0].Spec.Raw = unitinputs.ConvertStructToRaw(zoneCasted)
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: zone 'secondary-zone1' metadata pool must be only replicated,zone 'secondary-zone1' data pool should be either replicated or erasureCoded",
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
			expectedError: "ObjectStorage section is incorrect: more than one zone specified, but currently supported only one zone per cluster,more than one zonegroup specified, but currently supported only one zonegroup per cluster,more than one realm specified, but currently supported only one realm per cluster",
		},
		{
			name: "external rgw, but contains rgw pools specs",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployExternalRgw.DeepCopy()
				rgwCasted, _ := cd.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.DataPool.Replicated.Size = 1
				rgwCasted.MetadataPool.Replicated.Size = 1
				cd.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: rgw in external mode, pools (metadata and data) specification is not allowed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			err = validateObjectStorage(test.cephDpl, c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable)
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, test.expectedError, err.Error())
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		cephDpl  *cephlcmv1alpha1.CephDeployment
		nodeList *v1.NodeList
		expected cephlcmv1alpha1.CephDeploymentValidation
	}{
		{
			name:     "validate non-mosk cephdeployment, success",
			cephDpl:  unitinputs.CephDeployNonMosk.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 10,
			},
		},
		{
			name:     "validate mosk cephdeployment, success",
			cephDpl:  unitinputs.CephDeployMosk.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
		{
			name:     "validate rbdmirror cephdeployment, success",
			cephDpl:  unitinputs.CephDeployEnsureRbdMirror.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
		{
			name:     "validate external cephdeployment, success",
			cephDpl:  unitinputs.CephDeployExternal.DeepCopy(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 0,
			},
		},
		{
			name: "validate pool has no deviceClass and no default, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				castedSpec, _ := cd.Spec.BlockStorage.Pools[0].GetSpec()
				castedSpec.DeviceClass = ""
				cd.Spec.BlockStorage.Pools[0].PoolSpec.Raw = unitinputs.ConvertStructToRaw(castedSpec)
				cd.Spec.BlockStorage.Pools[0].StorageClassOpts.Default = false
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"CephDeployment pool pool1 has no deviceClass specified (valid options are: [hdd nvme ssd])",
					"CephDeployment has no default pool specified",
				},
			},
		},
		{
			name: "validate pool has custom deviceClass",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				castedSpec, _ := cd.Spec.BlockStorage.Pools[0].GetSpec()
				castedSpec.DeviceClass = "some-custom-class"
				cd.Spec.BlockStorage.Pools[0].PoolSpec.Raw = unitinputs.ConvertStructToRaw(castedSpec)
				cd.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"some-custom-class"}}
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 10,
			},
		},
		{
			name: "validate pool has neither replicated nor erasure coded, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.BlockStorage.Pools[0].PoolSpec.Raw = unitinputs.ConvertStructToRaw(cephv1.PoolSpec{DeviceClass: "hdd"})
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"CephDeployment pool pool1 spec should contain either replicated or erasureCoded spec",
				},
			},
		},
		{
			name: "validate pool has incorrect reclaimPolicy, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.BlockStorage.Pools[0].StorageClassOpts.ReclaimPolicy = "Fake"
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"CephDeployment pool pool1 spec contains invalid reclaimPolicy 'Fake', valid are: [Retain Delete]",
				},
			},
		},
		{
			name: "validate pool has osd failureDomain, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				castedSpec, _ := cd.Spec.BlockStorage.Pools[0].GetSpec()
				castedSpec.FailureDomain = "osd"
				cd.Spec.BlockStorage.Pools[0].PoolSpec.Raw = unitinputs.ConvertStructToRaw(castedSpec)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"CephDeployment pool pool1 spec contains prohibited 'osd' failureDomain",
				},
			},
		},
		{
			name: "validate pool has osd failureDomain but one node, success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.BaseCephDeployment.DeepCopy()
				pool := unitinputs.CephDeployPoolReplicated.DeepCopy()
				cd.Spec.Nodes = []cephlcmv1alpha1.CephDeploymentNode{cd.Spec.Nodes[0]}
				castedSpec, _ := pool.GetSpec()
				castedSpec.FailureDomain = "osd"
				pool.PoolSpec.Raw = unitinputs.ConvertStructToRaw(castedSpec)
				cd.Spec.BlockStorage = &cephlcmv1alpha1.CephBlockStorage{
					Pools: []cephlcmv1alpha1.CephPool{*pool},
				}
				cd.Generation = int64(10)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 10,
			},
		},
		{
			name: "validate incorrect nodes spec, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.BlockStorage.Pools = append(cd.Spec.BlockStorage.Pools, cd.Spec.BlockStorage.Pools[0])
				cd.Spec.Nodes = unitinputs.CephNodesExtendedInvalid
				cd.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"some-custom-class"}}
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"CephDeployment has multiple default pools specified",
					"failed to parse config parameter 'osdsPerDevice' for node 'node-1': strconv.Atoi: parsing \"3.5\": invalid syntax",
					"device 'sdb' on node 'node-1' has no deviceClass specified (valid options are: [hdd nvme ssd some-custom-class])",
					"CephDeployment node spec for node 'node-1' contains invalid crush topology key 'datecenter'. Valid are: chassis, datacenter, pdu, rack, region, room, row, zone",
					"failed to parse config parameter 'osdsPerDevice' for device 'sda' from node 'node-3': strconv.Atoi: parsing \"3.5\": invalid syntax",
					"device 'sda' on node 'node-3' has unknown deviceClass 'unknown-class' (valid options are: [hdd nvme ssd some-custom-class])",
					"detected using 'useAllDevices' for 'node-5' node item, which is not supported",
					"deviceClass is not specified for 'node-6' node item, but it is required",
					"CephDeployment monitors (roles 'mon') count 4 is even, but should be odd for a healthy quorum",
					"no 'mgr' roles specified, required at least one",
					"The following nodes are present in CephDeployment spec but not present in k8s cluster node list: node-4,node-5,node-6",
					"not enough 'mds' roles specified in nodes spec, CephFS rook-ceph/test-cephfs requires at least 1",
				},
			},
		},
		{
			name: "validate insufficient number of openstack pools, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMosk.DeepCopy()
				cd.Spec.BlockStorage.Pools = cd.Spec.BlockStorage.Pools[:len(cd.Spec.BlockStorage.Pools)-1]
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 0,
				Messages: []string{
					"Not all Openstack required pools was found: missed [backup]. Or it should not be Openstack pools at all",
				},
			},
		},
		{
			name: "validate incorrect object storage, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				rgwCasted, _ := cd.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.DataPool.Replicated = cephv1.ReplicatedSpec{Size: 3}
				cd.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages:                []string{"ObjectStorage section is incorrect: rgw data pool should be either replicated or erasureCoded"},
			},
		},
		{
			name: "validate incorrect cephfs, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				fsCasted, _ := cd.Spec.SharedFilesystem.Filesystems[0].GetSpec()
				fsCasted.DataPools = nil
				cd.Spec.SharedFilesystem.Filesystems[0].FsSpec.Raw = unitinputs.ConvertStructToRaw(fsCasted)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"dataPools sections for CephFS rook-ceph/test-cephfs has no data pools defined",
				},
			},
		},
		{
			name: "validate network section not specified, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Cluster.Raw = unitinputs.ConvertStructToRaw(cephv1.ClusterSpec{})
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages:                []string{"network addressRanges parameter is not specified"},
			},
		},
		{
			name: "validate incorrect network section, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Cluster.Raw = unitinputs.ConvertStructToRaw(
					cephv1.ClusterSpec{
						Network: cephv1.NetworkSpec{
							AddressRanges: &cephv1.AddressRangesSpec{
								Public:  []cephv1.CIDR{cephv1.CIDR("0.0.0.0/0")},
								Cluster: []cephv1.CIDR{cephv1.CIDR("0.0.0.0/0")},
							},
						},
					},
				)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"network address ranges public parameter should not be empty or contain range 0.0.0.0",
					"network address ranges cluster parameter should not be empty or contain range 0.0.0.0",
				},
			},
		},
		{
			name: "validate networks not specified, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Cluster.Raw = unitinputs.ConvertStructToRaw(
					cephv1.ClusterSpec{
						Network: cephv1.NetworkSpec{
							AddressRanges: &cephv1.AddressRangesSpec{},
						},
					},
				)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"network addressRanges public parameter is empty",
					"network addressRanges cluster parameter is empty",
				},
			},
		},
		{
			name: "validate incorrect network provider, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Cluster.Raw = unitinputs.ConvertStructToRaw(
					cephv1.ClusterSpec{
						Network: cephv1.NetworkSpec{
							Provider: "local",
						},
					},
				)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages:                []string{"network provider parameter should be empty or equals 'host' or 'multus'"},
			},
		},
		{
			name: "validate empty multus network params, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Cluster.Raw = unitinputs.ConvertStructToRaw(
					cephv1.ClusterSpec{
						Network: cephv1.NetworkSpec{
							Provider: "multus",
							AddressRanges: &cephv1.AddressRangesSpec{
								Public:  []cephv1.CIDR{cephv1.CIDR("10.0.0.0/16")},
								Cluster: []cephv1.CIDR{cephv1.CIDR("10.0.0.0/16")},
							},
						},
					},
				)
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"network.selector public and/or cluster parameters should not be empty for provider 'multus'",
				},
			},
		},
		{
			name: "validate correct multus network, success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Network = &cephlcmv1alpha1.CephNetworkSpec{
					Provider: "multus",
					Selector: map[cephv1.CephNetworkType]string{
						cephv1.CephNetworkPublic:  "127.0.0.1/24",
						cephv1.CephNetworkCluster: "127.0.0.1/24",
					},
				}
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationSucceed,
				LastValidatedGeneration: 10,
			},
		},
		{
			name: "validate incorrect rbdmirror cephdeployment, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				cd.Spec.RBDMirror.Peers = append(cd.Spec.RBDMirror.Peers, cephlcmv1alpha1.CephRBDMirrorSecret{
					Site:  "mirror2",
					Token: "fake-token",
					Pools: []string{"pool-1", "pool-2"},
				})
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 0,
				Messages: []string{
					"Multiple RBD Peers aren't supported yet",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, map[string]runtime.Object{"nodes": test.nodeList}, nil)

			err := c.castExtensions()
			assert.Nil(t, err)

			actual := c.validate()
			assert.Equal(t, test.expected, actual)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}
