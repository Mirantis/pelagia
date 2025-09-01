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
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	v1storage "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGenerateStorageClassPoolBased(t *testing.T) {
	tests := []struct {
		name        string
		cephDplPool cephlcmv1alpha1.CephPool
		isExternal  bool
		expected    *v1storage.StorageClass
	}{
		{
			name:        "generate storageclass - base",
			cephDplPool: unitinputs.GetCephDeployPool("pool1", "fake"),
			isExternal:  false,
			expected:    unitinputs.GetNamedStorageClass("pool1-hdd", false),
		},
		{
			name:        "generate storageclass - default",
			cephDplPool: unitinputs.CephDeployPoolReplicated,
			isExternal:  false,
			expected:    &unitinputs.BaseStorageClassDefault,
		},
		{
			name:        "generate storageclass - external base",
			cephDplPool: unitinputs.GetCephDeployPool("pool1", "fake"),
			isExternal:  true,
			expected:    unitinputs.GetNamedStorageClass("pool1-hdd", true),
		},
		{
			name: "generate storageclass - useAsFullName enabled",
			cephDplPool: func() cephlcmv1alpha1.CephPool {
				cephDplPoolStorageClassUseAsFullName := unitinputs.GetCephDeployPool("pool1", "fake")
				cephDplPoolStorageClassUseAsFullName.UseAsFullName = true
				return cephDplPoolStorageClassUseAsFullName
			}(),
			isExternal: false,
			expected: func() *v1storage.StorageClass {
				expectedStorageClassStandardUseAsFullName := unitinputs.GetNamedStorageClass("pool1", false)
				return expectedStorageClassStandardUseAsFullName
			}(),
		},
		{
			name: "generate storageclass - allowVolumeExtensions enabled",
			cephDplPool: func() cephlcmv1alpha1.CephPool {
				cephDplPoolStorageClassUseAsFullName := unitinputs.GetCephDeployPool("pool1", "fake")
				cephDplPoolStorageClassUseAsFullName.StorageClassOpts.AllowVolumeExpansion = true
				return cephDplPoolStorageClassUseAsFullName
			}(),
			isExternal: false,
			expected:   unitinputs.GetNamedStorageClass("pool1-hdd", true),
		},
		{
			name: "generate storageclass - rbd options present",
			cephDplPool: func() cephlcmv1alpha1.CephPool {
				cephDplPoolStorageClassMapOptions := unitinputs.GetCephDeployPool("pool1", "fake")
				cephDplPoolStorageClassMapOptions.StorageClassOpts.MapOptions = "nocephx_sign_messages,nocephx_require_signatures"
				cephDplPoolStorageClassMapOptions.StorageClassOpts.UnmapOptions = "force,noudev"
				cephDplPoolStorageClassMapOptions.StorageClassOpts.ImageFeatures = "layering,fast-diff,object-map"
				return cephDplPoolStorageClassMapOptions
			}(),
			isExternal: false,
			expected: func() *v1storage.StorageClass {
				expectedStorageClassStandardMapOptions := unitinputs.GetNamedStorageClass("pool1-hdd", false)
				expectedStorageClassStandardMapOptions.Parameters["mapOptions"] = "nocephx_sign_messages,nocephx_require_signatures"
				expectedStorageClassStandardMapOptions.Parameters["unmapOptions"] = "force,noudev"
				expectedStorageClassStandardMapOptions.Parameters["imageFeatures"] = "layering,fast-diff,object-map"
				return expectedStorageClassStandardMapOptions
			}(),
		},
		{
			name: "generate storageclass - reclaimPolicy present",
			cephDplPool: func() cephlcmv1alpha1.CephPool {
				mc := unitinputs.GetCephDeployPool("pool1", "fake")
				mc.StorageClassOpts.ReclaimPolicy = "Retain"
				return mc
			}(),
			isExternal: false,
			expected: func() *v1storage.StorageClass {
				sc := unitinputs.GetNamedStorageClass("pool1-hdd", false)
				rp := v1.PersistentVolumeReclaimRetain
				sc.ReclaimPolicy = &rp
				return sc
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := generateStorageClassPoolBased("rook-ceph", test.cephDplPool, "rook-ceph", test.isExternal)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestGenerateStorageClassForCephFS(t *testing.T) {
	actual := generateStorageClassCephFSBased(
		"rook-ceph",
		unitinputs.CephSharedFileSystemOk.CephFS[0].Name,
		unitinputs.CephSharedFileSystemOk.CephFS[0].DataPools[0].Name,
		"rook-ceph", unitinputs.CephSharedFileSystemOk.CephFS[0].PreserveFilesystemOnDelete)
	expected := &unitinputs.CephFSStorageClass
	assert.Equal(t, expected, actual)
}

func TestDeleteStorageClasses(t *testing.T) {
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		deleted        bool
		expectedError  string
	}{
		{
			name:           "delete storageclass - storageclass list failed",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get storage classes list: failed to list storageclasses",
		},
		{
			name: "delete storageclass - storageclass list empty, success",
			inputResources: map[string]runtime.Object{
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
			deleted: true,
		},
		{
			name: "delete storageclass - delete in progress, non-cephdeployment storageclasses skipped",
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						*unitinputs.CephFSStorageClass.DeepCopy(),
						func() v1storage.StorageClass {
							sc := unitinputs.GetNamedStorageClass("test", false)
							sc.Labels = nil
							return *sc
						}(),
					},
				},
				"persistentvolumes":      unitinputs.PersistentVolumeListEmpty.DeepCopy(),
				"persistentvolumeclaims": unitinputs.PersistentVolumeClaimListEmpty.DeepCopy(),
			},
			apiErrors: map[string]error{"delete-storageclasses-test": errors.New("unexpected test storageclass delete")},
		},
		{
			name: "delete storageclass - delete in progress, found pvc w/o device class",
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						*unitinputs.CephFSStorageClass.DeepCopy(),
					},
				},
				"persistentvolumes": unitinputs.PersistentVolumeListEmpty.DeepCopy(),
				"persistentvolumeclaims": &v1.PersistentVolumeClaimList{
					Items: []v1.PersistentVolumeClaim{
						{
							Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimBound},
						},
					},
				},
			},
		},
		{
			name: "delete storageclass - delete restricted, storageclass in use",
			inputResources: map[string]runtime.Object{
				"storageclasses":         unitinputs.StorageClassesList.DeepCopy(),
				"persistentvolumes":      unitinputs.PersistentVolumeList.DeepCopy(),
				"persistentvolumeclaims": unitinputs.PersistentVolumeClaimList.DeepCopy(),
			},
			apiErrors:     map[string]error{"delete-storageclasses": errors.New("unexpected storageclass delete")},
			expectedError: "delete storageclass(es) failed",
		},
		{
			name: "delete storageclass - delete failed",
			inputResources: map[string]runtime.Object{
				"storageclasses":         unitinputs.StorageClassesList.DeepCopy(),
				"persistentvolumes":      unitinputs.PersistentVolumeListEmpty.DeepCopy(),
				"persistentvolumeclaims": unitinputs.PersistentVolumeClaimListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"delete-storageclasses": errors.New("failed to delete storageclass")},
			expectedError: "delete storageclass(es) failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: &unitinputs.BaseCephDeployment}, nil)
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease

			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"storageclasses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"persistentvolumeclaims"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"persistentvolumes"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)

			deleted, err := c.deleteStorageClasses()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Contains(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.deleted, deleted)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
		})
	}
}

