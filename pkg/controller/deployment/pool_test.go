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

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGeneratePool(t *testing.T) {
	tests := []struct {
		name         string
		cephDpl      cephlcmv1alpha1.CephPool
		expectedPool *cephv1.CephBlockPool
	}{
		{
			name:         "generate ceph block pool standart",
			cephDpl:      unitinputs.CephDeployPoolReplicated,
			expectedPool: &unitinputs.CephBlockPoolReplicated,
		},
		{
			name: "generate ceph block pool with role vms",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPoolRole := unitinputs.CephDeployPoolReplicated.DeepCopy()
				cephDplPoolRole.Role = "vms"
				return *cephDplPoolRole
			}(),
			expectedPool: func() *cephv1.CephBlockPool {
				expectedCephBlockPoolRole := unitinputs.CephBlockPoolReplicated.DeepCopy()
				expectedCephBlockPoolRole.Spec.Replicated.TargetSizeRatio = 0.2
				return expectedCephBlockPoolRole
			}(),
		},
		{
			name: "generate ceph block pool with target ratio",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPoolSpec := unitinputs.CephDeployPoolReplicated.DeepCopy()
				cephDplPoolSpec.Replicated.TargetSizeRatio = 0.1
				return *cephDplPoolSpec
			}(),
			expectedPool: func() *cephv1.CephBlockPool {
				expectedCephBlockPoolSpec := unitinputs.CephBlockPoolReplicated.DeepCopy()
				expectedCephBlockPoolSpec.Spec.Replicated.TargetSizeRatio = 0.1
				return expectedCephBlockPoolSpec
			}(),
		},
		{
			name:         "generate ceph block pool erasure coded",
			cephDpl:      unitinputs.CephDeployPoolErasureCoded,
			expectedPool: &unitinputs.CephBlockPoolErasureCoded,
		},
		{
			name: "generate ceph block pool with full pool name",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPoolUseAsFullName := unitinputs.CephDeployPoolReplicated.DeepCopy()
				cephDplPoolUseAsFullName.UseAsFullName = true
				return *cephDplPoolUseAsFullName
			}(),
			expectedPool: func() *cephv1.CephBlockPool {
				expectedCephBlockPoolUseAsFullName := unitinputs.CephBlockPoolReplicated.DeepCopy()
				expectedCephBlockPoolUseAsFullName.Name = "pool1"
				return expectedCephBlockPoolUseAsFullName
			}(),
		},
		{
			name:         "generate ceph block pool with mirroring",
			cephDpl:      unitinputs.CephDeployPoolMirroring,
			expectedPool: &unitinputs.CephBlockPoolReplicatedMirroring,
		},
		{
			name: "generate ceph block pool with mirroring image mode",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPoolMirroringImage := unitinputs.CephDeployPoolMirroring.DeepCopy()
				cephDplPoolMirroringImage.Mirroring.Mode = "image"
				return *cephDplPoolMirroringImage
			}(),
			expectedPool: func() *cephv1.CephBlockPool {
				expectedCephBlockPoolMirroringImage := unitinputs.CephBlockPoolReplicatedMirroring.DeepCopy()
				expectedCephBlockPoolMirroringImage.Spec.Mirroring.Mode = "image"
				return expectedCephBlockPoolMirroringImage
			}(),
		},
		{
			name: "generate ceph block pool with mirroring no mode specified",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPoolMirroringImage := unitinputs.CephDeployPoolMirroring.DeepCopy()
				cephDplPoolMirroringImage.Mirroring.Mode = ""
				return *cephDplPoolMirroringImage
			}(),
			expectedPool: &unitinputs.CephBlockPoolReplicated,
		},
		{
			name: "generate ceph block pool with mirroring incorrect mode",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPoolMirroringImage := unitinputs.CephDeployPoolMirroring.DeepCopy()
				cephDplPoolMirroringImage.Mirroring.Mode = "fake"
				return *cephDplPoolMirroringImage
			}(),
			expectedPool: &unitinputs.CephBlockPoolReplicated,
		},
		{
			name: "generate pool with parameters",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPool := unitinputs.CephDeployPoolReplicated.DeepCopy()
				cephDplPool.Parameters = map[string]string{
					"pg_num":            "512",
					"target_size_ratio": "0",
					"pg_autoscale_mode": "off",
				}
				return *cephDplPool
			}(),
			expectedPool: func() *cephv1.CephBlockPool {
				cephpool := unitinputs.CephBlockPoolReplicated.DeepCopy()
				cephpool.Spec.Parameters = map[string]string{
					"pg_num":            "512",
					"target_size_ratio": "0",
					"pg_autoscale_mode": "off",
				}
				return cephpool
			}(),
		},
		{
			name: "generate pool with preserve on delete flag",
			cephDpl: func() cephlcmv1alpha1.CephPool {
				cephDplPool := unitinputs.CephDeployPoolReplicated.DeepCopy()
				cephDplPool.PreserveOnDelete = true
				return *cephDplPool
			}(),
			expectedPool: func() *cephv1.CephBlockPool {
				cephpool := unitinputs.CephBlockPoolReplicated.DeepCopy()
				cephpool.Annotations = map[string]string{"cephdeployment.lcm.mirantis.com/preserve-on-delete": "true"}
				return cephpool
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualPool := generatePool(test.cephDpl, "rook-ceph")
			assert.Equal(t, test.expectedPool, actualPool)
		})
	}
}

