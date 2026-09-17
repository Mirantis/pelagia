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
	"context"
	"reflect"
	"strings"
	"testing"

	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"
	vsapi "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/v3/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/v3/test/unit/inputs"
)

func TestEnsureVSCResources(t *testing.T) {
	forUpdateVSCList := &vsapi.VolumeSnapshotClassList{
		Items: []vsapi.VolumeSnapshotClass{
			func() vsapi.VolumeSnapshotClass {
				vsc := unitinputs.CsiCephFSPluginVSC.DeepCopy()
				vsc.Labels = nil
				return *vsc
			}(), unitinputs.CsiRBDPluginVSC},
	}
	tests := []struct {
		name            string
		cephDeployment  *cephlcmv1alpha1.CephDeployment
		lcmConfig       map[string]string
		vscList         *vsapi.VolumeSnapshotClassList
		driverList      *csiopapi.DriverList
		apiErrors       map[string]error
		expectedUpdate  bool
		expectedClasses *vsapi.VolumeSnapshotClassList
		expectedError   string
	}{
		{
			name:           "nothing to do - skip manage",
			cephDeployment: &unitinputs.BaseCephDeployment,
			lcmConfig: map[string]string{
				"DEPLOYMENT_MANAGE_VOLUMESNAPSHOTCLASSES": "false",
			},
			apiErrors: map[string]error{
				"get-drivers": errors.New("failed to get driver"),
			},
		},
		{
			name:           "failed to get volumesnapshotclass",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"get-volumesnapshotclasses": errors.New("failed to get volumesnapshotclass"),
			},
			expectedError: "failed to ensure VolumeSnapshotClasses",
		},
		{
			name:           "failed to get drivers",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"get-drivers": errors.New("failed to get driver"),
			},
			expectedError: "failed to ensure VolumeSnapshotClasses",
		},
		{
			name:           "failed to create volumesnapshotclass",
			cephDeployment: &unitinputs.BaseCephDeployment,
			driverList:     unitinputs.CsiDriversRook,
			apiErrors: map[string]error{
				"create-volumesnapshotclasses": errors.New("failed to create volumesnapshotclasses"),
			},
			expectedError: "failed to ensure VolumeSnapshotClasses",
		},
		{
			name:            "create volumesnapshotclass",
			cephDeployment:  &unitinputs.BaseCephDeployment,
			driverList:      unitinputs.CsiDriversRook,
			vscList:         unitinputs.VSCListEmpty.DeepCopy(),
			expectedClasses: unitinputs.VSCListPresent,
			expectedUpdate:  true,
		},
		{
			name:           "failed to remove volumesnapshotclass",
			cephDeployment: &unitinputs.BaseCephDeployment,
			vscList:        unitinputs.VSCListPresent.DeepCopy(),
			apiErrors: map[string]error{
				"delete-volumesnapshotclasses": errors.New("failed to delete volumesnapshotclasses"),
			},
			expectedClasses: unitinputs.VSCListPresent,
			expectedError:   "failed to ensure VolumeSnapshotClasses",
		},
		{
			name:            "remove volumesnapshotclass",
			cephDeployment:  &unitinputs.BaseCephDeployment,
			vscList:         unitinputs.VSCListPresent.DeepCopy(),
			expectedClasses: unitinputs.VSCListEmpty,
			expectedUpdate:  true,
		},
		{
			name:           "nothing to remove volumesnapshotclass",
			cephDeployment: &unitinputs.BaseCephDeployment,
			vscList:        unitinputs.VSCListEmpty.DeepCopy(),
		},
		{
			name:           "failed to update labels volumesnapshotclass",
			cephDeployment: &unitinputs.BaseCephDeployment,
			driverList:     unitinputs.CsiDriversRook,
			vscList:        forUpdateVSCList.DeepCopy(),
			apiErrors: map[string]error{
				"update-volumesnapshotclasses": errors.New("failed to update volumesnapshotclasses"),
			},
			expectedClasses: forUpdateVSCList,
			expectedError:   "failed to ensure VolumeSnapshotClasses",
		},
		{
			name:           "update labels for volumesnapshotclass",
			cephDeployment: &unitinputs.BaseCephDeployment,
			driverList:     unitinputs.CsiDriversRook,
			vscList:        forUpdateVSCList.DeepCopy(),
			expectedClasses: &vsapi.VolumeSnapshotClassList{
				Items: []vsapi.VolumeSnapshotClass{
					func() vsapi.VolumeSnapshotClass {
						vsc := unitinputs.CsiCephFSPluginVSC.DeepCopy()
						vsc.ResourceVersion = "2"
						return *vsc
					}(), unitinputs.CsiRBDPluginVSC},
			},
			expectedUpdate: true,
		},
		{
			name:            "nothing to do with volumesnapshotclass",
			cephDeployment:  &unitinputs.BaseCephDeployment,
			driverList:      unitinputs.CsiDriversRook,
			vscList:         unitinputs.VSCListPresent.DeepCopy(),
			expectedClasses: unitinputs.VSCListPresent,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDeployment}, test.lcmConfig)
			castErr := c.castExtensions()
			assert.Nil(t, castErr)

			builder := faketestclients.GetClientBuilder()
			if test.vscList != nil {
				builder = builder.WithLists(test.vscList)
			}
			if test.driverList != nil {
				builder = builder.WithLists(test.driverList)
			}
			if test.apiErrors != nil {
				interceptorFuncs := interceptor.Funcs{}
				if v, ok := test.apiErrors["get-drivers"]; ok {
					interceptorFuncs.Get = func(ctx context.Context, client crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "driver" {
							return v
						}
						return client.Get(ctx, key, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["get-volumesnapshotclasses"]; ok {
					interceptorFuncs.Get = func(ctx context.Context, client crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "volumesnapshotclass" {
							return v
						}
						return client.Get(ctx, key, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["create-volumesnapshotclasses"]; ok {
					interceptorFuncs.Create = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.CreateOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "volumesnapshotclass" {
							return v
						}
						return client.Create(ctx, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["update-volumesnapshotclasses"]; ok {
					interceptorFuncs.Update = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "volumesnapshotclass" {
							return v
						}
						return client.Update(ctx, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["delete-volumesnapshotclasses"]; ok {
					interceptorFuncs.Delete = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.DeleteOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "volumesnapshotclass" {
							return v
						}
						return client.Delete(ctx, obj, opts...)
					}
				}
				builder = builder.WithInterceptorFuncs(interceptorFuncs)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			updated, err := c.ensureVSCResources()
			if test.expectedError == "" {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedUpdate, updated)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, updated)
			}
			vscList := vsapi.VolumeSnapshotClassList{}
			err = c.api.ClientNoCache.List(c.context, &vscList)
			assert.Nil(t, err)
			if test.expectedClasses == nil {
				assert.Equal(t, 0, len(vscList.Items))
			} else {
				assert.Equal(t, test.expectedClasses, &vscList)
			}
		})
	}
}

func TestDeleteVolumeSnapshotClasses(t *testing.T) {
	tests := []struct {
		name       string
		vscList    *vsapi.VolumeSnapshotClassList
		notManaged bool
		removed    bool
	}{
		{
			name:       "volumesnapshotclasses are not managed, skipped",
			notManaged: true,
			vscList:    unitinputs.VSCListPresent.DeepCopy(),
			removed:    true,
		},
		{
			name:    "volumesnapshotclasses are removing",
			vscList: unitinputs.VSCListPresent.DeepCopy(),
		},
		{
			name:    "no volumesnapshotclasses to remove",
			vscList: unitinputs.VSCListEmpty,
			removed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			if test.notManaged {
				c.lcmConfig.DeployParams.ManageVolumeSnapshotClasses = false
			}
			builder := faketestclients.GetClientBuilder()
			if test.vscList != nil {
				builder = builder.WithLists(test.vscList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteVolumeSnapshotClasses()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}
