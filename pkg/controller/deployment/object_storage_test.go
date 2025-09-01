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
	"errors"
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureObjectStorage(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		rgwSSLCert: "some-time",
		cephConfigMap: map[string]string{
			"global": "some-time",
		},
	}
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		stateChanged   bool
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name:           "no object storage section, cleanup failed",
			cephDpl:        &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to cleanup object storage",
		},
		{
			name:    "no object storage section, cleanup in progress",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
			stateChanged: true,
		},
		{
			name:    "no object storage section, cleanup done",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
		},
		{
			name:    "object storage section present, no multisite, cleanup failed",
			cephDpl: unitinputs.CephDeployMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets":          &corev1.SecretList{},
				"storageclasses":   &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedError: "failed to cleanup object storage multisite: failed to get list CephObjectZones in 'rook-ceph' namespace: failed to list cephobjectzones",
		},
		{
			name:    "object storage section present, no multisite ok, rgw changed",
			cephDpl: unitinputs.CephDeployMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
			stateChanged: true,
		},
		{
			name:    "object storage section present, external, no multisite, rgw aligned",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy(), *unitinputs.RookCephRgwAdminSecret.DeepCopy()},
				},
				"storageclasses": &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							store := unitinputs.CephObjectStoreExternal.DeepCopy()
							store.Spec.Gateway.Annotations = map[string]string{
								"cephdeployment.lcm.mirantis.com/ssl-cert-generated": "current-time",
							}
							return *store
						}(),
					},
				},
			},
		},
		{
			name:    "object storage section present, external, no multisite, rgw apply failed",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"storageclasses": &storagev1.StorageClassList{},
			},
			expectedError: "failed to ensure ceph rgw: failed to list rgw object store: failed to list cephobjectstores",
		},
		{
			name:    "object storage section present, multisite apply failed, rgw changed",
			cephDpl: &unitinputs.CephDeployMultisiteRgw,
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"services":             &corev1.ServiceList{},
			},
			expectedError: "failed to ensure ceph object storage multisite: failed to ensure zone groups: failed to get list CephObjectZones in 'rook-ceph' namespace: failed to list cephobjectzones",
		},
		{
			name:    "object storage section present, multisite apply in progress, rgw changed",
			cephDpl: &unitinputs.CephDeployMultisiteRgw,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"services": &corev1.ServiceList{
					Items: []corev1.Service{*unitinputs.RgwExternalServiceGenerated.DeepCopy()},
				},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
			stateChanged: true,
		},
		{
			name:    "object storage section present, multisite and rgw aligned, no changes",
			cephDpl: &unitinputs.CephDeployMultisiteRgw,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						*unitinputs.RgwSSLCertSecret.DeepCopy(),
						*unitinputs.MultisiteRealmSecret.DeepCopy(),
					},
				},
				"services": &corev1.ServiceList{
					Items: []corev1.Service{*unitinputs.RgwExternalServiceGenerated.DeepCopy()},
				},
				"storageclasses": &storagev1.StorageClassList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{
						func() cephv1.CephObjectZone {
							correctZone := unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()
							correctZone.Spec.DataPool.CrushRoot = "default"
							correctZone.Spec.MetadataPool.CrushRoot = "default"
							return *correctZone
						}(),
					},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							store := unitinputs.CephObjectStoreWithZone.DeepCopy()
							store.Spec.Zone.Name = "secondary-zone1"
							// not default realm because it is not master zone
							store.Spec.DefaultRealm = false
							store.Spec.Gateway.Annotations = map[string]string{
								"cephdeployment.lcm.mirantis.com/config-global-updated":                 "some-time",
								"cephdeployment.lcm.mirantis.com/ssl-cert-generated":                    "current-time",
								"cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated": "",
							}
							return *store
						}(),
					},
				},
			},
		},
	}
	generateCrtFunc := lcmcommon.GenerateSelfSignedCert
	timeFunct := lcmcommon.GetCurrentTimeString
	rookRes := append([]string{"cephblockpools"}, multisiteResources...)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			lcmcommon.GetCurrentTimeString = func() string {
				return "current-time"
			}
			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
				return "fake-key", "fake-crt", "fake-ca", nil
			}
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease

			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets", "services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", rookRes, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", rookRes, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", rookRes, test.inputResources, test.apiErrors)

			stateChanged, err := c.ensureObjectStorage()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, stateChanged)

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.GetCurrentTimeString = timeFunct
	lcmcommon.GenerateSelfSignedCert = generateCrtFunc
	unsetTimestampsVar()
}

func TestDeleteObjectStorage(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		cleanupDone    bool
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name: "rgw object store is failed to remove",
			inputResources: map[string]runtime.Object{
				"storageclasses": &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"delete-cephobjectstores": errors.New("failed to delete CephObjectStore"),
			},
			expectedError: "failed to cleanup object storage",
		},
		{
			name: "rgw object store is removing",
			inputResources: map[string]runtime.Object{
				"storageclasses": &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
		},
		{
			name: "rgw certs/multisite are not removed",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"storageclasses":   &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			apiErrors: map[string]error{
				"delete-secrets": errors.New("failed to delete Secret"),
			},
			expectedError: "failed to cleanup object storage",
		},
		{
			name: "rgw certs and rgw root pool are removing",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{*unitinputs.BuiltinRgwRootPool.DeepCopy()},
				},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
		},
		{
			name: "rgw root pool is failed to remove",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{*unitinputs.BuiltinRgwRootPool.DeepCopy()},
				},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
			apiErrors: map[string]error{
				"delete-cephblockpools": errors.New("failed to delete cephblockpool"),
			},
			expectedError: "failed to cleanup object storage",
		},
		{
			name: "rgw multisite are removing",
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
		},
		{
			name: "object storage cleaned up",
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
			cleanupDone: true,
		},
		{
			name: "rgw external removing secrets is failed",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RookCephRgwAdminSecret.DeepCopy(), *unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"storageclasses":   &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			apiErrors: map[string]error{
				"delete-secrets": errors.New("failed to remove secret"),
			},
			cephDpl:       unitinputs.CephDeployExternalRgw.DeepCopy(),
			expectedError: "failed to cleanup object storage",
		},
		{
			name: "rgw external is removing secrets",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RookCephRgwAdminSecret.DeepCopy(), *unitinputs.RgwSSLCertSecret.DeepCopy()},
				},
				"storageclasses":   &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
		},
		{
			name: "external object storage cleaned up",
			inputResources: map[string]runtime.Object{
				"secrets":          &corev1.SecretList{},
				"storageclasses":   &storagev1.StorageClassList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			cephDpl:     unitinputs.CephDeployExternalRgw.DeepCopy(),
			cleanupDone: true,
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	rookRes := append([]string{"cephblockpools"}, multisiteResources...)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.cephDpl == nil {
				test.cephDpl = &unitinputs.BaseCephDeployment
			}
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "radosgw-admin realm list" {
					return "{}", "", nil
				}
				return "", "", errors.New("unexpected command call: " + e.Command)
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", rookRes, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", rookRes, test.inputResources, test.apiErrors)

			done, err := c.deleteObjectStorage()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.cleanupDone, done)

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}
