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
	"strings"
	"testing"

	"github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
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
		rgwSSLCert: "some-time",
		cephConfigMap: map[string]string{
			"global":                 "some-time",
			"client.rgw.rgw.store.a": "some-time",
		},
	}
	tests := []struct {
		name              string
		cephDplRGW        cephlcmv1alpha1.CephRGW
		useDedicatedNodes bool
		syncRgwDaemon     bool
		hyperconverge     *cephlcmv1alpha1.CephDeploymentHyperConverge
		expected          *cephv1.CephObjectStore
	}{
		{
			name:              "default rgw spec with pools, mon nodes placement, ssl enabled",
			cephDplRGW:        unitinputs.CephRgwBaseSpec,
			useDedicatedNodes: false,
			expected:          unitinputs.CephObjectStoreBase,
		},
		{
			name: "rgw spec override with health check, hyperconverge and dedicated nodes",
			cephDplRGW: func() cephlcmv1alpha1.CephRGW {
				rgw := unitinputs.CephRgwBaseSpec.DeepCopy()
				rgw.HealthCheck = &cephv1.ObjectHealthCheckSpec{
					StartupProbe: &cephv1.ProbeSpec{
						Probe: &v1.Probe{
							TimeoutSeconds:   10,
							FailureThreshold: 5,
						},
					},
				}
				rgw.PreservePoolsOnDelete = true
				rgw.DataPool = &cephlcmv1alpha1.CephPoolSpec{
					DeviceClass: "ssd",
					Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
						Size: 3,
					},
				}
				rgw.MetadataPool.FailureDomain = "rack"
				return *rgw
			}(),
			useDedicatedNodes: true,
			hyperconverge:     unitinputs.HyperConvergeForExtraSVC.DeepCopy(),
			expected: func() *cephv1.CephObjectStore {
				rgw := unitinputs.CephObjectStoreBase.DeepCopy()
				rgw.Spec.DataPool = cephv1.PoolSpec{
					DeviceClass: "ssd",
					Replicated: cephv1.ReplicatedSpec{
						TargetSizeRatio: 0.1,
						Size:            3,
					},
				}
				rgw.Spec.PreservePoolsOnDelete = true
				rgw.Spec.MetadataPool.FailureDomain = "rack"
				rgw.Spec.HealthCheck = cephv1.ObjectHealthCheckSpec{
					StartupProbe: &cephv1.ProbeSpec{
						Probe: &v1.Probe{
							TimeoutSeconds:   10,
							FailureThreshold: 5,
						},
					},
				}
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
					{
						Key:      "rgw-toleration",
						Operator: "Exists",
					},
				}
				rgw.Spec.Gateway.Resources = unitinputs.HyperConvergeForExtraSVC.Resources["rgw"]
				return rgw
			}(),
		},
		{
			name: "rgw spec with gateway overrides",
			cephDplRGW: func() cephlcmv1alpha1.CephRGW {
				rgw := unitinputs.CephRgwBaseSpec.DeepCopy()
				rgw.Gateway.Resources = &v1.ResourceRequirements{
					Limits:   unitinputs.ResourceListLimitsDefault,
					Requests: unitinputs.ResourceListRequestsDefault,
				}
				rgw.Gateway.Instances = 2
				rgw.Gateway.Port = 88
				rgw.Gateway.SecurePort = 444
				rgw.RgwUseHostNetwork = &[]bool{false}[0]
				//rgw.Gateway.SplitDaemonForMultisiteTrafficSync = true
				return *rgw
			}(),
			useDedicatedNodes: false,
			expected: func() *cephv1.CephObjectStore {
				rgw := unitinputs.CephObjectStoreBase.DeepCopy()
				rgw.Spec.Gateway.Resources = v1.ResourceRequirements{
					Limits:   unitinputs.ResourceListLimitsDefault,
					Requests: unitinputs.ResourceListRequestsDefault,
				}
				rgw.Spec.Gateway.Instances = 2
				rgw.Spec.Gateway.Port = 88
				rgw.Spec.Gateway.SecurePort = 444
				rgw.Spec.Gateway.HostNetwork = &[]bool{false}[0]
				//rgw.Spec.Gateway.DisableMultisiteSyncTraffic = true
				return rgw
			}(),
		},
		{
			name:              "multisite rgw run sync with single daemon",
			cephDplRGW:        unitinputs.CephDeployMultisiteMasterRgw.Spec.ObjectStorage.Rgw,
			useDedicatedNodes: false,
			expected:          unitinputs.CephObjectStoreWithZone,
		},
		{
			name:              "multisite rgw run sync with separate daemon, main rgw",
			cephDplRGW:        unitinputs.MultisiteRgwWithSyncDaemon.Spec.ObjectStorage.Rgw,
			useDedicatedNodes: false,
			expected: func() *cephv1.CephObjectStore {
				rgw := unitinputs.CephObjectStoreWithZone.DeepCopy()
				rgw.Spec.Zone.Name = "secondary-zone1"
				rgw.Spec.Gateway.DisableMultisiteSyncTraffic = true
				return rgw
			}(),
		},
		{
			name: "multisite rgw run sync with separate daemon, sync rgw",
			cephDplRGW: func() cephlcmv1alpha1.CephRGW {
				cd := unitinputs.MultisiteRgwWithSyncDaemon.Spec.ObjectStorage.Rgw.DeepCopy()
				cd.Gateway.RgwSyncPort = 8000
				return *cd
			}(),
			useDedicatedNodes: false,
			syncRgwDaemon:     true,
			expected: func() *cephv1.CephObjectStore {
				rgw := unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy()
				rgw.Spec.Gateway.Port = 8000
				return rgw
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.syncRgwDaemon {
				resourceUpdateTimestamps.cephConfigMap["client.rgw.rgw.store.sync.a"] = "some-time-sync"
			}
			actual := generateRgw(test.cephDplRGW, "rook-ceph", test.useDedicatedNodes, test.syncRgwDaemon, test.cephDplRGW.Zone == nil, test.hyperconverge)
			assert.Equal(t, test.expected, actual)
		})
	}
	unsetTimestampsVar()
}