func TestEnsureStorageClasses(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		changed           bool
		expectedError     string
		expectedResources map[string]runtime.Object
	}{
		{
			name:           "ensure storage classes - failed to list storage classes, failure",
			cephDpl:        &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get storage classes list: failed to list storageclasses",
		},
		{
			name:    "ensure storage classes - create storageclasses failed, pool and cephfs are not ready yet",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"storageclasses": unitinputs.StorageClassesListEmpty.DeepCopy(),
			},
			expectedError: "failed to create storageclasses: multiple errors during storageclasses create",
		},
		{
			name:    "ensure storage classes - create storageclasses failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"storageclasses":  unitinputs.StorageClassesListEmpty.DeepCopy(),
				"cephfilesystems": unitinputs.CephFSListReady,
				"cephblockpools":  &unitinputs.CephBlockPoolListBaseReady,
			},
			expectedResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{unitinputs.CephFSStorageClass},
				},
			},
			apiErrors:     map[string]error{"create-storageclasses-pool1-hdd": errors.New("storageclass create failed")},
			expectedError: "failed to create storageclasses: failed to create StorageClass \"pool1-hdd\": storageclass create failed",
		},
		{
			name:    "ensure storage classes - create storageclasses completed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"storageclasses":  unitinputs.StorageClassesListEmpty.DeepCopy(),
				"cephfilesystems": unitinputs.CephFSListReady,
				"cephblockpools":  &unitinputs.CephBlockPoolListBaseReady,
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						unitinputs.BaseStorageClassDefault, unitinputs.CephFSStorageClass,
					},
				},
			},
		},
		{
			name:    "ensure storage classes - create storageclasses created for external, no checks",
			cephDpl: &unitinputs.CephDeployExternalCephFS,
			inputResources: map[string]runtime.Object{
				"storageclasses": unitinputs.StorageClassesListEmpty.DeepCopy(),
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						unitinputs.ExternalStorageClassDefault, unitinputs.CephFSStorageClass,
					},
				},
			},
		},
		{
			name:    "ensure storage classes - update, fail",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						*unitinputs.GetNamedStorageClass("pool1-hdd", false), unitinputs.CephFSStorageClass,
					},
				},
			},
			apiErrors:     map[string]error{"update-storageclasses": errors.New("failed to update storageclass")},
			expectedError: "failed to update storageclasses: failed to update StorageClass \"pool1-hdd\": failed to update storageclass",
		},
		{
			name: "ensure storage classes - update available fields only, success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployNonMosk.DeepCopy()
				cephDpl.Spec.Pools[0].StorageClassOpts.MapOptions = "some-opts"
				return cephDpl
			}(),
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						func() v1storage.StorageClass {
							sc := unitinputs.GetNamedStorageClass("pool1-hdd", false)
							sc.Labels = nil
							return *sc
						}(),
						func() v1storage.StorageClass {
							sc := unitinputs.CephFSStorageClass.DeepCopy()
							sc.Labels = nil
							return *sc
						}(),
					},
				},
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						unitinputs.BaseStorageClassDefault, unitinputs.CephFSStorageClass,
					},
				},
			},
		},
		{
			name:    "ensure storage classes - delete, success",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						*unitinputs.GetNamedStorageClass("pool1-hdd", false), unitinputs.CephFSStorageClass,
					},
				},
				"persistentvolumes":      &unitinputs.PersistentVolumeListEmpty,
				"persistentvolumeclaims": &unitinputs.PersistentVolumeClaimListEmpty,
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"storageclasses": &unitinputs.StorageClassesListEmpty,
			},
		},
		{
			name:    "ensure storage classes - delete with keep on spec, skip",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						func() v1storage.StorageClass {
							sc := unitinputs.CephFSStorageClass.DeepCopy()
							sc.Labels["rook-ceph-storage-class-keep-on-spec-remove"] = "true"
							return *sc
						}(),
					},
				},
				"persistentvolumes":      &unitinputs.PersistentVolumeListEmpty,
				"persistentvolumeclaims": &unitinputs.PersistentVolumeClaimListEmpty,
			},
		},
		{
			name:    "ensure storage classes - delete storageclass in use, failure",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						unitinputs.BaseStorageClassDefault,
					},
				},
				"persistentvolumes":      &unitinputs.PersistentVolumeList,
				"persistentvolumeclaims": &unitinputs.PersistentVolumeClaimList,
			},
			expectedError: "failed to delete storageclasses: delete storageclass(es) failed",
		},
		{
			name:    "ensure storage classes - delete failed, failure",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						unitinputs.BaseStorageClassDefault,
					},
				},
				"persistentvolumes":      &unitinputs.PersistentVolumeListEmpty,
				"persistentvolumeclaims": &unitinputs.PersistentVolumeClaimListEmpty,
			},
			apiErrors:     map[string]error{"delete-storageclasses": errors.New("storageclass delete failed")},
			expectedError: "failed to delete storageclasses: delete storageclass(es) failed",
		},
		{
			name:    "ensure storage classes - multiple errors for ensure",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						func() v1storage.StorageClass {
							sc := unitinputs.GetNamedStorageClass("pool1-hdd", false)
							sc.Labels = nil
							return *sc
						}(),
						*unitinputs.GetNamedStorageClass("pool2", false),
					},
				},
				"persistentvolumes":      &unitinputs.PersistentVolumeListEmpty,
				"persistentvolumeclaims": &unitinputs.PersistentVolumeClaimListEmpty,
			},
			apiErrors: map[string]error{
				"create-storageclasses": errors.New("storageclass create failed"),
				"update-storageclasses": errors.New("storageclass update failed"),
				"delete-storageclasses": errors.New("storageclass delete failed"),
			},
			expectedError: "multiple errors during storageclasses ensure",
		},
		{
			name:    "ensure storage classes - nothing to do",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"storageclasses": &v1storage.StorageClassList{
					Items: []v1storage.StorageClass{
						unitinputs.BaseStorageClassDefault, unitinputs.CephFSStorageClass,
					},
				},
				"cephfilesystems": unitinputs.CephFSListReady,
				"cephblockpools":  &unitinputs.CephBlockPoolListBaseReady,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"persistentvolumeclaims"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"persistentvolumes"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"storageclasses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "create", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "update", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephblockpools"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephfilesystems"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			scChanged, err := c.ensureStorageClasses()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.changed, scChanged)
			assert.Equal(t, test.expectedResources, test.inputResources)

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
