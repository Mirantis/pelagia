/*
Copyright 2026 Mirantis IT.

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

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestCastExtensions(t *testing.T) {
	tests := []struct {
		name          string
		cephDpl       *cephlcmv1alpha1.CephDeployment
		expectedError string
	}{
		{
			name: "failed to cast cephdeployment spec",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				ObjectMeta: unitinputs.BaseCephDeployment.ObjectMeta,
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: &cephlcmv1alpha1.CephCluster{
						RawExtension: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true"}`)},
					},
					BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
						Pools: []cephlcmv1alpha1.CephPool{
							{
								Name:     "testpool",
								PoolSpec: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true}`)},
							},
						},
					},
					Clients: []cephlcmv1alpha1.CephClient{
						{
							RawExtension: runtime.RawExtension{Raw: []byte(`{"unknonwApiField": 1}`)},
						},
					},
					ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
						Rgws: []cephlcmv1alpha1.CephObjectStore{
							{
								Name: "testrgw",
								Spec: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true}`)},
							},
						},
						Users: []cephlcmv1alpha1.CephObjectStoreUser{
							{
								Name: "testuser",
								Spec: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true}`)},
							},
						},
						Realms: []cephlcmv1alpha1.CephObjectRealm{
							{
								Name: "testrealm",
								Spec: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true}`)},
							},
						},
						Zonegroups: []cephlcmv1alpha1.CephObjectZonegroup{
							{
								Name: "testzonegroup",
								Spec: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true}`)},
							},
						},
						Zones: []cephlcmv1alpha1.CephObjectZone{
							{
								Name: "testzone",
								Spec: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true}`)},
							},
						},
					},
					SharedFilesystem: &cephlcmv1alpha1.CephSharedFilesystem{
						Filesystems: []cephlcmv1alpha1.CephFilesystem{
							{
								Name:   "testfilesystem",
								FsSpec: runtime.RawExtension{Raw: []byte(`{"unknownApiField": true}`)},
							},
						},
					},
					Nodes: unitinputs.CephDeployWithWrongNodes.Spec.Nodes,
				},
			},
			expectedError: "failed to cast spec fields: failed to cast cephdeployment fields to Rook API, failed to cast block storage pool 'testpool' fields to Rook API, failed to cast client #0 to Rook API, failed to cast rgw 'testrgw' to Rook API, failed to cast user 'testuser' to Rook API, failed to cast realm 'testrealm' to Rook API, failed to cast zonegroup 'testzonegroup' to Rook API, failed to cast zone 'testzone' to Rook API, failed to cast ceph filesystem 'testfilesystem' to Rook API, failed to expand nodes list",
		},
		{
			name:    "validate base cephdeployment",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
		},
		{
			name:    "validate singlenode cephdeployment",
			cephDpl: unitinputs.CephDeploySingleNode.DeepCopy(),
		},
		{
			name:    "validate cephdeployment",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
		},
		{
			name:    "validate mosk cephdeployment",
			cephDpl: unitinputs.CephDeployMosk.DeepCopy(),
		},
		{
			name:    "validate mosk cephdeployment",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
		},
		{
			name:    "validate mosk cephdeployment",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