func TestGenerateRgwBucket(t *testing.T) {
	actual := generateRgwBucket("rgw-storage-class", "fake-bucket", "rook-ceph")
	expected := unitinputs.GetSimpleBucket("fake-bucket")
	assert.Equal(t, *expected, actual)
}

func TestGenerateRgwUser(t *testing.T) {
	fakeUser := cephlcmv1alpha1.CephRGWUser{
		Name: "test-user",
	}
	actual := generateRgwUser("rgw-store", fakeUser, "rook-ceph")
	expected := unitinputs.RgwUserBase
	assert.Equal(t, expected, actual)
	fakeUserExtra := cephlcmv1alpha1.CephRGWUser{
		Name: "test-user",
		Capabilities: &cephv1.ObjectUserCapSpec{
			User: "*",
		},
		Quotas: &cephv1.ObjectUserQuotaSpec{
			MaxBuckets: &[]int{1}[0],
		},
	}
	actual = generateRgwUser("rgw-store", fakeUserExtra, "rook-ceph")
	expected = unitinputs.RgwUserWithCapsAndQuotas
	assert.Equal(t, expected, actual)
}

func TestGenerateRgwStorageClass(t *testing.T) {
	actual := generateRgwStorageClass("rgw-store", "rgw-storage-class", "rook-ceph", "rgw-store")
	expected := unitinputs.RgwStorageClass
	assert.Equal(t, expected, actual)
}

func TestGenerateRgwExternalService(t *testing.T) {
	labelSelector, err := metav1.ParseToLabelSelector("external_access=rgw")
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"external_access": "rgw"}, labelSelector.MatchLabels)
	rgwExternalSvc := generateRgwExternalService("rgw-store", "rook-ceph", labelSelector, &unitinputs.CephDeployObjectStorageCeph)
	assert.Equal(t, unitinputs.RgwExternalServiceGenerated, rgwExternalSvc)
}

