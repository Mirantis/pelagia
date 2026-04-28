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
	"fmt"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	v1storage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGenerateRgwStore(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		rgwSSLCert: map[string]string{
			"rgw-store": "some-time",
		},
		cephConfigMap: map[string]string{
			"global":                 "some-time",
			"client.rgw.rgw.store.a": "some-time",
		},
	}
	tests := []struct {
		name              string
		cephDplRGW        cephlcmv1alpha1.CephObjectStore
		useDedicatedNodes bool
		syncRgwDaemon     bool
		hyperconverge     *cephlcmv1alpha1.CephDeploymentHyperConverge
		expected          *cephv1.CephObjectStore
	}{
		{
			name:              "base rgw spec, mon nodes placement, ssl enabled",
			cephDplRGW:        unitinputs.CephRgwBaseSpec,
			useDedicatedNodes: false,
			expected:          unitinputs.CephObjectStoreBase,
		},
		{
			name: "override rgw spec, rgw nodes placement, extra tolerations, ssl certs set",
			cephDplRGW: func() cephlcmv1alpha1.CephObjectStore {
				rgw := unitinputs.CephRgwBaseSpec.DeepCopy()
				rgwCasted, _ := rgw.GetSpec()
				rgwCasted.Gateway.SSLCertificateRef = "some-cert"
				rgwCasted.Gateway.Placement.Tolerations = []v1.Toleration{{Key: "custom-toleration", Operator: "Exists"}}
				rgw.Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return *rgw
			}(),
			useDedicatedNodes: true,
			expected: func() *cephv1.CephObjectStore {
				rgw := unitinputs.CephObjectStoreBase.DeepCopy()
				rgw.Spec.Gateway.SSLCertificateRef = "some-cert"
				rgw.Spec.Gateway.CaBundleRef = "some-cert"
				rgw.Spec.Gateway.Placement.Tolerations = []v1.Toleration{{Key: "ceph_role_rgw", Operator: "Exists"}, {Key: "custom-toleration", Operator: "Exists"}}
				rgw.Spec.Gateway.Placement.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key = "ceph_role_rgw"
				return rgw
			}(),
		},
		{
			name:              "multisite rgw run sync with single daemon",
			cephDplRGW:        unitinputs.CephDeployMultisiteMasterRgw.Spec.ObjectStorage.Rgws[0],
			useDedicatedNodes: false,
			expected:          unitinputs.CephObjectStoreWithZone,
		},
		{
			name:              "multisite rgw run sync with separate daemon, main rgw",
			cephDplRGW:        unitinputs.MultisiteRgwWithSyncDaemon.Spec.ObjectStorage.Rgws[0],
			useDedicatedNodes: false,
			expected: func() *cephv1.CephObjectStore {
				rgw := unitinputs.CephObjectStoreWithZone.DeepCopy()
				rgw.Spec.Zone.Name = "secondary-zone1"
				rgw.Spec.Gateway.DisableMultisiteSyncTraffic = true
				return rgw
			}(),
		},
		{
			name:              "multisite rgw run sync with separate daemon, sync rgw",
			cephDplRGW:        unitinputs.MultisiteRgwWithSyncDaemon.Spec.ObjectStorage.Rgws[1],
			useDedicatedNodes: false,
			syncRgwDaemon:     true,
			expected:          unitinputs.CephObjectStoreWithSyncDaemon,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.syncRgwDaemon {
				resourceUpdateTimestamps.cephConfigMap["client.rgw.rgw.store.sync.a"] = "some-time-sync"
			}
			castedSpec, err := test.cephDplRGW.GetSpec()
			assert.Nil(t, err)
			actual := generateRgw(castedSpec, test.cephDplRGW.Name, "rook-ceph", test.useDedicatedNodes)
			assert.Equal(t, test.expected, actual)
		})
	}
	unsetTimestampsVar()
}

func TestGenerateRgwStorageClass(t *testing.T) {
	actual := generateRgwStorageClass("rgw-store", "rgw-store-bucket", "rook-ceph")
	expected := unitinputs.RgwStorageClass
	assert.Equal(t, expected, actual)
}

func TestGenerateRgwExternalService(t *testing.T) {
	labelSelector, err := metav1.ParseToLabelSelector("external_access=rgw")
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"external_access": "rgw"}, labelSelector.MatchLabels)
	rgwExternalSvc := generateRgwExternalService("rgw-store", "rook-ceph", labelSelector, int32(80), int32(8443))
	assert.Equal(t, unitinputs.RgwExternalServiceGenerated, rgwExternalSvc)
}

