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
	"sort"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestProcessCephClients(t *testing.T) {
	cephlist := &cephv1.CephClientList{Items: []cephv1.CephClient{*unitinputs.CephClientTest.DeepCopy()}}
	cephlistUpdate := &cephv1.CephClientList{
		Items: []cephv1.CephClient{
			func() cephv1.CephClient {
				c := unitinputs.CephClientTest.DeepCopy()
				c.Spec.Caps = map[string]string{"test": "test"}
				return *c
			}(),
		},
	}
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		process           objectProcess
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:           "create client failed",
			inputResources: map[string]runtime.Object{"cephclients": unitinputs.CephClientListEmpty.DeepCopy()},
			process:        objectCreate,
			apiErrors:      map[string]error{"create-cephclients": errors.New("create failed")},
			expectedError:  "failed to create CephClient rook-ceph/test: create failed",
		},
		{
			name:              "create client completed",
			inputResources:    map[string]runtime.Object{"cephclients": unitinputs.CephClientListEmpty.DeepCopy()},
			process:           objectCreate,
			expectedResources: map[string]runtime.Object{"cephclients": cephlist},
		},
		{
			name:           "update client failed",
			inputResources: map[string]runtime.Object{"cephclients": cephlistUpdate.DeepCopy()},
			process:        objectUpdate,
			apiErrors:      map[string]error{"update-cephclients": errors.New("update failed")},
			expectedError:  "failed to update CephClient rook-ceph/test: update failed",
		},
		{
			name:              "update client completed",
			inputResources:    map[string]runtime.Object{"cephclients": cephlistUpdate.DeepCopy()},
			process:           objectUpdate,
			expectedResources: map[string]runtime.Object{"cephclients": cephlist},
		},
		{
			name:           "delete client failed",
			inputResources: map[string]runtime.Object{"cephclients": cephlist.DeepCopy()},
			process:        objectDelete,
			apiErrors:      map[string]error{"delete-cephclients": errors.New("delete failed")},
			expectedError:  "failed to delete CephClient rook-ceph/test: delete failed",
		},
		{
			name:              "delete client completed",
			inputResources:    map[string]runtime.Object{"cephclients": cephlist.DeepCopy()},
			process:           objectDelete,
			expectedResources: map[string]runtime.Object{"cephclients": &unitinputs.CephClientListEmpty},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephclients"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephclients"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephclients"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			err := c.processCephClients(test.process, unitinputs.TestCephClient)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestGenerateOpenStackClient(t *testing.T) {
	cephDplClients := cephlcmv1alpha1.CephDeployment{
		Spec: cephlcmv1alpha1.CephDeploymentSpec{
			Pools: []cephlcmv1alpha1.CephPool{
				unitinputs.GetCephDeployPool("vms", "vms"),
				unitinputs.GetCephDeployPool("images", "images"),
				unitinputs.GetCephDeployPool("volumes", "volumes"),
				unitinputs.GetCephDeployPool("backup", "backup"),
			},
		},
	}
	cephDplClientExtraVolumes := cephlcmv1alpha1.CephDeployment{
		Spec: cephlcmv1alpha1.CephDeploymentSpec{
			Pools: []cephlcmv1alpha1.CephPool{
				unitinputs.GetCephDeployPool("vms", "vms"),
				unitinputs.GetCephDeployPool("images", "images"),
				unitinputs.GetCephDeployPool("volumes", "volumes"),
				unitinputs.GetCephDeployPool("backup", "backup"),
				unitinputs.GetCephDeployPool("volumes-backend-1", "volumes-backend"),
				unitinputs.GetCephDeployPool("volumes-2", "volumes"),
			},
		},
	}
	blockPoolsExtra := &cephv1.CephBlockPoolList{Items: []cephv1.CephBlockPool{
		unitinputs.GetOpenstackPool("vms-hdd", false, 0), unitinputs.GetOpenstackPool("backup-hdd", false, 0), unitinputs.GetOpenstackPool("images-hdd", false, 0),
		unitinputs.GetOpenstackPool("volumes-hdd", false, 0), unitinputs.GetOpenstackPool("volumes-backend-1-hdd", false, 0), unitinputs.GetOpenstackPool("volumes-2-hdd", false, 0),
	}}

	tests := []struct {
		name               string
		cephDpl            cephlcmv1alpha1.CephDeployment
		clientName         string
		poolsList          *cephv1.CephBlockPoolList
		expectedCephClient cephlcmv1alpha1.CephClient
		expectedError      string
	}{
		{
			name:               "cephclient cinder - success",
			clientName:         "cinder",
			cephDpl:            cephDplClients,
			poolsList:          &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClient: unitinputs.CephDeployClientCinder,
		},
		{
			name:       "cephclient cinder with volumes-backend pools - success",
			clientName: "cinder",
			cephDpl:    cephDplClientExtraVolumes,
			poolsList:  blockPoolsExtra,
			expectedCephClient: func() cephlcmv1alpha1.CephClient {
				cinderClient := unitinputs.CephDeployClientCinder.DeepCopy()
				cinderClient.Caps["osd"] = "profile rbd pool=volumes-hdd, profile rbd pool=volumes-backend-1-hdd, profile rbd pool=volumes-2-hdd, profile rbd-read-only pool=images-hdd, profile rbd pool=backup-hdd"
				return *cinderClient
			}(),
		},
		{
			name:       "cephclient cinder required pool missing in spec - failed",
			clientName: "cinder",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{Pools: []cephlcmv1alpha1.CephPool{unitinputs.GetCephDeployPool("images", "images")}},
			},
			poolsList:     &unitinputs.OpenstackCephBlockPoolsList,
			expectedError: "ceph block pool with role volumes not found in pools",
		},
		{
			name:          "cephclient cinder required pool not found on cluster - failed",
			clientName:    "cinder",
			cephDpl:       cephDplClients,
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to get one of the required cephblockpools for cinder client: cephblockpools \"volumes-hdd\" not found",
		},
		{
			name:               "cephclient glance - success",
			clientName:         "glance",
			cephDpl:            cephDplClients,
			poolsList:          &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClient: unitinputs.CephDeployClientGlance,
		},
		{
			name:       "cephclient glance required pool missing in spec - failed",
			clientName: "glance",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{Pools: []cephlcmv1alpha1.CephPool{unitinputs.GetCephDeployPool("volumes", "volumes")}},
			},
			poolsList:     &unitinputs.OpenstackCephBlockPoolsList,
			expectedError: "ceph block pool with role images not found in pools",
		},
		{
			name:          "cephclient glance required pool not found on cluster - failed",
			clientName:    "glance",
			cephDpl:       cephDplClients,
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to get one of the required cephblockpools for glance client: cephblockpools \"images-hdd\" not found",
		},
		{
			name:               "cephclient nova - success",
			clientName:         "nova",
			cephDpl:            cephDplClients,
			poolsList:          &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClient: unitinputs.CephDeployClientNova,
		},
		{
			name:       "cephclient nova with volumes-backend pools - success",
			clientName: "nova",
			cephDpl:    cephDplClientExtraVolumes,
			poolsList:  blockPoolsExtra,
			expectedCephClient: func() cephlcmv1alpha1.CephClient {
				novaClient := unitinputs.CephDeployClientNova.DeepCopy()
				novaClient.Caps["osd"] = "profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd, profile rbd pool=volumes-backend-1-hdd, profile rbd pool=volumes-2-hdd"
				return *novaClient
			}(),
		},
		{
			name:       "cephclient nova required pool missing in spec - failed",
			clientName: "nova",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{Pools: []cephlcmv1alpha1.CephPool{unitinputs.GetCephDeployPool("volumes", "volumes")}},
			},
			poolsList:     &unitinputs.OpenstackCephBlockPoolsList,
			expectedError: "ceph block pool with role vms not found in pools",
		},
		{
			name:          "cephclient nova required pool not found on cluster - failed",
			clientName:    "nova",
			cephDpl:       cephDplClients,
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to get one of the required cephblockpools for nova client: cephblockpools \"vms-hdd\" not found",
		},
		{
			name:          "unknown openstack client type - failed",
			clientName:    "unknown",
			cephDpl:       cephDplClients,
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to find pool type for 'unknown' client",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: &test.cephDpl}, nil)
			inputResources := map[string]runtime.Object{"cephblockpools": test.poolsList}
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephblockpools"}, inputResources, nil)

			actualCephClient, err := c.generateOpenStackClient(test.clientName)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, actualCephClient, test.expectedCephClient)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestCalculateOpenstackClients(t *testing.T) {
	basePoolsForSpec := []cephlcmv1alpha1.CephPool{
		unitinputs.GetCephDeployPool("images", "images"),
		unitinputs.GetCephDeployPool("volumes", "volumes"),
		unitinputs.GetCephDeployPool("backup", "backup"),
		unitinputs.GetCephDeployPool("vms", "vms"),
	}
	tests := []struct {
		name                string
		cephDpl             cephlcmv1alpha1.CephDeployment
		poolsList           *cephv1.CephBlockPoolList
		expectedCephClients []cephlcmv1alpha1.CephClient
		expectedError       string
	}{
		{
			name: "no openstack clients in spec - success",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{Pools: basePoolsForSpec},
			},
			poolsList: &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClients: []cephlcmv1alpha1.CephClient{
				unitinputs.CephDeployClientCinder, unitinputs.CephDeployClientGlance, unitinputs.CephDeployClientNova,
			},
		},
		{
			name: "openstack cinder client in spec - success",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{unitinputs.CephDeployClientCinder},
					Pools:   basePoolsForSpec,
				},
			},
			poolsList: &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClients: []cephlcmv1alpha1.CephClient{
				unitinputs.CephDeployClientGlance, unitinputs.CephDeployClientNova,
			},
		},
		{
			name: "openstack cinder and nova client in spec - success",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientCinder, unitinputs.CephDeployClientNova,
					},
					Pools: basePoolsForSpec,
				},
			},
			poolsList: &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClients: []cephlcmv1alpha1.CephClient{
				unitinputs.CephDeployClientGlance,
			},
		},
		{
			name: "all openstack clients in spec - success",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientCinder, unitinputs.CephDeployClientGlance, unitinputs.CephDeployClientNova,
					},
					Pools: basePoolsForSpec,
				},
			},
			poolsList:           &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClients: []cephlcmv1alpha1.CephClient{},
		},
		{
			name: "manila not in spec, cephfs deployed - success",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Pools: basePoolsForSpec,
					SharedFilesystem: &cephlcmv1alpha1.CephSharedFilesystem{
						CephFS: []cephlcmv1alpha1.CephFS{{Name: "cephfs"}},
					},
				},
			},
			poolsList: &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClients: []cephlcmv1alpha1.CephClient{
				unitinputs.CephDeployClientCinder, unitinputs.CephDeployClientGlance, unitinputs.CephDeployClientNova, unitinputs.CephDeployClientManila,
			},
		},
		{
			name: "manila in spec, cephfs deployed - success",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientCinder, unitinputs.CephDeployClientGlance, unitinputs.CephDeployClientNova, unitinputs.CephDeployClientManila,
					},
					Pools: basePoolsForSpec,
				},
			},
			poolsList:           &unitinputs.OpenstackCephBlockPoolsList,
			expectedCephClients: []cephlcmv1alpha1.CephClient{},
		},
		{
			name: "no openstack clients in spec and backup pool missing in spec - fail",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Pools: []cephlcmv1alpha1.CephPool{
						unitinputs.GetCephDeployPool("images", "images"),
						unitinputs.GetCephDeployPool("volumes", "volumes"),
						unitinputs.GetCephDeployPool("vms", "vms"),
					},
				},
			},
			poolsList:     &unitinputs.OpenstackCephBlockPoolsList,
			expectedError: "failed to generate spec for Ceph openstack client cinder: ceph block pool with role backup not found in pools",
		},
		{
			name: "no openstack clients in spec and vms pool missing in spec - fail",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Pools: []cephlcmv1alpha1.CephPool{
						unitinputs.GetCephDeployPool("images", "images"),
						unitinputs.GetCephDeployPool("volumes", "volumes"),
						unitinputs.GetCephDeployPool("backup", "backup"),
					},
				},
			},
			poolsList:     &unitinputs.OpenstackCephBlockPoolsList,
			expectedError: "failed to generate spec for Ceph openstack client nova: ceph block pool with role vms not found in pools",
		},
		{
			name: "no cinder client in spec and volumes pool missing in spec - fail",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientGlance,
						unitinputs.CephDeployClientNova,
					},
					Pools: []cephlcmv1alpha1.CephPool{
						unitinputs.GetCephDeployPool("images", "images"),
						unitinputs.GetCephDeployPool("backup", "backup"),
						unitinputs.GetCephDeployPool("vms", "vms"),
					},
				},
			},
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to generate spec for Ceph openstack client cinder: ceph block pool with role volumes not found in pools",
		},
		{
			name: "no glance client in spec and images pool missing in spec - failed",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientCinder,
						unitinputs.CephDeployClientNova,
					},
					Pools: []cephlcmv1alpha1.CephPool{
						unitinputs.GetCephDeployPool("volumes", "volumes"),
						unitinputs.GetCephDeployPool("backup", "backup"),
						unitinputs.GetCephDeployPool("vms", "vms"),
					},
				},
			},
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to generate spec for Ceph openstack client glance: ceph block pool with role images not found in pools",
		},
		{
			name: "no glance client in spec and images pool raises error - failed",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientCinder,
						unitinputs.CephDeployClientNova,
					},
					Pools: basePoolsForSpec,
				},
			},
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to generate spec for Ceph openstack client glance: failed to get one of the required cephblockpools for glance client: cephblockpools \"images-hdd\" not found",
		},
		{
			name: "no cinder client in spec and volumes pool raises error - failed",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientGlance,
						unitinputs.CephDeployClientNova,
					},
					Pools: basePoolsForSpec,
				},
			},
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to generate spec for Ceph openstack client cinder: failed to get one of the required cephblockpools for cinder client: cephblockpools \"volumes-hdd\" not found",
		},
		{
			name: "no nova client in spec and vms pool raises error - failed",
			cephDpl: cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Clients: []cephlcmv1alpha1.CephClient{
						unitinputs.CephDeployClientGlance,
						unitinputs.CephDeployClientCinder,
					},
					Pools: basePoolsForSpec,
				},
			},
			poolsList:     &unitinputs.CephBlockPoolListEmpty,
			expectedError: "failed to generate spec for Ceph openstack client nova: failed to get one of the required cephblockpools for nova client: cephblockpools \"vms-hdd\" not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: &test.cephDpl}, nil)
			inputResources := map[string]runtime.Object{"cephblockpools": test.poolsList}
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephblockpools"}, inputResources, nil)

			actualCephClients, err := c.calculateOpenStackClients()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Nil(t, actualCephClients)
			} else {
				assert.Nil(t, err)
				sort.SliceStable(test.expectedCephClients, func(i, j int) bool {
					return test.expectedCephClients[i].Name < test.expectedCephClients[j].Name
				})
				sort.SliceStable(actualCephClients, func(i, j int) bool {
					return actualCephClients[i].Name < actualCephClients[j].Name
				})
				assert.Equal(t, test.expectedCephClients, actualCephClients)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestDeleteCephClients(t *testing.T) {
	tests := []struct {
		name           string
		deleted        bool
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name: "cephclients empty list: skip delete - success",
			inputResources: map[string]runtime.Object{
				"cephclients": &unitinputs.CephClientListEmpty,
			},
			deleted: true,
		},
		{
			name:           "cephclients list failed, skip delete - failed",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list ceph clients: failed to list cephclients",
		},
		{
			name: "cephclients list ok, delete partially failed - in progress",
			inputResources: map[string]runtime.Object{
				"cephclients": unitinputs.CephClientListReady.DeepCopy(),
			},
			apiErrors:     map[string]error{"delete-cephclients-client1": errors.New("failed to delete")},
			expectedError: "some CephClients failed to delete",
		},
		{
			name: "cephclients list ok, delete ok - in progress",
			inputResources: map[string]runtime.Object{
				"cephclients": unitinputs.CephClientListReady.DeepCopy(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.BaseCephDeployment.DeepCopy()}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephclients"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephclients"}, test.inputResources, test.apiErrors)

			deleted, err := c.deleteCephClients()
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

func TestEnsureCephClients(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedChange    bool
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:           "ensure ceph clients - list cephclients failed",
			cephDpl:        unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list CephClients in rook-ceph namespace: failed to list cephclients",
		},
		{
			name:    "ensure ceph clients - no clients, nothing to do",
			cephDpl: unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclients": unitinputs.CephClientListEmpty.DeepCopy(),
			},
		},
		{
			name:    "ensure ceph clients - client created",
			cephDpl: unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephclients": unitinputs.CephClientListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{unitinputs.TestCephClient},
				},
			},
			expectedChange: true,
		},
		{
			name:    "ensure ceph clients - client create failed",
			cephDpl: unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephclients": unitinputs.CephClientListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-cephclients": errors.New("create failed")},
			expectedError: "failed to ensure CephClients: failed to create CephClient rook-ceph/test: create failed",
		},
		{
			name:    "ensure ceph clients - client updated",
			cephDpl: unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{
						func() cephv1.CephClient {
							c := unitinputs.TestCephClientReady.DeepCopy()
							c.Spec.Caps["osd"] = "yyy"
							return *c
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{unitinputs.TestCephClientReady},
				},
			},
			expectedChange: true,
		},
		{
			name:    "ensure ceph clients - client update failed",
			cephDpl: unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{
						func() cephv1.CephClient {
							c := unitinputs.TestCephClientReady.DeepCopy()
							c.Spec.Caps["osd"] = "yyy"
							return *c
						}(),
					},
				},
			},
			apiErrors:     map[string]error{"update-cephclients": errors.New("update failed")},
			expectedError: "failed to ensure CephClients: failed to update CephClient rook-ceph/test: update failed",
		},
		{
			name:    "ensure ceph clients - client removed",
			cephDpl: unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{*unitinputs.TestCephClientReady.DeepCopy()},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephclients": &unitinputs.CephClientListEmpty,
			},
			expectedChange: true,
		},
		{
			name:    "ensure ceph clients - client remove failed",
			cephDpl: unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{*unitinputs.TestCephClientReady.DeepCopy()},
				},
			},
			apiErrors:     map[string]error{"delete-cephclients": errors.New("delete failed")},
			expectedError: "failed to ensure CephClients: failed to delete CephClient rook-ceph/test: delete failed",
		},
		{
			name:    "ensure ceph clients - client not ready",
			cephDpl: unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{*unitinputs.TestCephClientNotReady.DeepCopy()},
				},
			},
			expectedError: "failed to ensure CephClients: found not ready CephClient rook-ceph/test, waiting for readiness (current phase is Progressing)",
		},
		{
			name:    "ensure ceph clients - failed to prepare openstack clients",
			cephDpl: unitinputs.CephDeployMoskWithCephFS,
			inputResources: map[string]runtime.Object{
				"cephclients": &unitinputs.CephClientListEmpty,
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{unitinputs.GetOpenstackPool("images-hdd", false, 0), unitinputs.GetOpenstackPool("volumes-hdd", false, 0), unitinputs.GetOpenstackPool("backup-hdd", false, 0)},
				},
			},
			expectedError: "failed to calculate OpenStack CephClients: failed to generate spec for Ceph openstack client nova: failed to get one of the required cephblockpools for nova client: cephblockpools \"vms-hdd\" not found",
		},
		{
			name:    "ensure ceph clients - openstack clients are not ready, multiple issues",
			cephDpl: unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephclients":    unitinputs.CephClientListOpenstack.DeepCopy(),
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
			},
			expectedError: "failed to ensure CephClients, multiple errors during CephClients ensure",
		},
		{
			name:    "ensure ceph clients - client ready, nothing todo",
			cephDpl: unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{*unitinputs.TestCephClientReady.DeepCopy()},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: &test.cephDpl}, nil)

			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephclients"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephblockpools"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephclients"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephclients"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephclients"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			clientsChanged, err := c.ensureCephClients()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedChange, clientsChanged)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