func TestGenerateRgwExternal(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		rgwSSLCert: "some-time",
		cephConfigMap: map[string]string{
			"global":                 "some-time",
			"client.rgw.rgw.store.a": "some-time",
		},
	}
	tests := []struct {
		name          string
		cephDplRGW    cephlcmv1alpha1.CephRGW
		expected      *cephv1.CephObjectStore
		expectedError string
	}{
		{
			name:          "generate external rgw - no external endpoints, failed",
			cephDplRGW:    cephlcmv1alpha1.CephRGW{Name: "rgw-store"},
			expectedError: "external RGW endpoint is not specified for external ceph cluster",
		},
		{
			name:       "generate external rgw - success",
			cephDplRGW: unitinputs.RgwExternalSslEnabled,
			expected:   unitinputs.CephObjectStoreExternal,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := generateRgwExternal(test.cephDplRGW, "rook-ceph")
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
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase},
				},
			},
		},
		{
			name:    "ensure rgw consistence - multisite with sync daemon, no cleanup",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon,
						*unitinputs.CephObjectStoreWithZone,
					},
				},
			},
		},
		{
			name:    "ensure rgw consistence - multisite no sync daemon",
			cephDpl: &unitinputs.CephDeployMultisiteRgw,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon,
						*unitinputs.CephObjectStoreWithZone,
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectstores"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			err := c.ensureRgwConsistence()
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
		keepSyncStore     bool
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
			name: "delete rgw - delete cephobjectstore failed",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     unitinputs.CephObjectStoreListReady.DeepCopy(),
				"services":             &unitinputs.ServicesListEmpty,
			},
			apiErrors: map[string]error{
				"delete-cephobjectstores": errors.New("cephObjectStore delete failed"),
			},
			expectedError: "failed to cleanup rgw object store resources",
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
			name: "delete rgw - delete in progress",
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
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
				"services":         &unitinputs.ServicesListEmpty,
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{*unitinputs.RgwStorageClass.DeepCopy()},
				},
			},
		},
		{
			name: "delete rgw - delete rgw storageclass failed",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     &unitinputs.CephObjectStoreListEmpty,
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
			name: "delete rgw - delete rgw storageclass in progress",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores":     &unitinputs.CephObjectStoreListEmpty,
				"services":             &unitinputs.ServicesListEmpty,
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{*unitinputs.RgwStorageClass.DeepCopy()},
				},
			},
			expectedResources: map[string]runtime.Object{
				"storageclasses": &unitinputs.StorageClassesListEmpty,
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
			name: "delete rgw - multisite only sync daemon cleanup",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy(),
						*unitinputs.CephObjectStoreWithZone.DeepCopy(),
					},
				},
				"services": &v1.ServiceList{},
			},
			objectStoreName: "rgw-store",
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone},
				},
			},
		},
		{
			name: "delete rgw - multisite and sync daemon cleanup",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy(),
						*unitinputs.CephObjectStoreWithZone.DeepCopy(),
					},
				},
				"services": &v1.ServiceList{},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
		},
		{
			name: "delete rgw - multisite with sync daemon nothing to delete",
			inputResources: map[string]runtime.Object{
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						*unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy(),
						*unitinputs.CephObjectStoreWithZone.DeepCopy(),
					},
				},
				"services": &unitinputs.ServicesListEmpty,
			},
			objectStoreName: "rgw-store",
			keepSyncStore:   true,
			deleted:         true,
		},
	}
	cephAPIResources := []string{"cephobjectstoreusers", "cephobjectstores"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", cephAPIResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", cephAPIResources, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Claimclientset, "list", []string{"objectbucketclaims"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Claimclientset, "delete", []string{"objectbucketclaims"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			deleted, err := c.deleteRgw(test.objectStoreName, test.keepSyncStore)
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

			_, err := c.statusRgw()
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
			name: "ensure rgw storageclass - create success",
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

			changed, err := c.ensureRgwStorageClass()
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
				rgwSSLCert: "new-ssl-time",
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
					External: true,
					ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
						Rgw: cephlcmv1alpha1.CephRGW{Name: "rgw-store"},
					},
				},
			},
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RookCephRgwAdminSecret},
				},
			},
			expectedError: "failed to generate external rgw: external RGW endpoint is not specified for external ceph cluster",
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
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal.DeepCopy()},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.RookCephRgwAdminSecret},
				},
			},
			newTimestamps: &updateTimestamps{
				rgwSSLCert: "new-time",
				cephConfigMap: map[string]string{
					"global":                 "some-time",
					"client.rgw.rgw.store.a": "some-time",
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						func() cephv1.CephObjectStore {
							rgw := unitinputs.CephObjectStoreExternal.DeepCopy()
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "new-time"
							return *rgw
						}(),
					},
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
					ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
						Rgw: unitinputs.CephDeployMultisiteMasterRgw.Spec.ObjectStorage.Rgw,
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
					ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
						Rgw: unitinputs.CephDeployMultisiteMasterRgw.Spec.ObjectStorage.Rgw,
						MultiSite: &cephlcmv1alpha1.CephMultiSite{
							Zones: []cephlcmv1alpha1.CephRGWZone{
								{
									Name: "zone2",
								},
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
		},
		{
			name:    "ensure rgw - multisite rgw with sync daemon created",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &unitinputs.CephObjectStoreMultisiteSyncList,
			},
			changed: true,
		},
		{
			name:    "ensure rgw - multisite rgw with sync daemon updated",
			cephDpl: &unitinputs.MultisiteRgwWithSyncDaemon,
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
				rgwSSLCert: "new-ssl-time",
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
							rgw := unitinputs.CephObjectStoreWithZone.DeepCopy()
							rgw.Spec.Gateway.DisableMultisiteSyncTraffic = true
							rgw.Spec.Zone.Name = "secondary-zone1"
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "new-ssl-time"
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "new-rgw-time"
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "new-global-time"
							return *rgw
						}(),
						func() cephv1.CephObjectStore {
							rgw := unitinputs.CephObjectStoreWithSyncDaemon.DeepCopy()
							rgw.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/ssl-cert-generated"] = "new-ssl-time"
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
			inputResources: map[string]runtime.Object{
				"cephobjectstores": &unitinputs.CephObjectStoreMultisiteSyncList,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephobjectstores"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephobjectstores"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			if test.newTimestamps != nil {
				resourceUpdateTimestamps = *test.newTimestamps
			} else {
				resourceUpdateTimestamps = updateTimestamps{
					rgwSSLCert: "some-time",
					cephConfigMap: map[string]string{
						"global":                      "some-time",
						"client.rgw.rgw.store.a":      "some-time",
						"client.rgw.rgw.store.sync.a": "some-time-sync",
					},
				}
			}

			changed, err := c.ensureRgwObject()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.changed, changed)
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

			changed, err := c.ensureRgwUsers()
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

func TestEnsureRgwResourcesBuckets(t *testing.T) {
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		changed           bool
		expectedError     string
	}{
		{
			name:           "ensure rgw buckets - list buckets failed",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list rgw buckets: failed to list objectbucketclaims",
		},
		{
			name: "ensure rgw buckets - create failed, delete failed",
			inputResources: map[string]runtime.Object{
				"objectbucketclaims": &v1alpha1.ObjectBucketClaimList{
					Items: []v1alpha1.ObjectBucketClaim{
						{ObjectMeta: metav1.ObjectMeta{Namespace: "rook-ceph", Name: "bucket-1"}},
					},
				},
			},
			apiErrors: map[string]error{
				"create-objectbucketclaims": errors.New("bucket create failed"),
				"delete-objectbucketclaims": errors.New("bucket delete failed"),
			},
			expectedError: "failed to ensure rgw buckets, multiple errors during buckets ensure",
		},
		{
			name: "ensure rgw buckets - create success, delete success",
			inputResources: map[string]runtime.Object{
				"objectbucketclaims": &v1alpha1.ObjectBucketClaimList{
					Items: []v1alpha1.ObjectBucketClaim{
						{ObjectMeta: metav1.ObjectMeta{Namespace: "rook-ceph", Name: "bucket-1"}},
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"objectbucketclaims": &v1alpha1.ObjectBucketClaimList{
					Items: []v1alpha1.ObjectBucketClaim{
						*unitinputs.GetSimpleBucket("fake-bucket-1"), *unitinputs.GetSimpleBucket("fake-bucket-2"),
					},
				},
			},
			changed: true,
		},
		{
			name: "ensure rgw buckets - some are not ready for update",
			inputResources: map[string]runtime.Object{
				"objectbucketclaims": &v1alpha1.ObjectBucketClaimList{
					Items: []v1alpha1.ObjectBucketClaim{
						*unitinputs.GetSimpleBucket("fake-bucket-1"),
						func() v1alpha1.ObjectBucketClaim {
							bucket := unitinputs.GetSimpleBucket("fake-bucket-2")
							bucket.Status.Phase = v1alpha1.ObjectBucketClaimStatusPhasePending
							return *bucket
						}(),
					},
				},
			},
			expectedError: "failed to ensure rgw buckets: found not ready bucket rook-ceph/fake-bucket-2, waiting for readiness (current phase is Pending)",
		},
		{
			name: "ensure rgw buckets - nothing to update",
			inputResources: map[string]runtime.Object{
				"objectbucketclaims": &unitinputs.CephRgwBucketsList,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.CephDeployNonMosk.DeepCopy()}, nil)
			faketestclients.FakeReaction(c.api.Claimclientset, "list", []string{"objectbucketclaims"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Claimclientset, "create", []string{"objectbucketclaims"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Claimclientset, "delete", []string{"objectbucketclaims"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureRgwBuckets()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.changed, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Claimclientset)
		})
	}
}

func TestEnsureExternalService(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		labelSelector     string
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		changed           bool
		expectedError     string
	}{
		{
			// case should not happen at all - because label parsed in config controller
			// and used default in case of any problems
			name:          "ensure external service - incorrect label in pelagia config",
			cephDpl:       &unitinputs.CephDeployNonMosk,
			labelSelector: "e!!!!!ss",
			expectedError: "failed to parse provided rgw public access label 'e!!!!!ss': couldn't parse the selector string \"e!!!!!ss\": unable to parse requirement: found '!', expected: in, notin, =, ==, !=, gt, lt",
		},
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
			name:    "ensure external service - nothing todo",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"services": &v1.ServiceList{Items: []v1.Service{*unitinputs.RgwExternalServiceGenerated.DeepCopy()}},
			},
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
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			if test.labelSelector != "" {
				c.lcmConfig.DeployParams.RgwPublicAccessLabel = test.labelSelector
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"services"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureExternalService()
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

func TestEnsureRgw(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		rgwSSLCert: "some-time",
		cephConfigMap: map[string]string{
			"global":                 "some-time",
			"client.rgw.rgw.store.a": "some-time",
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
			expectedError: "failed to ensure rgw: failed to get object store: get cephobjectstore failed",
		},
		{
			name:    "ensure rgw - failed to ensure rgw secrets",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"secrets":          unitinputs.ServicesListEmpty.DeepCopy(),
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-secrets": errors.New("get secret failed")},
			expectedError: "failed to ensure rgw ssl cert: failed to get secret rook-ceph/rgw-ssl-certificate: get secret failed",
		},
		{
			name:    "ensure rgw - failed to ensure rgw object",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"secrets":          &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"create-cephobjectstores": errors.New("create cephobjectstore failed")},
			expectedError: "failed to ensure rgw object store: failed to create rgw: create cephobjectstore failed",
		},
		{
			name:    "ensure rgw - failed to ensure rgw storageclass, users, buckets and external service",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores": unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"secrets":          &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase}},
			},
			expectedError: "multiple errors during rgw ensure",
		},
		{
			name:    "ensure rgw - ensure rgw completed, all created",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreListEmpty.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"objectbucketclaims":   unitinputs.ObjectBucketClaimListEmpty.DeepCopy(),
				"secrets":              &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
				"storageclasses":       unitinputs.StorageClassesListEmpty.DeepCopy(),
				"services":             unitinputs.ServicesListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephobjectstores": &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase}},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{unitinputs.GetCephRgwUser("fake-user-1", "rook-ceph", "rgw-store"), unitinputs.GetCephRgwUser("fake-user-2", "rook-ceph", "rgw-store")},
				},
				"storageclasses":     &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":           &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
				"objectbucketclaims": &v1alpha1.ObjectBucketClaimList{Items: []v1alpha1.ObjectBucketClaim{*unitinputs.GetSimpleBucket("fake-bucket-1"), *unitinputs.GetSimpleBucket("fake-bucket-2")}},
			},
			changed: true,
		},
		{
			name:    "ensure rgw - ensure rgw, failed to check zone hostnames",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreBaseListReady.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephRgwUsersList.DeepCopy(),
				"objectbucketclaims":   unitinputs.CephRgwBucketsList.DeepCopy(),
				"secrets":              &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
				"storageclasses":       &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":             &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
			expectedError: "failed to ensure rgw zonegroup hostnames: failed to get zonegroups info for cluster 'rook-ceph/cephcluster': failed to run command 'radosgw-admin zonegroup get --rgw-zonegroup=rgw-store --format json': failed to find pod to run command: no pods found matching criteria (label(s): 'app=pelagia-ceph-toolbox') in namespace 'rook-ceph'",
		},
		{
			name: "ensure rgw - ensure rgw, nothing to do",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMosk.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.SkipAutoZoneGroupHostnameUpdate = true
				return cd
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreBaseListReady.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephRgwUsersList.DeepCopy(),
				"objectbucketclaims":   unitinputs.CephRgwBucketsList.DeepCopy(),
				"secrets":              &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
				"storageclasses":       &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":             &v1.ServiceList{Items: []v1.Service{unitinputs.RgwExternalServiceGenerated}},
			},
		},
		{
			name:    "ensure rgw - ensure rgw, external, nothing to do",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy(), *unitinputs.RookCephRgwAdminSecret.DeepCopy()},
				},
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListEmpty.DeepCopy(),
				"objectbucketclaims":   unitinputs.ObjectBucketClaimListEmpty.DeepCopy(),
				"storageclasses":       &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"cephobjectstores":     &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreExternal}},
			},
		},
		{
			name: "ensure rgw - ensure rgw, openstack, nothing to do",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployMosk.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.SkipAutoZoneGroupHostnameUpdate = true
				return cd
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectstores":     unitinputs.CephObjectStoreBaseListReady.DeepCopy(),
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListMetrics.DeepCopy(),
				"objectbucketclaims":   unitinputs.ObjectBucketClaimListEmpty.DeepCopy(),
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Data["cabundle"] = []byte(string(secret.Data["cabundle"]) + unitinputs.CephDeployMosk.Spec.IngressConfig.TLSConfig.TLSCerts.Cacert + "\n")
						return *secret
					}(),
				}},
				"storageclasses": &v1storage.StorageClassList{Items: []v1storage.StorageClass{unitinputs.RgwStorageClass}},
				"services":       unitinputs.ServicesListEmpty.DeepCopy(),
			},
		},
	}
	oldCrtFunc := lcmcommon.GenerateSelfSignedCert
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
				return "fake-key", "fake-crt", "fake-ca", nil
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