func TestGenerateRgwExternal(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		rgwSSLCert: map[string]string{
			"rgw-store": "some-time",
		},
		cephConfigMap: map[string]string{
			"global":                 "some-time",
			"client.rgw.rgw.store.a": "some-time",
		},
	}
	tests := []struct {
		name          string
		cephDplRGW    cephlcmv1alpha1.CephObjectStore
		expected      *cephv1.CephObjectStore
		expectedError string
	}{
		{
			name: "generate external rgw - no external endpoints, failed",
			cephDplRGW: cephlcmv1alpha1.CephObjectStore{
				Name: "rgw-store",
				Spec: runtime.RawExtension{
					Raw: []byte(`{}`),
				},
			},
			expectedError: "external RGW endpoints is not specified for external ceph cluster",
		},
		{
			name:       "generate external rgw - success",
			cephDplRGW: unitinputs.CephRgwExternal,
			expected:   unitinputs.CephObjectStoreExternal,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			casted, err := test.cephDplRGW.GetSpec()
			assert.Nil(t, err)
			actual, err := generateRgwExternal(casted, test.cephDplRGW.Name, "rook-ceph")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
	unsetTimestampsVar()
}

func TestEnsureRgwConsistence(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		consistent        bool
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:           "ensure rgw consistence - cant list cephobjectstore",
			cephDpl:        &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list rgw object store: failed to list cephobjectstores",
		},
		{
			name:    "ensure rgw consistence - multiple cephobjectstores, delete failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						{ObjectMeta: metav1.ObjectMeta{Namespace: "rook-ceph", Name: "unexpected-rgw"}},
						*unitinputs.CephObjectStoreBase,
					},
				},
				"secrets":        &v1.SecretList{},
				"services":       &v1.ServiceList{},
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
			apiErrors:     map[string]error{"delete-cephobjectstores": errors.New("CephObjectStore delete failed")},
			expectedError: "failed to cleanup inconsistent rgw resources: failed to cleanup rgw object store resources",
		},
		{
			name:    "ensure rgw consistence - multiple cephobjectstores, delete in progress",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						{ObjectMeta: metav1.ObjectMeta{Namespace: "rook-ceph", Name: "unexpected-rgw"}},
						*unitinputs.CephObjectStoreBase,
					},
				},
				"secrets":        &v1.SecretList{Items: []v1.Secret{unitinputs.RgwSSLCertSecret}},
				"services":       &v1.ServiceList{},
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase},
				},
			},
		},
		{
			name: "ensure rgw consistence - multisite with sync daemon, no cleanup",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.MultisiteRgwWithSyncDaemon.DeepCopy()
				dpl.Spec.ObjectStorage.Rgws = dpl.Spec.ObjectStorage.Rgws[1:]
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreMultisiteSyncList.DeepCopy(),
				"secrets":          &v1.SecretList{},
				"services":         &v1.ServiceList{},
				"storageclasses":   &unitinputs.StorageClassesListEmpty,
			},
		},
		{
			name:    "ensure rgw consistence - multisite no sync daemon, cleanup",
			cephDpl: &unitinputs.CephDeployMultisiteRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon,
						*unitinputs.CephObjectStoreWithZone,
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.RgwSSLCertSecret,
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "custom-secret",
								Namespace: "rook-ceph",
								Labels:    map[string]string{"cephdeployment.lcm.mirantis.com/self-signed-ssl-cert-for": "unexpected-rgw"},
							},
						},
					},
				},
				"services":       &v1.ServiceList{},
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone},
				},
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.RgwSSLCertSecret}},
			},
		},
		{
			name:    "ensure rgw consistence - multisite no sync daemon, no changes",
			cephDpl: &unitinputs.CephDeployMultisiteRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RgwSSLCertSecret},
				},
				"services":       &v1.ServiceList{},
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
			consistent: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets", "services"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectstores"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			consistent, err := c.ensureRgwConsistence()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.consistent, consistent)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestDeleteRgw(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		rgwRuntimeParams: "some-time",
		cephConfigMap: map[string]string{
			"client.rgw.rgw.store.a": "some-time",
		},
	}
	tests := []struct {
		name              string
		objectStoreName   string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		deleted           bool
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:           "delete rgw - failed to list users",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get list of rgw users: failed to list cephobjectstoreusers",
		},
		{
			name: "delete rgw - failed to list buckets",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
			},
			expectedError: "failed to get list of rgw buckets: failed to list objectbucketclaims",
		},
		{
			name: "delete rgw - failed to remove users/buckets",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": unitinputs.CephRgwUsersList.DeepCopy(),
				"objectbucketclaims":   unitinputs.CephRgwBucketsList.DeepCopy(),
			},
			apiErrors: map[string]error{
				"delete-cephobjectstoreusers": errors.New("user delete failed"),
				"delete-objectbucketclaims":   errors.New("bucket delete failed"),
			},
			expectedError: "failed to remove some rgw user/buckets",
		},
		{
			name: "delete rgw - users/buckets removing",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": unitinputs.CephRgwUsersList.DeepCopy(),
				"objectbucketclaims":   unitinputs.CephRgwBucketsList.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
			},
		},
		{
			name: "delete rgw - failed to list cephobjectstore",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
			},
			expectedError: "failed to list rgw object stores",
		},
		{
			name: "delete rgw - delete rgw service failed",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     unitinputs.CephObjectStoreListReady.DeepCopy(),
				"services":             &unitinputs.ServicesListEmpty,
			},
			apiErrors: map[string]error{
				"delete-services": errors.New("service delete failed"),
			},
			expectedError: "failed to cleanup rgw object store resources",
		},
		{
			name: "delete rgw - delete rgw storageclass failed",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     unitinputs.CephObjectStoreListReady.DeepCopy(),
				"services":             &unitinputs.ServicesListEmpty,
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{*unitinputs.RgwStorageClass.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"delete-storageclasses": errors.New("storageclass delete failed"),
			},
			expectedError: "failed to cleanup rgw object store resources",
		},
		{
			name: "delete rgw - rgw resources delete in progress",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBaseReady.DeepCopy()},
				},
				"services": &v1.ServiceList{
					Items: []v1.Service{*unitinputs.RgwExternalService.DeepCopy()},
				},
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{*unitinputs.RgwStorageClass.DeepCopy()},
				},
			},
			expectedResources: map[string]runtime.Object{
				"services":       &unitinputs.ServicesListEmpty,
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{}},
			},
		},
		{
			name: "delete rgw - delete cephobjectstore failed",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     unitinputs.CephObjectStoreListReady.DeepCopy(),
				"services":             &unitinputs.ServicesListEmpty,
				"storageclasses":       &unitinputs.StorageClassesListEmpty,
			},
			apiErrors: map[string]error{
				"delete-cephobjectstores": errors.New("cephObjectStore delete failed"),
			},
			expectedError: "failed to cleanup rgw object store resources",
		},
		{
			name: "delete rgw - delete cephobjectstore in progress",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     unitinputs.CephObjectStoreListReady.DeepCopy(),
				"services":             &unitinputs.ServicesListEmpty,
				"storageclasses":       &unitinputs.StorageClassesListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
		},
		{
			name: "delete rgw - nothing to delete",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     &unitinputs.CephObjectStoreListEmpty,
				"services":             &unitinputs.ServicesListEmpty,
				"storageclasses":       &unitinputs.StorageClassesListEmpty,
			},
			deleted: true,
		},
		{
			name: "delete rgw - multiple rgw cleanup",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy(),
						*unitinputs.CephObjectStoreWithZone.DeepCopy(),
					},
				},
				"services":       &unitinputs.ServicesListEmpty,
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
		},
		{
			name: "delete rgw - cleanup only specified",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy(),
						*unitinputs.CephObjectStoreWithZone.DeepCopy(),
					},
				},
				"services":       &unitinputs.ServicesListEmpty,
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
			objectStoreName: "rgw-store-sync",
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithZone.DeepCopy(),
					},
				},
			},
		},
	}
	cephAPIResources := []string{"cephobjectstoreusers", "cephobjectstores"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.BaseCephDeployment.DeepCopy()}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			faketestclients.FakeReaction(c.api.Rookclientset, "list", cephAPIResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", cephAPIResources, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Claimclientset, "list", []string{"objectbucketclaims"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Claimclientset, "delete", []string{"objectbucketclaims"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			deleted, err := c.deleteRgw(test.objectStoreName)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.deleted, deleted)
			if test.deleted {
				assert.Equal(t, updateTimestamps{cephConfigMap: map[string]string{}}, resourceUpdateTimestamps)
			}

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Claimclientset)
		})
	}
	unsetTimestampsVar()
}