func TestEnsurePools(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		stateChanged      bool
		expectedError     string
	}{
		{
			name:           "ensure pools - failed to list",
			cephDpl:        &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get list pools: failed to list cephblockpools",
		},
		{
			name:    "ensure pools - no pools present and no in spec",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListEmpty.DeepCopy(),
			},
		},
		{
			name:    "ensure pools - pool created",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{unitinputs.CephBlockPoolReplicated},
				},
			},
			stateChanged: true,
		},
		{
			name:    "ensure pools - pool create failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-cephblockpools": errors.New("create failed")},
			expectedError: "failed to ensure CephBlockPools: failed to create CephBlockPool rook-ceph/pool1-hdd: create failed",
		},
		{
			name:    "ensure pools - pool progressing",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListBaseNotReady.DeepCopy(),
			},
			expectedError: "failed to ensure CephBlockPools: found not ready CephBlockPool rook-ceph/pool1-hdd, waiting for readiness (current phase is Progressing)",
		},
		{
			name:    "ensure pools - pool updated",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{unitinputs.GetOpenstackPool("pool1-hdd", true, 0.5)},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListBaseReady.DeepCopy(),
			},
			stateChanged: true,
		},
		{
			name:    "ensure pools - pool update failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{unitinputs.GetOpenstackPool("pool1-hdd", true, 0.5)},
				},
			},
			apiErrors:     map[string]error{"update-cephblockpools": errors.New("update failed")},
			expectedError: "failed to ensure CephBlockPools: failed to update CephBlockPool rook-ceph/pool1-hdd: update failed",
		},
		{
			name:    "ensure pools - pool deleted",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListBaseReady.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListEmpty.DeepCopy(),
			},
			stateChanged: true,
		},
		{
			name:    "ensure pools - pool delete failed",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListBaseReady.DeepCopy(),
			},
			apiErrors:     map[string]error{"delete-cephblockpools": errors.New("delete failed")},
			expectedError: "failed to ensure CephBlockPools: failed to delete CephBlockPool rook-ceph/pool1-hdd: delete failed",
		},
		{
			name:    "ensure pools delete skipped",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{
						func() cephv1.CephBlockPool {
							cephpool := unitinputs.CephBlockPoolReplicated.DeepCopy()
							cephpool.Annotations = map[string]string{"cephdeployment.lcm.mirantis.com/preserve-on-delete": "true"}
							return *cephpool
						}(),
					},
				},
			},
			apiErrors: map[string]error{"delete-cephblockpools": errors.New("delete failed")},
		},
		{
			name:    "ensure pools - pools multiple errors",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.OpenstackCephBlockPoolsList.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-cephblockpools": errors.New("create failed")},
			expectedError: "failed to ensure CephBlockPools, multiple errors during pools ensure",
		},
		{
			name:    "ensure pools - nothing changed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": unitinputs.CephBlockPoolListBaseReady.DeepCopy(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephblockpools"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephblockpools"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephblockpools"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephblockpools"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			poolsChanged, err := c.ensurePools()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.stateChanged, poolsChanged)

			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestDeletePools(t *testing.T) {
	listBlockPools := unitinputs.CephBlockPoolListReady.DeepCopy()
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		deleted        bool
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name:           "failed to list ceph block pools",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list ceph block pools: failed to list cephblockpools",
		},
		{
			name:           "ceph block pools delete failed",
			inputResources: map[string]runtime.Object{"cephblockpools": listBlockPools},
			apiErrors:      map[string]error{"delete-cephblockpools": errors.New("failed to delete")},
			expectedError:  "some ceph block pools failed to delete",
		},
		{
			name:           "ceph block pools delete in progress",
			inputResources: map[string]runtime.Object{"cephblockpools": listBlockPools},
		},
		{
			name:           "no ceph block pools present",
			inputResources: map[string]runtime.Object{"cephblockpools": listBlockPools},
			deleted:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.BaseCephDeployment.DeepCopy()}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephblockpools"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephblockpools"}, test.inputResources, test.apiErrors)

			deleted, err := c.deletePools()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.deleted, deleted)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