func TestEnsureRgwInternalSslCert(t *testing.T) {
	tests := []struct {
		name                   string
		cephDpl                *cephlcmv1alpha1.CephDeployment
		inputResources         map[string]runtime.Object
		apiErrors              map[string]error
		stateChanged           bool
		multisiteSecretRef     string
		expectedGenerationTime string
		expectedResources      map[string]runtime.Object
		expectedError          string
	}{
		{
			name:    "failed to get openstack shared secret",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-secrets-openstack-rgw-creds": errors.New("failed to get openstack-rgw-creds secret")},
			expectedError: "failed to get rgw creds secret openstack-ceph-shared/openstack-rgw-creds: failed to get openstack-rgw-creds secret",
		},
		{
			name:    "failed to get rgw ssl secret",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-secrets-rgw-ssl-certificate": errors.New("failed to get rgw-ssl-certificate secret")},
			expectedError: "failed to get secret rook-ceph/rgw-ssl-certificate: failed to get rgw-ssl-certificate secret",
		},
		{
			name:    "rgw ssl secret present no annotation and no in memory and no update required",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
			},
		},
		{
			name:    "rgw ssl secret present with ssl cert annotation and no update required",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						sc := unitinputs.RgwSSLCertSecret.DeepCopy()
						sc.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-prev-time"}
						return *sc
					}(),
				}}},
			expectedGenerationTime: "a-prev-time",
		},
		{
			name:    "rgw ssl secret present and openstack ca updated",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						*unitinputs.OpenstackRgwCredsSecret.DeepCopy(),
						*unitinputs.RgwSSLCertSecret.DeepCopy(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					*unitinputs.OpenstackRgwCredsSecret.DeepCopy(),
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-4-time"}
						secret.Data["cabundle"] = []byte(unitinputs.RgwCaCert + "\n" + unitinputs.OpenstackCaCert + "\n")
						return *secret
					}(),
				},
				},
			},
			expectedGenerationTime: "a-test-4-time",
			stateChanged:           true,
		},
		{
			name: "rgw ssl secret from spec failed to validate",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMosk.DeepCopy()
				mc.Spec.ObjectStorage.Rgw.SSLCert = &cephlcmv1alpha1.CephDeploymentCert{
					TLSKey:  "fake",
					TLSCert: "fake",
					Cacert:  "fake",
				}
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedGenerationTime: "a-test-4-time",
			expectedError:          "ssl verification failed for provided rgw ssl certs in spec: pem block in cacert is not found",
		},
		{
			name: "rgw ssl secret updated in ceph spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMosk.DeepCopy()
				mc.Spec.ObjectStorage.Rgw.SSLCert = &cephlcmv1alpha1.CephDeploymentCert{
					TLSKey:  unitinputs.OpenstackTLSKey,
					TLSCert: unitinputs.OpenstackTLSCert,
					Cacert:  unitinputs.OpenstackCaCert,
				}
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-6-time"}
						secret.Data["cert"] = []byte(unitinputs.OpenstackTLSKey + "\n" + unitinputs.OpenstackTLSCert + "\n" + unitinputs.OpenstackCaCert)
						secret.Data["cacert"] = []byte(unitinputs.OpenstackCaCert)
						secret.Data["cabundle"] = []byte(unitinputs.OpenstackCaCert + "\n")
						return *secret
					}(),
				},
				}},
			expectedGenerationTime: "a-test-6-time",
			stateChanged:           true,
		},
		{
			name:    "rgw ssl secret self-signed created",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-7-time"}
						secret.Data = unitinputs.RgwSSLCertSecretSelfSigned.Data
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-7-time",
			stateChanged:           true,
		},
		{
			name:    "rgw ssl secret renewed",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertExpiredSecret.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-8-time"}
						secret.Data = unitinputs.RgwSSLCertSecretSelfSigned.Data
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-8-time",
			stateChanged:           true,
		},
		{
			name:    "failed to get multisite cabundle ssl secret",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
			},
			multisiteSecretRef:     "extra-rook-ceph-cabundle",
			apiErrors:              map[string]error{"get-secrets-extra-rook-ceph-cabundle": errors.New("failed to get extra-rook-ceph-cabundle secret")},
			expectedGenerationTime: "a-test-8-time",
			expectedError:          "failed to get multisite cabundle secret rook-ceph/extra-rook-ceph-cabundle: failed to get extra-rook-ceph-cabundle secret",
		},
		{
			name:    "rgw ssl secret present, multisite cabundle present updated",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.MultisiteCabundleSecret, *unitinputs.RgwSSLCertSecret.DeepCopy()}},
			},
			multisiteSecretRef: "extra-rook-ceph-cabundle",
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.MultisiteCabundleSecret,
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-10-time"}
						secret.Data["cabundle"] = []byte(unitinputs.RgwCaCert + "\n" + "fake-extra-cabundle\n")
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-10-time",
			stateChanged:           true,
		},
		{
			name:    "rgw ssl secret present, multisite cabundle is not specified",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							s := unitinputs.MultisiteCabundleSecret.DeepCopy()
							s.Data["cabundle"] = nil
							return *s
						}(),
						*unitinputs.RgwSSLCertSecret.DeepCopy(),
					},
				}},
			multisiteSecretRef:     "extra-rook-ceph-cabundle",
			expectedGenerationTime: "a-test-10-time",
			expectedError:          "multisite cabundle secret rook-ceph/extra-rook-ceph-cabundle has no provided 'cabundle' or empty",
		},
		{
			name:    "rgw ssl secret self-signed create failed",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			apiErrors:              map[string]error{"create-secrets": errors.New("failed to create secret")},
			expectedGenerationTime: "a-test-10-time",
			expectedError:          "failed to create rgw ssl cert secret: failed to create secret",
		},
		{
			name:    "rgw ssl secret renew failed",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertExpiredSecret.DeepCopy()}},
			},
			apiErrors:              map[string]error{"update-secrets": errors.New("failed to update secret")},
			expectedGenerationTime: "a-test-10-time",
			expectedError:          "failed to update rgw ssl cert secret: failed to update secret",
		},
		{
			name: "rgw ssl secret is specified in secret directly, verification failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMosk.DeepCopy()
				mc.Spec.ObjectStorage.Rgw.SSLCertInRef = true
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertExpiredSecret.DeepCopy()}},
			},
			expectedGenerationTime: "a-test-10-time",
			expectedError:          "ssl verification failed for rgw ssl certs provided in 'rgw-ssl-certificate' secret, update manually: pkg cacert is expired",
		},
		{
			name: "rgw ssl secret is specified in secret directly, cabundle updated",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMosk.DeepCopy()
				mc.Spec.ObjectStorage.Rgw.SSLCertInRef = true
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.MultisiteCabundleSecret, *unitinputs.RgwSSLCertSecret.DeepCopy()}},
			},
			multisiteSecretRef: "extra-rook-ceph-cabundle",
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.MultisiteCabundleSecret,
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-15-time"}
						secret.Data["cabundle"] = []byte(unitinputs.RgwCaCert + "\n" + "fake-extra-cabundle\n")
						return *secret
					}(),
				}},
			},
			expectedGenerationTime: "a-test-15-time",
			stateChanged:           true,
		},
		{
			name: "failed to get ingress secret",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedGenerationTime: "a-test-15-time",
			expectedError:          "failed to get ingress secret rook-ceph/rgw-store-ingress-secret: secrets \"rgw-store-ingress-secret\" not found",
		},
		{
			name: "rgw ssl cert with ingress cabundle by ref, created",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.IngressRuleSecretCustom}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.IngressRuleSecretCustom,
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-17-time"}
						secret.Data["cabundle"] = []byte("fake-ca" + "\n" + "spec-cacert\n")
						secret.Data["cacert"] = []byte("fake-ca")
						secret.Data["cert"] = []byte("fake-keyfake-crtfake-ca")
						return *secret
					}(),
				}},
			},
			stateChanged:           true,
			expectedGenerationTime: "a-test-17-time",
		},
		{
			name:    "rgw ssl cert with ingress cabundle from spec, created",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-18-time"}
						secret.Data["cabundle"] = []byte("fake-ca" + "\n" + "spec-cacert\n")
						secret.Data["cacert"] = []byte("fake-ca")
						secret.Data["cert"] = []byte("fake-keyfake-crtfake-ca")
						return *secret
					}(),
				}},
			},
			stateChanged:           true,
			expectedGenerationTime: "a-test-18-time",
		},
		{
			name: "rgw ssl cert with ingress empty cert, openstack cabundle, created",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.OpenstackRgwCredsSecret,
					func() v1.Secret {
						secret := unitinputs.RgwSSLCertSecret.DeepCopy()
						secret.Annotations = map[string]string{sslCertGenerationTimestampLabel: "a-test-19-time"}
						secret.Data["cabundle"] = []byte("fake-ca" + "\n" + unitinputs.OpenstackCaCert + "\n")
						secret.Data["cacert"] = []byte("fake-ca")
						secret.Data["cert"] = []byte("fake-keyfake-crtfake-ca")
						return *secret
					}(),
				}},
			},
			stateChanged:           true,
			expectedGenerationTime: "a-test-19-time",
		},
	}

	oldGenerateCertFunc := lcmcommon.GenerateSelfSignedCert
	oldCurrentTimeFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, map[string]string{"DEPLOYMENT_MULTISITE_CABUNDLE_SECRET": test.multisiteSecretRef})
			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
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

			changed, err := c.ensureRgwInternalSslCert()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			assert.Equal(t, test.expectedGenerationTime, resourceUpdateTimestamps.rgwSSLCert)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.GenerateSelfSignedCert = oldGenerateCertFunc
	lcmcommon.GetCurrentTimeString = oldCurrentTimeFunc
	// unset global var to do not override with other tests
	unsetTimestampsVar()
}