func TestStatusRgw(t *testing.T) {
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		expectedError  string
	}{
		{
			name:           "ensure status rgw - failed to get rgw",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get object store: failed to get resource(s) kind of 'cephobjectstores': list object is not specified in test",
		},
		{
			name: "ensure status rgw - no rgw created yet",
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
		},
		{
			name: "ensure status rgw - rgw not ready to process",
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							rgw := unitinputs.CephObjectStoreBaseReady.DeepCopy()
							rgw.Status.Phase = "Progressing"
							return *rgw
						}(),
					},
				},
			},
			expectedError: "rgw is not ready to be updated, current phase is Progressing",
		},
		{
			name: "ensure status rgw - rgw ready to reconcile",
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &unitinputs.CephObjectStoreBaseListReady,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.CephDeployNonMosk.DeepCopy()}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephobjectstores"}, test.inputResources, nil)

			err := c.statusRgw("rgw-store")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestEnsureRgwStorageClass(t *testing.T) {
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		changed           bool
		expectedError     string
	}{
		{
			name: "ensure rgw storageclass - check failed",
			inputResources: map[string]runtime.Object{
				"storageclasses": unitinputs.StorageClassesListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-storageclasses": errors.New("storageclass get failed")},
			expectedError: "storageclass get failed",
		},
		{
			name: "ensure rgw storageclass - create failed",
			inputResources: map[string]runtime.Object{
				"storageclasses": unitinputs.StorageClassesListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-storageclasses": errors.New("storageclass create failed")},
			expectedError: "failed to create rgw storage class: storageclass create failed",
		},
		{
			name: "ensure rgw storageclass - create success",
			inputResources: map[string]runtime.Object{
				"storageclasses": unitinputs.StorageClassesListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{unitinputs.RgwStorageClass},
				},
			},
			changed: true,
		},
		{
			name: "ensure rgw storageclass - create skipped",
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{unitinputs.RgwStorageClass},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.CephDeployObjectStorageCeph.DeepCopy()}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "get", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "create", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureRgwStorageClass("rgw-store")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.changed, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
		})
	}
	// unset global var to avoid intersection
	unsetTimestampsVar()
}

func TestEnsureRgwObject(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		rgwIdx            int
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		newTimestamps     *updateTimestamps
		changed           bool
		expectedError     string
	}{
		{
			name:    "ensure rgw - create rgw failed",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-cephobjectstores": errors.New("failed to create cephobjectstore")},
			expectedError: "failed to create rgw: failed to create cephobjectstore",
		},
		{
			name:    "ensure rgw - create rgw base",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			newTimestamps: &updateTimestamps{
				rgwSSLCert: map[string]string{
					"rgw-store": "some-time",
				},
				cephConfigMap: map[string]string{
					"global":                 "some-time",
					"client.rgw.rgw.store.a": "some-time",
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase},
				},
			},
			changed: true,
		},
		{
			name: "ensure rgw - update rgw base",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMosk.DeepCopy()
				cd.Spec.Nodes = unitinputs.CephDeployEnsureRolesCrush.Spec.Nodes
				return cd
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
			newTimestamps: &updateTimestamps{
				rgwSSLCert: map[string]string{
					"rgw-store": "new-ssl-time",
				},
				cephConfigMap: map[string]string{
					"global":                 "new-global-time",
					"client.rgw.rgw.store.a": "new-rgw-time",
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							rgw := unitinputs.CephObjectStoreBase.DeepCopy()
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "new-ssl-time"
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "new-global-time"
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "new-rgw-time"
							rgw.Spec.Gateway.Placement.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      "ceph_role_rgw",
											Operator: "In",
											Values: []string{
												"true",
											},
										},
									},
								},
							}
							rgw.Spec.Gateway.Placement.Tolerations = []v1.Toleration{
								{
									Key:      "ceph_role_rgw",
									Operator: "Exists",
								},
							}
							return *rgw
						}(),
					},
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - update rgw failed",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal.DeepCopy()},
				},
			},
			apiErrors:     map[string]error{"update-cephobjectstores": errors.New("failed to update cephobjectstore")},
			expectedError: "failed to update rgw: failed to update cephobjectstore",
		},
		{
			name:    "ensure rgw - nothing to do rgw base",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase},
				},
			},
			newTimestamps: &updateTimestamps{
				rgwSSLCert: map[string]string{
					"rgw-store": "some-time",
				},
				cephConfigMap: map[string]string{
					"global":                 "some-time",
					"client.rgw.rgw.store.a": "some-time",
				},
			},
		},
		{
			name:    "ensure rgw - external rgw failed, no rgw ops user secret",
			cephDpl: &unitinputs.CephDeployExternalRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"secrets":          &unitinputs.SecretsListEmpty,
			},
			expectedError: "failed to get rgw admin user secret rook-ceph/rgw-admin-ops-user: secrets \"rgw-admin-ops-user\" not found",
		},
		{
			name: "ensure rgw - external rgw spec without external endpoints - create failed",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: unitinputs.CephDeployExternal.Spec.Cluster.DeepCopy(),
					ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
						Rgws: []cephlcmv1alpha1.CephObjectStore{
							{
								Name: "rgw-store",
								Spec: runtime.RawExtension{
									Raw: []byte(`{}`),
								},
							},
						},
					},
				},
			},
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RookCephRgwAdminSecret},
				},
			},
			expectedError: "failed to generate external rgw: external RGW endpoints is not specified for external ceph cluster",
		},
		{
			name:    "ensure rgw - external rgw created",
			cephDpl: &unitinputs.CephDeployExternalRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RookCephRgwAdminSecret},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal},
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - external rgw updated",
			cephDpl: &unitinputs.CephDeployExternalRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							rgw := unitinputs.CephObjectStoreExternal.DeepCopy()
							rgw.Labels = nil
							rgw.Spec.Gateway.Port = 3333
							return *rgw
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RookCephRgwAdminSecret},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal},
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - external rgw nothing to do",
			cephDpl: &unitinputs.CephDeployExternalRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RookCephRgwAdminSecret},
				},
			},
		},
		{
			name: "ensure rgw - multiste rgw failed with unknown zone",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: unitinputs.BaseCephDeployment.Spec.Cluster.DeepCopy(),
					ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
						Rgws: unitinputs.CephDeployMultisiteMasterRgw.Spec.ObjectStorage.Rgws,
					},
				},
			},
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			expectedError: "failed to generate rgw with unknown zone1 zone",
		},
		{
			name: "ensure rgw - multiste rgw failed with invalid zone",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: unitinputs.BaseCephDeployment.Spec.Cluster.DeepCopy(),
					ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
						Rgws: unitinputs.CephDeployMultisiteMasterRgw.Spec.ObjectStorage.Rgws,
						Zones: []cephlcmv1alpha1.CephObjectZone{
							{
								Name: "zone2",
								Spec: runtime.RawExtension{Raw: []byte(`{}`)},
							},
						},
					},
				},
			},
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			expectedError: "failed to generate rgw with unknown zone1 zone",
		},
		{
			name:    "ensure rgw - multiste rgw created",
			cephDpl: &unitinputs.CephDeployMultisiteMasterRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone},
				},
			},
			newTimestamps: &updateTimestamps{
				rgwSSLCert: map[string]string{
					"rgw-store": "some-time",
				},
				cephConfigMap: map[string]string{
					"global":                 "some-time",
					"client.rgw.rgw.store.a": "some-time",
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - multiste rgw update failed, zone cant be changed",
			cephDpl: &unitinputs.CephDeployMultisiteRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone.DeepCopy()},
				},
			},
			expectedError: "failed to update rgw, zone change is not supported",
		},
		{
			name:    "ensure rgw - multiste rgw updated from single",
			cephDpl: &unitinputs.CephDeployMultisiteMasterRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone},
				},
			},
			newTimestamps: &updateTimestamps{
				rgwSSLCert: map[string]string{
					"rgw-store": "some-time",
				},
				cephConfigMap: map[string]string{
					"global":                 "some-time",
					"client.rgw.rgw.store.a": "some-time",
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - multiste rgw nothing to do",
			cephDpl: &unitinputs.CephDeployMultisiteMasterRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone.DeepCopy()},
				},
			},
			newTimestamps: &updateTimestamps{
				rgwSSLCert: map[string]string{
					"rgw-store": "some-time",
				},
				cephConfigMap: map[string]string{
					"global":                 "some-time",
					"client.rgw.rgw.store.a": "some-time",
				},
			},
		},
		{
			name:    "ensure rgw - multisite rgw with sync daemon created",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			rgwIdx:  1,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithSyncDaemon},
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - multisite rgw with sync daemon updated",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			rgwIdx:  1,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							store := unitinputs.CephObjectStoreWithZone.DeepCopy()
							store.Spec.Zone.Name = "secondary-zone1"
							return *store
						}(),
						func() cephv1.CephObjectStore {
							store := unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy()
							store.Spec.Gateway.Port = 3333
							return *store
						}(),
					},
				},
			},
			newTimestamps: &updateTimestamps{
				cephConfigMap: map[string]string{
					"global":                      "new-global-time",
					"client.rgw.rgw.store.a":      "new-rgw-time",
					"client.rgw.rgw.store.sync.a": "new-rgw-sync-time",
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							store := unitinputs.CephObjectStoreWithZone.DeepCopy()
							store.Spec.Zone.Name = "secondary-zone1"
							return *store
						}(),
						func() cephv1.CephObjectStore {
							rgw := unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy()
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.sync.a-updated"] = "new-rgw-sync-time"
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "new-global-time"
							return *rgw
						}(),
					},
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - multisite rgw with sync have no changes",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			rgwIdx:  1,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithSyncDaemon},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephobjectstores"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			if test.newTimestamps != nil {
				resourceUpdateTimestamps = *test.newTimestamps
			} else {
				resourceUpdateTimestamps = updateTimestamps{
					cephConfigMap: map[string]string{
						"global":                      "some-time",
						"client.rgw.rgw.store.a":      "some-time",
						"client.rgw.rgw.store.sync.a": "some-time-sync",
					},
				}
			}

			changed, err := c.ensureRgwObject(test.rgwIdx)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.changed, changed)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	// unset global var to avoid intersection
	unsetTimestampsVar()
}

