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
				expanded, err := c.buildExpandedNodeList()
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
		CephFS: []cephlcmv1alpha1.CephFS{
			{
				Name:         "test-cephfs",
				MetadataPool: cephlcmv1alpha1.CephPoolSpec{FailureDomain: "osd"},
				DataPools: []cephlcmv1alpha1.CephFSPool{
					{
						Name: "some-pool-name",
						CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
							DeviceClass:   "some-custom-device-class",
							FailureDomain: "osd",
							ErasureCoded:  &cephlcmv1alpha1.CephPoolErasureCodedSpec{},
						},
					},
					{
						Name: "some-pool-name-2",
					},
				},
				MetadataServer: cephlcmv1alpha1.CephMetadataServer{
					ActiveCount:   1,
					ActiveStandby: true,
				},
			},
		},
	}
	cephDplNotOk2 := cephDplOk.DeepCopy()
	cephDplNotOk2.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{
		CephFS: []cephlcmv1alpha1.CephFS{
			{
				Name: "test-cephfs",
				MetadataPool: cephlcmv1alpha1.CephPoolSpec{
					Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{},
				},
				DataPools: []cephlcmv1alpha1.CephFSPool{},
			},
		},
	}
	cephDplOkWithExtraClasses := cephDplOk.DeepCopy()
	cephDplOkWithExtraClasses.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"some-custom-class"}}
	cephfs := cephDplOkWithExtraClasses.Spec.SharedFilesystem.CephFS[0]
	cephfs.MetadataPool.DeviceClass = "some-custom-class"
	dataPool := cephfs.DataPools[0]
	dataPool.DeviceClass = "some-custom-class"
	cephfs.DataPools[0] = dataPool
	cephDplOkWithExtraClasses.Spec.SharedFilesystem.CephFS[0] = cephfs

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
			name:           "validation failed - no datapools in spec",
			cephDpl:        cephDplNotOk2,
			expectedErrors: []string{"dataPools sections for CephFS rook-ceph/test-cephfs has no data pools defined"},
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
			expanded, err := c.buildExpandedNodeList()
			assert.Nil(t, err)

			errors := cephSharedFilesystemValidate(test.cephDpl, "rook-ceph", expanded)
			assert.Equal(t, test.expectedErrors, errors)
		})
	}
}