func TestDeleteRgwInternalSslCert(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{rgwSSLCert: "some-time"}
	inputResources := map[string]runtime.Object{
		"secrets": &v1.SecretList{Items: []v1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy()}},
	}
	tests := []struct {
		name          string
		deleted       bool
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:          "delete rgw pkg ssl cert - failed",
			apiErrors:     map[string]error{"delete-secrets": errors.New("secret delete failed")},
			expectedError: "failed to delete rgw ssl cert secret rook-ceph/rgw-ssl-certificate: secret delete failed",
		},
		{
			name: "delete rgw pkg ssl cert - in progress",
		},
		{
			name:    "delete rgw pkg ssl cert - success",
			deleted: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, inputResources, test.apiErrors)

			deleted, err := c.deleteRgwInternalSslCert()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.deleted, deleted)
			if test.deleted {
				assert.Equal(t, "", resourceUpdateTimestamps.rgwSSLCert)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	unsetTimestampsVar()
}

func TestEnsureDefaultZoneGroupHostnames(t *testing.T) {
	tests := []struct {
		name               string
		cephDpl            *cephlcmv1alpha1.CephDeployment
		inputResources     map[string]runtime.Object
		apiErrors          map[string]error
		zonegroupInfo      string
		zonegroupUpdate    bool
		zonegroupUpdateCmd string
		zonegroupError     string
		expectedError      string
	}{
		{
			name: "skip auto update hostnames for zonegroup",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cd.Spec.ObjectStorage.Rgw.SkipAutoZoneGroupHostnameUpdate = true
				return cd
			}(),
		},
		{
			name:           "failed to get zonegroup info",
			cephDpl:        &unitinputs.CephDeployObjectStorageCeph,
			zonegroupError: "get",
			expectedError:  "failed to get zonegroups info for cluster 'rook-ceph/cephcluster': failed to run command 'radosgw-admin zonegroup get --rgw-zonegroup=rgw-store --format json': failed to get zonegroup info",
		},
		{
			name:           "failed to get os secret",
			cephDpl:        &unitinputs.CephDeployMoskWithoutIngress,
			zonegroupInfo:  unitinputs.CephZoneGroupInfoEmptyHostnames,
			inputResources: map[string]runtime.Object{"secrets": &unitinputs.ServicesListEmpty},
			apiErrors:      map[string]error{"get-secrets": errors.New("cant get secret")},
			expectedError:  "failed to get rgw creds secret openstack-rgw-creds: cant get secret",
		},
		{
			name:               "update failed",
			cephDpl:            &unitinputs.CephDeployObjectStorageCeph,
			zonegroupInfo:      unitinputs.CephZoneGroupInfoHostnamesFromConfig,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --unset",
			zonegroupError:     "update",
			expectedError:      "failed to update zonegroup 'rgw-store' hostnames for cluster 'rook-ceph/cephcluster': failed to run command '/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --unset': failed to update zonegroup info",
		},
		{
			name:           "no ingress, os secret is not found, no hostnames - no update",
			cephDpl:        &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{"secrets": &unitinputs.ServicesListEmpty},
			zonegroupInfo:  unitinputs.CephZoneGroupInfoEmptyHostnames,
		},
		{
			name:               "no ingress, os secret is not found, hostnames present - updated on empty",
			cephDpl:            &unitinputs.CephDeployObjectStorageCeph,
			zonegroupInfo:      unitinputs.CephZoneGroupInfoHostnamesFromConfig,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --unset",
		},
		{
			name:    "no ingress, os secret, no hostnames - updated on openstack fqdn",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecretNoBarbican}},
			},
			zonegroupInfo:      unitinputs.CephZoneGroupInfoEmptyHostnames,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --hostnames rgw-store.openstack.com,rook-ceph-rgw-rgw-store.rook-ceph.svc",
		},
		{
			name: "no ingress, os secret, spec override - updated on spec fqdn",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMoskWithoutIngress.DeepCopy()
				cephDpl.Spec.RookConfig = unitinputs.CephDeployMoskWithoutIngressRookConfigOverrideBarbican.Spec.RookConfig
				return cephDpl
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecretNoBarbican}},
			},
			zonegroupInfo:      unitinputs.CephZoneGroupInfoHostnamesFromOpenstack,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --hostnames rgw-store.ms2.wxlsd.com,rook-ceph-rgw-rgw-store.rook-ceph.svc",
		},
		{
			name:    "no ingress, os secret, hostnames present - no update",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecretNoBarbican}},
			},
			zonegroupInfo: unitinputs.CephZoneGroupInfoHostnamesFromOpenstack,
		},
		{
			name:               "ingress, no hostnames - updated on ingress fqdn",
			cephDpl:            &unitinputs.CephDeployMosk,
			zonegroupInfo:      unitinputs.CephZoneGroupInfoEmptyHostnames,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --hostnames rgw-store.test,rook-ceph-rgw-rgw-store.rook-ceph.svc",
		},
		{
			name: "ingress, spec override - updated on spec fqdn",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMosk.DeepCopy()
				cephDpl.Spec.RookConfig = unitinputs.CephDeployMoskWithoutIngressRookConfigOverrideBarbican.Spec.RookConfig
				return cephDpl
			}(),
			zonegroupInfo:      unitinputs.CephZoneGroupInfoHostnamesFromIngress,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --hostnames rgw-store.ms2.wxlsd.com,rook-ceph-rgw-rgw-store.rook-ceph.svc",
		},
		{
			name:          "ingress, hostnames present - no update",
			cephDpl:       &unitinputs.CephDeployMosk,
			zonegroupInfo: unitinputs.CephZoneGroupInfoHostnamesFromIngress,
		},
		{
			name: "no ingress, no os cert, spec override - updated on spec fqdn",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cephDpl.Spec.RookConfig = unitinputs.CephDeployMoskWithoutIngressRookConfigOverrideBarbican.Spec.RookConfig
				return cephDpl
			}(),
			zonegroupInfo:      unitinputs.CephZoneGroupInfoEmptyHostnames,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --hostnames rgw-store.ms2.wxlsd.com,rook-ceph-rgw-rgw-store.rook-ceph.svc",
		},
		{
			name: "no ingress, no os cert, spec override - no update",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cephDpl.Spec.RookConfig = unitinputs.CephDeployMoskWithoutIngressRookConfigOverrideBarbican.Spec.RookConfig
				return cephDpl
			}(),
			zonegroupInfo: unitinputs.CephZoneGroupInfoHostnamesFromConfig,
		},
		{
			name: "ingress, custom hostnames - updated on ingress fqdn",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				mc.Spec.IngressConfig.TLSConfig.Hostname = "test"
				mc.Spec.IngressConfig.Annotations = nil
				mc.Spec.IngressConfig.ControllerClassName = ""
				return mc
			}(),
			zonegroupInfo:      unitinputs.CephZoneGroupInfoEmptyHostnames,
			zonegroupUpdate:    true,
			zonegroupUpdateCmd: "/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup rgw-store --hostnames rook-ceph-rgw-rgw-store.rook-ceph.svc,test.example.com",
		},
	}

	updated := false
	oldCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			updated = false

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.Contains(e.Command, "radosgw-admin") {
					if test.zonegroupError == "get" {
						return "", "", errors.New("failed to get zonegroup info")
					}
					return test.zonegroupInfo, "", nil
				} else if strings.Contains(e.Command, "/usr/local/bin/zonegroup_hostnames_update.sh") {
					if !test.zonegroupUpdate {
						return "", "", errors.New("unexpected zonegroup update call")
					}
					updated = true
					assert.Equal(t, test.zonegroupUpdateCmd, e.Command)
					if test.zonegroupError == "update" {
						return "", "", errors.New("failed to update zonegroup info")
					}
					return "", "", nil
				}
				return "", "", errors.New("cant run ceph cmd: unknown command")
			}

			changed, err := c.ensureDefaultZoneGroupHostnames()
			assert.Equal(t, test.zonegroupUpdate, updated)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, changed)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.zonegroupUpdate, changed)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
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