func TestEnsureRgwResourcesUsers(t *testing.T) {
	fakeUser3 := func() *cephv1.CephObjectStoreUser {
		user := unitinputs.RgwUserBase.DeepCopy()
		user.Name = "fake-user-3"
		user.Spec.DisplayName = "fake-user-3"
		return user
	}()
	listUsersNotReady := &cephv1.CephObjectStoreUserList{
		Items: []cephv1.CephObjectStoreUser{
			unitinputs.CephRgwUsersList.Items[0],
			func() cephv1.CephObjectStoreUser {
				user := unitinputs.CephRgwUsersList.Items[1].DeepCopy()
				user.Status = nil
				return *user
			}(),
		},
	}
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		changed           bool
		expectedError     string
	}{
		{
			name:           "ensure rgw users - list users failed",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list rgw users: failed to list cephobjectstoreusers",
		},
		{
			name: "ensure rgw users - create failed, delete failed",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{unitinputs.CephRgwUsersList.Items[1], *fakeUser3},
				},
			},
			apiErrors: map[string]error{
				"create-cephobjectstoreusers": errors.New("cephobjectuser create failed"),
				"delete-cephobjectstoreusers": errors.New("cephobjectuser delete failed"),
			},
			expectedError: "failed to ensure CephObjectStoreUsers, multiple errors during users ensure",
		},
		{
			name: "ensure rgw users - create success, delete success",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{unitinputs.CephRgwUsersList.Items[0], *fakeUser3},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstoreusers": listUsersNotReady,
			},
			changed: true,
		},
		{
			name: "ensure rgw users - some users are not ready",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": listUsersNotReady,
			},
			expectedError: "failed to ensure CephObjectStoreUsers: found not ready CephObjectStoreUser rook-ceph/fake-user-2, waiting for readiness",
		},
		{
			name: "ensure rgw users - some users updated",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{
						unitinputs.CephRgwUsersList.Items[0],
						func() cephv1.CephObjectStoreUser {
							user := unitinputs.CephRgwUsersList.Items[1].DeepCopy()
							user.Labels = nil
							user.Spec.DisplayName = "fake-user"
							return *user
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephRgwUsersList,
			},
			changed: true,
		},
		{
			name: "ensure rgw users - some users update failed",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{
						unitinputs.CephRgwUsersList.Items[0],
						func() cephv1.CephObjectStoreUser {
							user := unitinputs.CephRgwUsersList.Items[1].DeepCopy()
							user.Spec.DisplayName = "fake-user"
							return *user
						}(),
					},
				},
			},
			apiErrors:     map[string]error{"update-cephobjectstoreusers": errors.New("cephobjectuser update failed")},
			expectedError: "failed to ensure CephObjectStoreUsers: failed to update CephObjectStoreUser rook-ceph/fake-user-2: cephobjectuser update failed",
		},
		{
			name: "ensure rgw users - nothing to update",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephRgwUsersList,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.CephDeployNonMosk.DeepCopy()}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectstoreusers"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephobjectstoreusers"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephobjectstoreusers"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectstoreusers"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureRgwUsers("rgw-store")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.changed, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestEnsureExternalService(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		lcmConfig         map[string]string
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		changed           bool
		expectedError     string
	}{
		{
			name:    "ensure external service - get failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"services": unitinputs.ServicesListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-services": errors.New("service get failed")},
			expectedError: "failed to get rgw external service: service get failed",
		},
		{
			name:    "ensure external service - create failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"services": unitinputs.ServicesListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-services": errors.New("service create failed")},
			expectedError: "failed to create rgw external service: service create failed",
		},
		{
			name:    "ensure external service - create success",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"services": unitinputs.ServicesListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
			changed: true,
		},
		{
			name:    "ensure external service - update ports failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{
					Items: []v1.Service{
						func() v1.Service {
							svc := unitinputs.RgwExternalServiceGenerated.DeepCopy()
							svc.Spec.Ports[0].Name = "custom"
							return *svc
						}(),
					},
				},
			},
			apiErrors:     map[string]error{"update-services": errors.New("service update failed")},
			expectedError: "failed to update rgw external service: service update failed",
		},
		{
			name:    "ensure external service - update ports success",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{
					Items: []v1.Service{
						func() v1.Service {
							svc := unitinputs.RgwExternalServiceGenerated.DeepCopy()
							svc.Spec.Ports = []v1.ServicePort{svc.Spec.Ports[0]}
							svc.Spec.Ports[0].Name = "custom"
							return *svc
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
			changed: true,
		},
		{
			name:      "ensure external service - update labels success",
			cephDpl:   &unitinputs.CephDeployNonMosk,
			lcmConfig: map[string]string{"RGW_PUBLIC_ACCESS_SERVICE_SELECTOR": "external_access=rgw"},
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{
					Items: []v1.Service{
						func() v1.Service {
							svc := unitinputs.RgwExternalServiceGenerated.DeepCopy()
							delete(svc.Labels, "external_access")
							delete(svc.Labels, "app.kubernetes.io/part-of")
							return *svc
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
			changed: true,
		},
		{
			name:    "ensure external service - nothing todo",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{*unitinputs.RgwExternalServiceGenerated.DeepCopy()}},
			},
		},
		{
			name:      "ensure external service - public is not required",
			cephDpl:   &unitinputs.CephDeployNonMosk,
			lcmConfig: map[string]string{"RGW_PUBLIC_ACCESS_SERVICE_SELECTOR": ""},
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{*unitinputs.RgwExternalServiceGenerated.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"services": &unitinputs.ServicesListEmpty,
			},
			changed: true,
		},
		{
			name: "ensure external service - failed to check ingress",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMosk.DeepCopy()
				cephDpl.Spec.IngressConfig = nil
				return cephDpl
			}(),
			inputResources: map[string]runtime.Object{
				"services": unitinputs.ServicesListEmpty.DeepCopy(),
				"secrets":  &unitinputs.SecretsListEmpty,
			},
			apiErrors:     map[string]error{"get-secrets": errors.New("failed to get secret")},
			expectedError: "failed to check ingress proxy presence: failed to get openstack-rgw-creds secret to ensure ingress: failed to get secret",
		},
		{
			name: "ensure external service - openstack pools specified, but no default openstack config",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMosk.DeepCopy()
				cephDpl.Spec.IngressConfig = nil
				return cephDpl
			}(),
			inputResources: map[string]runtime.Object{
				"services": unitinputs.ServicesListEmpty.DeepCopy(),
				"secrets":  &unitinputs.SecretsListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
			changed: true,
		},
		{
			name:    "ensure external service - ingress required, delete not found success",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"services": &unitinputs.ServicesListEmpty,
			},
		},
		{
			name:    "ensure external service - ingress required, delete failed",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{*unitinputs.RgwExternalServiceGenerated.DeepCopy()}},
			},
			apiErrors:     map[string]error{"delete-services": errors.New("service failed to delete")},
			expectedError: "failed to cleanup rgw external service rook-ceph-rgw-rgw-store-external: service failed to delete",
		},
		{
			name:    "ensure external service - ingress required, delete success",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{*unitinputs.RgwExternalServiceGenerated.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"services": &unitinputs.ServicesListEmpty,
			},
			changed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.lcmConfig)
			err := c.castExtensions()
			assert.Nil(t, err)

			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"services", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"services"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureExternalService(0)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.changed, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestEnsureRgwMain(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		rgwSSLCert: map[string]string{
			"rgw-store": "new-time",
		},
		cephConfigMap: map[string]string{
			"global":                      "some-time",
			"client.rgw.rgw.store.a":      "some-time",
			"client.rgw.rgw.store.sync.a": "some-time-sync",
		},
	}
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		changed           bool
		expectedError     string
	}{
		{
			name:           "ensure rgw - failed to check consistence",
			cephDpl:        &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list rgw object store: failed to list cephobjectstores",
		},
		{
			name:    "ensure rgw - failed to check rgw status",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase}},
			},
			apiErrors:     map[string]error{"get-cephobjectstores": errors.New("get cephobjectstore failed")},
			expectedError: "error(s) during rgw ensure: failed to check rgw 'rgw-store' state",
		},
		{
			name:    "ensure rgw - failed to ensure rgw secrets",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"secrets":          unitinputs.ServicesListEmpty.DeepCopy(),
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-secrets": errors.New("get secret failed")},
			expectedError: "error(s) during rgw ensure: failed to ensure rgw ssl cert for rgw 'rgw-store'",
		},
		{
			name:    "ensure rgw - failed to ensure rgw object",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"secrets":          &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-cephobjectstores": errors.New("create cephobjectstore failed")},
			expectedError: "error(s) during rgw ensure: failed to ensure rgw object store 'rgw-store'",
		},
		{
			name:    "ensure rgw - failed to ensure rgw storageclass, users, buckets and external service",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"secrets":          &v1.SecretList{},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{
					func() cephv1.CephObjectStore {
						store := unitinputs.CephObjectStoreBase.DeepCopy()
						store.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "test-4-time"
						return *store
					}(),
				},
				},
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "test-4-time"}
						return *secret
					}(),
				},
				},
			},
			expectedError: "error(s) during rgw ensure: failed to ensure rgw 'rgw-store' storage class, failed to ensure rgw 'rgw-store' external service, failed to ensure rgw 'rgw-store' users",
		},
		{
			name:    "ensure rgw - ensure rgw completed, all created",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"objectbucketclaims":   unitinputs.ObjectBucketClaimListEmpty.DeepCopy(),
				"secrets":              &v1.SecretList{Items: []v1.Secret{}},
				"storageclasses":       unitinputs.StorageClassesListEmpty.DeepCopy(),
				"services":             unitinputs.ServicesListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{
					func() cephv1.CephObjectStore {
						store := unitinputs.CephObjectStoreBase.DeepCopy()
						store.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "test-5-time"
						return *store
					}(),
				},
				},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{unitinputs.GetCephRgwUser("fake-user-1", "rook-ceph", "rgw-store"), unitinputs.GetCephRgwUser("fake-user-2", "rook-ceph", "rgw-store")},
				},
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":       &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "test-5-time"}
						return *secret
					}(),
				},
				},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - ensure rgw, nothing to do",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreBaseListReady.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephRgwUsersList.DeepCopy(),
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "some-time"}
						return *secret
					}(),
				},
				},
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":       &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
		},
		{
			name:    "ensure rgw - ensure rgw, external, all created",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.RookCephRgwAdminSecret.DeepCopy()},
				},
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"storageclasses":       &v1storage.StorageClassList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.RookCephRgwAdminSecret.DeepCopy()},
				},
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"storageclasses":       &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"cephobjectstores":     &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal}},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - ensure rgw, external, nothing to do",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy(), *unitinputs.RookCephRgwAdminSecret.DeepCopy()},
				},
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"storageclasses":       &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"cephobjectstores":     &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal}},
			},
		},
		{
			name:    "ensure rgw - ensure rgw, openstack, all created",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"secrets":              &v1.SecretList{},
				"storageclasses":       &v1storage.StorageClassList{},
				"services":             unitinputs.ServicesListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{
					func() cephv1.CephObjectStore {
						store := unitinputs.CephObjectStoreBase.DeepCopy()
						store.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "test-9-time"
						return *store
					}(),
				},
				},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{unitinputs.RgwCeilometerUser},
				},
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{"cephdeployment.lcm.mirantis.com/ssl-cert-generated": "test-9-time"}
						return *secret
					}(),
				}},
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":       &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - ensure rgw, openstack, nothing to do",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreBaseListReady.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListMetrics.DeepCopy(),
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.OpenstackRgwCredsSecret,
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{"cephdeployment.lcm.mirantis.com/ssl-cert-generated": "some-time"}
						secret.Data["cabundle"] = []byte(string(secret.Data["cabundle"]) + string(unitinputs.OpenstackRgwCredsSecret.Data["ca_cert"]) + "\n")
						return *secret
					}(),
				}},
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":       unitinputs.ServicesListEmpty.DeepCopy(),
			},
		},
		{
			name:    "ensure rgw - multisite rgw with sync daemon created",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"secrets":              &v1.SecretList{Items: []v1.Secret{unitinputs.MultisiteCabundleSecret}},
				"storageclasses":       &v1storage.StorageClassList{},
				"services":             unitinputs.ServicesListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							rgw := unitinputs.CephObjectStoreWithZone.DeepCopy()
							rgw.Spec.Zone.Name = "secondary-zone1"
							rgw.Spec.Gateway.DisableMultisiteSyncTraffic = true
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "test-11-time"
							return *rgw
						}(),
						*unitinputs.CephObjectStoreWithSyncDaemon,
					},
				},
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.MultisiteCabundleSecret,
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{"cephdeployment.lcm.mirantis.com/ssl-cert-generated": "test-11-time"}
						return *secret
					}(),
				}},
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":       &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - multisite rgw with sync daemon, nothing to do",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     &unitinputs.CephObjectStoreMultisiteSyncList,
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.MultisiteCabundleSecret,
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{"cephdeployment.lcm.mirantis.com/ssl-cert-generated": "some-time"}
						return *secret
					}(),
				}},
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":       &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
		},
	}
	oldCrtFunc := lcmcommon.GenerateSelfSignedCert
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl.DeepCopy()}, nil)
			err := c.castExtensions()
			assert.Nil(t, err)

			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
				return "fake-key", "fake-crt", "fake-ca", nil
			}
			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("test-%d-time", idx)
			}

			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectstores", "cephobjectstoreusers"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephobjectstores", "cephobjectstoreusers"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "list", []string{"deployments"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"services", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"services", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"services", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "get", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "create", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Claimclientset, "list", []string{"objectbucketclaims"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Claimclientset, "create", []string{"objectbucketclaims"}, test.inputResources, nil)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureRgw()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.changed, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Claimclientset)
		})
	}
	unsetTimestampsVar()
	lcmcommon.GenerateSelfSignedCert = oldCrtFunc
}