func TestOpenstackPoolsValidate(t *testing.T) {
	cephDplMissedPools := unitinputs.CephDeployMosk.DeepCopy()
	newPools := cephDplMissedPools.Spec.Pools[:len(cephDplMissedPools.Spec.Pools)-1]
	cephDplMissedPools.Spec.Pools = newPools
	cephDplExtraPools := unitinputs.CephDeployMosk.DeepCopy()
	cephDplExtraPools.Spec.Pools = append(cephDplExtraPools.Spec.Pools, cephDplExtraPools.Spec.Pools...)

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
	cephDplRgwEmptyPoolTypes.Spec.ObjectStorage.Rgw.DataPool.Replicated = nil
	cephDplRgwEmptyPoolTypes.Spec.ObjectStorage.Rgw.DataPool.ErasureCoded = nil
	cephDplRgwEmptyPoolTypes.Spec.ObjectStorage.Rgw.MetadataPool.Replicated = nil
	cephDplRgwEmptyPoolTypes.Spec.ObjectStorage.Rgw.MetadataPool.ErasureCoded = nil

	cephDplRgwWrongPoolTypes := unitinputs.CephDeployNonMosk.DeepCopy()
	cephDplRgwWrongPoolTypes.Spec.ObjectStorage.Rgw.DataPool = &cephlcmv1alpha1.CephPoolSpec{
		ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{},
		Replicated:   &cephlcmv1alpha1.CephPoolReplicatedSpec{},
	}
	cephDplRgwWrongPoolTypes.Spec.ObjectStorage.Rgw.MetadataPool = &cephlcmv1alpha1.CephPoolSpec{
		ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{},
		Replicated:   &cephlcmv1alpha1.CephPoolReplicatedSpec{},
	}

	cephDplWithExtraOpts := unitinputs.CephDeployNonMosk.DeepCopy()
	cephDplWithExtraOpts.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{CustomDeviceClasses: []string{"some-custom-class"}}
	cephDplWithExtraOpts.Spec.ObjectStorage.Rgw.DataPool.DeviceClass = "some-custom-class"
	cephDplWithExtraOpts.Spec.ObjectStorage.Rgw.MetadataPool.DeviceClass = "some-custom-class"
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
			expectedError: "ObjectStorage section is incorrect: rgw metadata pool must be only replicated,rgw data pool has no pool type specified",
		},
		{
			name:          "both pool types for data pool and wrong for metadata pool has no device class provided",
			cephDpl:       cephDplRgwWrongPoolTypes,
			expectedError: "ObjectStorage section is incorrect: rgw metadata pool must be only replicated,rgw metadata pool has no deviceClass specified (valid options are: [hdd nvme ssd]),rgw data pool must have only one pool type specified,rgw data pool has no deviceClass specified (valid options are: [hdd nvme ssd])",
		},
		{
			name:          "not enough rgw roles",
			cephDpl:       cephDplWithRgwRoles,
			expectedError: "not enough 'rgw' roles specified in nodes spec, ObjectStorage requires at least 2",
		},
		{
			name: "osd failure domain, failure",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.DataPool.FailureDomain = "osd"
				cd.Spec.ObjectStorage.Rgw.MetadataPool.FailureDomain = "osd"
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: rgw metadata pool contains prohibited 'osd' failureDomain,rgw data pool contains prohibited 'osd' failureDomain",
		},
		{
			name: "no metadata and datapool specified",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.DataPool = nil
				cd.Spec.ObjectStorage.Rgw.MetadataPool = nil
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: no rgw metadata/data pool(s) specified",
		},
		{
			name:    "multisite rgw correct",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
		},
		{
			name: "multiste is specified, but rgw has no zone",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.Zone = nil
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: rgw has no specified zone name, but multisite configuration is present",
		},
		{
			name: "multiste is not specified, but rgw has a zone",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				cd.Spec.ObjectStorage.MultiSite = nil
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: rgw has specified zone name, but it is allowed only for multisite configuration, which is not present",
		},
		{
			name: "multiste is specified, but rgw has a wrong zone",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.Zone.Name = "fake"
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: incorrect multisite configuration, specified zone 'fake' is not found",
		},
		{
			name: "multiste is specified, but rgw has a wrong zonegroup",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zone := cd.Spec.ObjectStorage.MultiSite.Zones[0]
				zone.ZoneGroup = "fake"
				cd.Spec.ObjectStorage.MultiSite.Zones[0] = zone
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: incorrect multisite configuration, specified zonegroup 'fake' is not found",
		},
		{
			name: "multiste is specified, but rgw has a wrong realm",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zoneGroup := cd.Spec.ObjectStorage.MultiSite.ZoneGroups[0]
				zoneGroup.Realm = "fake"
				cd.Spec.ObjectStorage.MultiSite.ZoneGroups[0] = zoneGroup
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: incorrect multisite configuration, specified realm 'fake' is not found",
		},
		{
			name: "multiste is specified, but rgw has a wrong zone pools config",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zone := cd.Spec.ObjectStorage.MultiSite.Zones[0]
				zone.DataPool.ErasureCoded = nil
				zone.MetadataPool.Replicated = nil
				cd.Spec.ObjectStorage.MultiSite.Zones[0] = zone
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: rgw metadata pool in zone secondary-zone1 must be only replicated,rgw data pool in zone secondary-zone1 has no pool type specified",
		},
		{
			name: "multiste is specified, but multiple zones, realms, zonegroups",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zone2 := cd.Spec.ObjectStorage.MultiSite.Zones[0].DeepCopy()
				zone2.Name = "zone2"
				zonegroup2 := cd.Spec.ObjectStorage.MultiSite.ZoneGroups[0].DeepCopy()
				zonegroup2.Name = "zonegroup2"
				realm2 := cd.Spec.ObjectStorage.MultiSite.Realms[0].DeepCopy()
				realm2.Name = "realm2"
				cd.Spec.ObjectStorage.MultiSite.Zones = append(cd.Spec.ObjectStorage.MultiSite.Zones, *zone2)
				cd.Spec.ObjectStorage.MultiSite.ZoneGroups = append(cd.Spec.ObjectStorage.MultiSite.ZoneGroups, *zonegroup2)
				cd.Spec.ObjectStorage.MultiSite.Realms = append(cd.Spec.ObjectStorage.MultiSite.Realms, *realm2)
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: more than one zone specified, but currently supported only one zone per cluster,more than one zonegroup specified, but currently supported only one zonegroup per cluster,more than one realm specified, but currently supported only one realm per cluster",
		},
		{
			name: "external rgw, but contains rgw pools specs",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployExternalRgw.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.DataPool = &cephlcmv1alpha1.CephPoolSpec{}
				cd.Spec.ObjectStorage.Rgw.MetadataPool = &cephlcmv1alpha1.CephPoolSpec{}
				return cd
			}(),
			expectedError: "ObjectStorage section is incorrect: rgw in external mode, pools (metadata and data) specification is not allowed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			expanded, err := c.buildExpandedNodeList()
			assert.Nil(t, err)

			err = validateObjectStorage(test.cephDpl, expanded)
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
				cd.Spec.Pools[0].DeviceClass = ""
				cd.Spec.Pools[0].StorageClassOpts.Default = false
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
				cd.Spec.Pools[0].DeviceClass = "some-custom-class"
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
				cd.Spec.Pools[0].Replicated = nil
				cd.Spec.Pools[0].ErasureCoded = nil
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
				cd.Spec.Pools[0].StorageClassOpts.ReclaimPolicy = "Fake"
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
				cd.Spec.Pools[0].FailureDomain = "osd"
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
				pool.FailureDomain = "osd"
				cd.Spec.Pools = []cephlcmv1alpha1.CephPool{*pool}
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
				cd.Spec.Pools = append(cd.Spec.Pools, cd.Spec.Pools[0])
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
				cd.Spec.Pools = cd.Spec.Pools[:len(cd.Spec.Pools)-1]
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
				cd.Spec.ObjectStorage.Rgw.DataPool.Replicated = &cephlcmv1alpha1.CephPoolReplicatedSpec{
					Size: 3,
				}
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"ObjectStorage section is incorrect: rgw data pool must have only one pool type specified",
				},
			},
		},
		{
			name: "validate incorrect cephfs, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.SharedFilesystem.CephFS[0].DataPools = nil
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
			name: "validate incorrect network section, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Network = cephlcmv1alpha1.CephNetworkSpec{
					PublicNet: "0.0.0.0/0",
				}
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"network publicNet parameter contains prohibited 0.0.0.0 range",
					"network clusterNet parameter is empty",
				},
			},
		},
		{
			name: "validate incorrect network provider, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Network = cephlcmv1alpha1.CephNetworkSpec{
					Provider: "local",
				}
				return cd
			}(),
			nodeList: unitinputs.GetOsdNodesList([]string{"node-1", "node-2", "node-3"}),
			expected: cephlcmv1alpha1.CephDeploymentValidation{
				Result:                  cephlcmv1alpha1.ValidationFailed,
				LastValidatedGeneration: 10,
				Messages: []string{
					"network provider parameter should be empty or equals 'host' or 'multus'",
				},
			},
		},
		{
			name: "validate empty multus network params, failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.Network = cephlcmv1alpha1.CephNetworkSpec{
					Provider: "multus",
					Selector: map[cephv1.CephNetworkType]string{},
				}
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
				cd.Spec.Network = cephlcmv1alpha1.CephNetworkSpec{
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

			expanded, err := c.buildExpandedNodeList()
			assert.Nil(t, err)
			c.cdConfig.nodesListExpanded = expanded

			actual := c.validate()
			assert.Equal(t, test.expected, actual)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}