func TestEnsureRgwBackendSSLCert(t *testing.T) {
	tests := []struct {
		name                   string
		certName               string
		inputResources         map[string]runtime.Object
		apiErrors              map[string]error
		stateChanged           bool
		expectedGenerationTime string
		expectedResources      map[string]runtime.Object
		expectedError          string
	}{
		{
			name:     "failed to get rgw ssl secret when it should be present",
			certName: "rgw-store-ssl-cert",
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-secrets-rgw-store-ssl-cert": errors.New("failed to get rgw-store-ssl-cert secret")},
			expectedError: "failed to get secret rook-ceph/rgw-store-ssl-cert: failed to get rgw-store-ssl-cert secret",
		},
		{
			name:     "rgw ssl secret is specified in secret directly, verification failed",
			certName: "rgw-store-ssl-cert",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertExpiredSecret.DeepCopy()}},
			},
			expectedError: "ssl verification failed for rgw 'rgw-store' ssl certs provided in 'rgw-store-ssl-cert' secret, update manually: pkg cacert is expired",
		},
		{
			name:     "rgw ssl secret is specified in secret directly, but secret has incorrect data",
			certName: "rgw-store-ssl-cert",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Data = nil
						return *secret
					}(),
				}},
			},
			expectedError: "rgw 'rgw-store' ssl certs provided in 'rgw-store-ssl-cert' secret has no required 'cert' and 'cacert' fields",
		},
		{
			name:     "rgw ssl secret is specified in secret directly, nothing to do",
			certName: "rgw-store-ssl-cert",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
			},
		},
		{
			name: "rgw ssl secret self-signed created",
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-4-time"}
						delete(secret.Data, "cabundle")
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-4-time",
			stateChanged:           true,
		},
		{
			name: "rgw ssl secret self-signed create failed",
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			apiErrors:              map[string]error{"create-secrets": errors.New("failed to create secret")},
			expectedGenerationTime: "a-test-4-time",
			expectedError:          "failed to create rgw ssl cert secret: failed to create secret",
		},
		{
			name: "rgw ssl secret self-generated renewed",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertExpiredSecret.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-6-time"}
						secret.Data["cabundle"] = unitinputs.RgwSSLCertExpiredSecret.Data["cabundle"]
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-6-time",
			stateChanged:           true,
		},
		{
			name: "rgw ssl secret self-generated renew failed",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertExpiredSecret.DeepCopy()}},
			},
			apiErrors:              map[string]error{"update-secrets": errors.New("failed to update secret")},
			expectedGenerationTime: "a-test-6-time",
			expectedError:          "failed to update rgw ssl cert secret: failed to update secret",
		},
		{
			name: "rgw ssl secret self-generated nothing to do",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-6-time"}
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-6-time",
		},
		{
			name: "rgw ssl secret self-generated, nothing to do, but update timing in memory",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "secret-timing"}
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "secret-timing",
		},
	}

	oldGenerateCertFunc := lcmcommon.GenerateSelfSignedCert
	oldCurrentTimeFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)

			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
				if _, ok := test.apiErrors["ssl-cert"]; ok {
					return "", "", "", errors.New("failed to generate ssl certificate")
				}
				return "fake-key", "fake-crt", "fake-ca", nil
			}

			// test global var resourceUpdateTimestamps.RgwSSLCert is updated correctly
			// after each test run, it should be non-set if create/update is not happened
			// or existing cert has no annotation, should be set if create/update happened
			// or existing cert has annotation, and should not be changed in next test if no
			// create/update happened
			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("a-test-%d-time", idx)
			}

			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			rgwName := "rgw-store"
			changed, err := c.ensureRgwBackendSSLCert(rgwName, test.certName)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			assert.Equal(t, test.expectedGenerationTime, resourceUpdateTimestamps.rgwSSLCert[rgwName])
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.GenerateSelfSignedCert = oldGenerateCertFunc
	lcmcommon.GetCurrentTimeString = oldCurrentTimeFunc
	// unset global var to do not override with other tests
	unsetTimestampsVar()
}

func TestEnsureRgwCaBundleCert(t *testing.T) {
	tests := []struct {
		name                   string
		cephDpl                *cephlcmv1alpha1.CephDeployment
		rgwIdx                 int
		inputResources         map[string]runtime.Object
		apiErrors              map[string]error
		stateChanged           bool
		expectedGenerationTime string
		expectedResources      map[string]runtime.Object
		expectedError          string
	}{
		{
			name:    "rgw cabundle required, but failed to get",
			cephDpl: unitinputs.MultisiteRgwWithSyncDaemon.DeepCopy(),
			rgwIdx:  1,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{}},
			},
			expectedError: "failed to get secret 'rook-ceph/multisite-rgw-secret' with cabundle: secrets \"multisite-rgw-secret\" not found",
		},
		{
			name:    "rgw cabundle required, but has no required data",
			cephDpl: unitinputs.MultisiteRgwWithSyncDaemon.DeepCopy(),
			rgwIdx:  1,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.MultisiteCabundleSecret.DeepCopy()
						delete(secret.Data, "cabundle")
						return *secret
					}(),
				}},
			},
			expectedError: "rgw 'rgw-store-sync' secret 'rook-ceph/multisite-rgw-secret' used for as cabundle has no required field 'cabundle'",
		},
		{
			name:    "rgw cabundle required and ok",
			rgwIdx:  1,
			cephDpl: unitinputs.MultisiteRgwWithSyncDaemon.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.MultisiteCabundleSecret}},
			},
		},
		{
			name: "failed to get ingress ssl secret",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.CephDeployMosk.DeepCopy()
				dpl.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				dpl.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedError: "failed to get ingress secret 'rook-ceph/rgw-store-ingress-secret': secrets \"rgw-store-ingress-secret\" not found",
		},
		{
			name: "cabundle created from ingress ssl secret",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.CephDeployMosk.DeepCopy()
				dpl.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				dpl.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.IngressRuleSecret}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.IngressRuleSecret,
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							delete(secret.Labels, "cephdeployment.lcm.mirantis.com/self-signed-ssl-cert-for")
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-4-time"}
							secret.Data = map[string][]byte{
								"cabundle": append(unitinputs.IngressRuleSecret.Data["ca.crt"], []byte("\n")...),
							}
							return *secret
						}(),
					},
				},
			},
			expectedGenerationTime: "a-test-4-time",
			stateChanged:           true,
		},
		{
			name: "cabundle updating from base and ingress ssl secret",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.CephDeployMosk.DeepCopy()
				dpl.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				dpl.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.IngressRuleSecret, unitinputs.RgwSSLCertSecret}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.IngressRuleSecret,
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-5-time"}
							caBundle := string(unitinputs.RgwSSLCertSecret.Data["cacert"]) + "\n" + string(unitinputs.IngressRuleSecret.Data["ca.crt"]) + "\n"
							secret.Data["cabundle"] = []byte(caBundle)
							return *secret
						}(),
					},
				},
			},
			expectedGenerationTime: "a-test-5-time",
			stateChanged:           true,
		},
		{
			name: "cabundle from base and ingress ssl secret nothing to do",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.CephDeployMosk.DeepCopy()
				dpl.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				dpl.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.IngressRuleSecret,
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-5-time"}
							caBundle := string(unitinputs.RgwSSLCertSecret.Data["cacert"]) + "\n" + string(unitinputs.IngressRuleSecret.Data["ca.crt"]) + "\n"
							secret.Data["cabundle"] = []byte(caBundle)
							return *secret
						}(),
					},
				},
			},
			expectedGenerationTime: "a-test-5-time",
		},
		{
			name:    "cabundle created from ingress in-spec ssl secret",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							delete(secret.Labels, "cephdeployment.lcm.mirantis.com/self-signed-ssl-cert-for")
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-7-time"}
							secret.Data = map[string][]byte{"cabundle": []byte("spec-cacert\n")}
							return *secret
						}(),
					},
				},
			},
			expectedGenerationTime: "a-test-7-time",
			stateChanged:           true,
		},
		{
			name:    "cabundle created from mosk ssl secret, but it is not present, no base ssl, skipping",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{}},
			},
			expectedGenerationTime: "a-test-7-time",
		},
		{
			name:    "cabundle created from mosk ssl secret, but failed to get",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{}},
			},
			apiErrors:              map[string]error{"get-secrets-openstack-rgw-creds": errors.New("failed to get openstack-rgw-creds secret")},
			expectedGenerationTime: "a-test-7-time",
			expectedError:          "failed to get rgw creds secret 'openstack-ceph-shared/openstack-rgw-creds': failed to get openstack-rgw-creds secret",
		},
		{
			name:    "cabundle created from mosk ssl secret",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							delete(secret.Labels, "cephdeployment.lcm.mirantis.com/self-signed-ssl-cert-for")
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-10-time"}
							secret.Data = map[string][]byte{"cabundle": []byte(unitinputs.OpenstackCaCert + "\n")}
							return *secret
						}(),
					},
				},
			},
			stateChanged:           true,
			expectedGenerationTime: "a-test-10-time",
		},
		{
			name:    "cabundle nothing to do for mosk ssl secret",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-10-time"}
							secret.Data = map[string][]byte{"cabundle": []byte(unitinputs.OpenstackCaCert + "\n")}
							return *secret
						}(),
					},
				},
			},
			expectedGenerationTime: "a-test-10-time",
		},
		{
			name:    "cabundle create failed",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			apiErrors:              map[string]error{"create-secrets": errors.New("failed to create secret")},
			expectedGenerationTime: "a-test-10-time",
			expectedError:          "failed to create rgw cabundle cert secret: failed to create secret",
		},
		{
			name:    "cabundle create failed, failed to check base secret",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			apiErrors:              map[string]error{"get-secrets-rgw-store-ssl-cert": errors.New("failed to get rgw-store-ssl-cert secret")},
			expectedGenerationTime: "a-test-10-time",
			expectedError:          "failed to get secret 'rook-ceph/rgw-store-ssl-cert': failed to get rgw-store-ssl-cert secret",
		},
		{
			name: "cabundle create failed, base secret is not present when it specified",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.CephDeployMoskWithoutIngress.DeepCopy()
				rgwCasted, _ := dpl.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.SSLCertificateRef = "some-secret"
				dpl.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			expectedGenerationTime: "a-test-10-time",
			expectedError:          "failed to get secret 'rook-ceph/some-secret': secrets \"some-secret\" not found",
		},
		{
			name: "cabundle created, base secret is present and it specified",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.MultisiteRgwWithSyncDaemon.DeepCopy()
				rgwCasted, _ := dpl.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.SSLCertificateRef = "rgw-store-ssl-cert"
				dpl.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							delete(secret.Data, "cabundle")
							return *secret
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-15-time"}
							return *secret
						}(),
					},
				},
			},
			stateChanged:           true,
			expectedGenerationTime: "a-test-15-time",
		},
		{
			name: "cabundle nothing to do, base secret is present and it specified",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.MultisiteRgwWithSyncDaemon.DeepCopy()
				rgwCasted, _ := dpl.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.SSLCertificateRef = "rgw-store-ssl-cert"
				dpl.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RgwSSLCertSecret},
				},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RgwSSLCertSecret},
				},
			},
		},
		{
			name:    "cabundle created only from base default cert",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							delete(secret.Data, "cabundle")
							return *secret
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-17-time"}
							return *secret
						}(),
					},
				},
			},
			stateChanged:           true,
			expectedGenerationTime: "a-test-17-time",
		},
		{
			name:    "cabundle failed to create, base default cert corrupted",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							delete(secret.Data, "cacert")
							delete(secret.Data, "cabundle")
							return *secret
						}(),
					},
				},
			},
			expectedError:          "rgw 'rgw-store' seems to be must have cabundle, but it is not found",
			expectedGenerationTime: "a-test-17-time",
		},
	}

	oldGenerateCertFunc := lcmcommon.GenerateSelfSignedCert
	oldCurrentTimeFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)

			// test global var resourceUpdateTimestamps.RgwSSLCert is updated correctly
			// after each test run, it should be non-set if create/update is not happened
			// or existing cert has no annotation, should be set if create/update happened
			// or existing cert has annotation, and should not be changed in next test if no
			// create/update happened
			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("a-test-%d-time", idx)
			}

			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			rgwName := test.cephDpl.Spec.ObjectStorage.Rgws[test.rgwIdx].Name
			rgwCasted, _ := test.cephDpl.Spec.ObjectStorage.Rgws[test.rgwIdx].GetSpec()
			ingress := test.cephDpl.Spec.ObjectStorage.Rgws[test.rgwIdx].ServedByIngress
			rockoon := test.cephDpl.Spec.ObjectStorage.Rgws[test.rgwIdx].UsedByRockoon
			changed, err := c.ensureRgwCaBundleCert(rgwName, rgwCasted.Gateway.SSLCertificateRef, rgwCasted.Gateway.CaBundleRef, ingress, rockoon)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			assert.Equal(t, test.expectedGenerationTime, resourceUpdateTimestamps.rgwSSLCert[test.cephDpl.Spec.ObjectStorage.Rgws[test.rgwIdx].Name])
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.GenerateSelfSignedCert = oldGenerateCertFunc
	lcmcommon.GetCurrentTimeString = oldCurrentTimeFunc
	// unset global var to do not override with other tests
	unsetTimestampsVar()
}

func TestEnsureRgwSslCert(t *testing.T) {
	tests := []struct {
		name                   string
		cephDpl                *cephlcmv1alpha1.CephDeployment
		rgwIdx                 int
		inputResources         map[string]runtime.Object
		apiErrors              map[string]error
		stateChanged           bool
		expectedGenerationTime string
		expectedResources      map[string]runtime.Object
		expectedError          string
	}{
		{
			name:    "failed to create ssl certificate",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-secrets-rgw-store-ssl-cert": errors.New("failed to get rgw-store-ssl-cert secret")},
			expectedError: "failed to ensure rgw 'rgw-store' ssl certificate: failed to get secret rook-ceph/rgw-store-ssl-cert: failed to get rgw-store-ssl-cert secret",
		},
		{
			name:    "failed to create bundle certificate",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-1-time"}
						delete(secret.Data, "cabundle")
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-1-time",
			apiErrors:              map[string]error{"update-secrets-rgw-store-ssl-cert": errors.New("failed to get rgw-store-ssl-cert secret")},
			expectedError:          "failed to ensure rgw 'rgw-store' cabundle certificate: failed to update rgw cabundle cert secret: failed to get rgw-store-ssl-cert secret",
		},
		{
			name:    "ensure ssl certificate base, all updated",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-2-time"}
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-2-time",
			stateChanged:           true,
		},
		{
			name: "ensure ssl certificate base, ssl and labels updated only",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.CephDeployNonMosk.DeepCopy()
				rgwCasted, _ := dpl.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.CaBundleRef = "multisite-rgw-secret"
				dpl.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Labels = nil
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-2-time"}
						return *secret
					}(),
					unitinputs.MultisiteCabundleSecret,
				}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-3-time"}
						return *secret
					}(),
					unitinputs.MultisiteCabundleSecret,
				}},
			},
			expectedGenerationTime: "a-test-3-time",
			stateChanged:           true,
		},
		{
			name: "ensure ssl certificate base, cabundle updated only",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				dpl := unitinputs.CephDeployNonMosk.DeepCopy()
				rgwCasted, _ := dpl.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.SSLCertificateRef = "rgw-store-ssl-cert"
				dpl.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return dpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Data["cabundle"] = nil
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-2-time"}
						return *secret
					}(),
				}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-4-time"}
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-4-time",
			stateChanged:           true,
		},
		{
			name:    "ensure ssl certificate for ingress no changes",
			cephDpl: unitinputs.CephDeployMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "n-time"}
							secret.Data["cabundle"] = append(secret.Data["cabundle"], []byte("spec-cacert\n")...)
							return *secret
						}(),
					},
				},
			},
			expectedGenerationTime: "n-time",
		},
		{
			name:    "ensure ssl certificate for rockoon no changes",
			cephDpl: unitinputs.CephDeployMoskWithoutIngress.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.RgwSSLCertSecret.DeepCopy()
							secret.Data["cabundle"] = append(secret.Data["cabundle"], []byte(unitinputs.OpenstackCaCert+"\n")...)
							return *secret
						}(),
					},
				},
			},
		},
		{
			name:    "rgw cabundle required and no changes",
			rgwIdx:  1,
			cephDpl: unitinputs.MultisiteRgwWithSyncDaemon.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.MultisiteCabundleSecret}},
			},
		},
		{
			name:    "no any ssl certs required and no changes",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{},
			},
		},
		{
			name:    "ensure ssl certificate base, nothing to do",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.RgwSSLCertSecret}},
			},
		},
	}

	oldGenerateCertFunc := lcmcommon.GenerateSelfSignedCert
	oldCurrentTimeFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)

			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
				if _, ok := test.apiErrors["ssl-cert"]; ok {
					return "", "", "", errors.New("failed to generate ssl certificate")
				}
				return "fake-key", "fake-crt", "fake-ca", nil
			}

			// test global var resourceUpdateTimestamps.RgwSSLCert is updated correctly
			// after each test run, it should be non-set if create/update is not happened
			// or existing cert has no annotation, should be set if create/update happened
			// or existing cert has annotation, and should not be changed in next test if no
			// create/update happened
			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("a-test-%d-time", idx)
			}

			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureRgwSslCert(test.rgwIdx)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			assert.Equal(t, test.expectedGenerationTime, resourceUpdateTimestamps.rgwSSLCert[test.cephDpl.Spec.ObjectStorage.Rgws[test.rgwIdx].Name])
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.GenerateSelfSignedCert = oldGenerateCertFunc
	lcmcommon.GetCurrentTimeString = oldCurrentTimeFunc
	// unset global var to do not override with other tests
	unsetTimestampsVar()
}

func TestDeleteRgwAdminOpsSecret(t *testing.T) {
	inputResources := map[string]runtime.Object{
		"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RookCephRgwAdminSecret.DeepCopy()}},
	}
	tests := []struct {
		name          string
		deleted       bool
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:          "delete rgw ops admin keys - failed",
			apiErrors:     map[string]error{"delete-secrets": errors.New("secret delete failed")},
			expectedError: "failed to delete rgw admin ops secret rook-ceph/rgw-admin-ops-user: secret delete failed",
		},
		{
			name: "delete rgw ops admin keys - in progress",
		},
		{
			name:    "delete rgw ops admin keys - success",
			deleted: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, inputResources, test.apiErrors)

			deleted, err := c.deleteRgwAdminOpsSecret()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.deleted, deleted)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}
