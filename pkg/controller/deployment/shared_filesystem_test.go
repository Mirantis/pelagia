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
	"strings"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureSharedFilesystem(t *testing.T) {
	okCephFs := unitinputs.BaseCephDeployment.DeepCopy()
	okCephFs.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		changed           bool
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:           "fail to ensure shared filesystems",
			cephDpl:        okCephFs,
			inputResources: map[string]runtime.Object{},
			expectedError:  "errors faced during Ceph shared filesystems ensure",
		},
		{
			name:    "shared filesystems is not set",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephfilesystems": unitinputs.CephFilesystemListEmpty.DeepCopy(),
			},
		},
		{
			name:    "ensure shared filesystems completes",
			cephDpl: okCephFs,
			inputResources: map[string]runtime.Object{
				"cephfilesystems": unitinputs.CephFilesystemListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephfilesystems": unitinputs.CephFSList.DeepCopy(),
			},
			changed: true,
		},
		{
			name:    "ensure no changed",
			cephDpl: okCephFs,
			inputResources: map[string]runtime.Object{
				"cephfilesystems": unitinputs.CephFSList.DeepCopy(),
			},
		},
		{
			name:    "ensure no shared filesystems, removing",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephfilesystems": unitinputs.CephFSList.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephfilesystems": unitinputs.CephFilesystemListEmpty.DeepCopy(),
			},
			changed: true,
		},
		{
			name:    "ensure no shared filesystems, nothing to remove",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephfilesystems": unitinputs.CephFilesystemListEmpty.DeepCopy(),
			},
		},
	}

	oldCephCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			resourceUpdateTimestamps = updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "some-time",
					"mds":    "some-time",
				},
			}
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.HasPrefix(e.Command, "ceph fs subvolumegroup -f json ls") {
					if test.changed {
						return "[]", "", nil
					}
					return `[{"name":"csi"}]`, "", nil
				}
				if strings.HasPrefix(e.Command, "ceph fs subvolumegroup -f json create") {
					return "", "", nil
				}
				if strings.HasPrefix(e.Command, "ceph fs subvolumegroup -f json rm") {
					return "", "", nil
				}
				return "", "", errors.Errorf("unexpected command '%v'", e.Command)
			}

			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephfilesystems"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephfilesystems"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephfilesystems"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephfilesystems"}, test.inputResources, nil)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureSharedFilesystem()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.changed, changed)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCephCmdFunc
	unsetTimestampsVar()
}

func TestEnsureCephFS(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		cephConfigMap: map[string]string{
			"global": "some-time",
			"mds":    "some-time",
		},
	}
	cephDpl := unitinputs.BaseCephDeployment.DeepCopy()
	// updated case
	updateSharedFs := unitinputs.CephSharedFileSystemOk.DeepCopy()
	updateCephFS := updateSharedFs.CephFS[0]
	updateCephFS.MetadataPool.Replicated.Size = 1
	updateSharedFs.CephFS[0] = updateCephFS
	updatedCephFS := unitinputs.TestCephFs.DeepCopy()
	updatedCephFS.Spec.MetadataPool.Replicated.Size = 1
	// delete case
	delSharedFs := unitinputs.CephSharedFileSystemOk.DeepCopy()
	delSharedFs.CephFS = make([]cephlcmv1alpha1.CephFS, 0)
	tests := []struct {
		name                  string
		sharedFs              *cephlcmv1alpha1.CephSharedFilesystem
		cephFilesystemList    *cephv1.CephFilesystemList
		expectedCephFsList    *cephv1.CephFilesystemList
		apiErrors             map[string]error
		subvolumegroupCommand map[string]string
		stateChanged          bool
		expectedError         string
		// to reflect ceph.conf changes
		configUpdated bool
	}{
		{
			name:          "fail to list present cephfs",
			sharedFs:      unitinputs.CephSharedFileSystemOk,
			expectedError: "failed to get CephFS list: failed to list cephfilesystems",
		},
		{
			name:               "fail to check cephfs is present",
			sharedFs:           unitinputs.CephSharedFileSystemOk,
			cephFilesystemList: unitinputs.CephFilesystemListEmpty.DeepCopy(),
			expectedCephFsList: &unitinputs.CephFilesystemListEmpty,
			apiErrors:          map[string]error{"get-cephfilesystems": errors.New("failed to get")},
			expectedError:      "failed to get CephFS rook-ceph/test-cephfs: failed to get",
		},
		{
			name:               "create new cephfs",
			sharedFs:           unitinputs.CephSharedFileSystemOk,
			cephFilesystemList: unitinputs.CephFilesystemListEmpty.DeepCopy(),
			expectedCephFsList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{unitinputs.TestCephFs}},
			subvolumegroupCommand: map[string]string{
				"ls": "[]",
			},
			stateChanged: true,
		},
		{
			name:               "fail to create new cephfs",
			sharedFs:           unitinputs.CephSharedFileSystemOk,
			cephFilesystemList: unitinputs.CephFilesystemListEmpty.DeepCopy(),
			expectedCephFsList: &unitinputs.CephFilesystemListEmpty,
			apiErrors:          map[string]error{"create-cephfilesystems": errors.New("failed to create")},
			expectedError:      "failed to create CephFS rook-ceph/test-cephfs: failed to create",
		},
		{
			name:               "fail to create cephfs subvolumegroup",
			sharedFs:           unitinputs.CephSharedFileSystemOk,
			cephFilesystemList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{*unitinputs.TestCephFs.DeepCopy()}},
			expectedCephFsList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{unitinputs.TestCephFs}},
			subvolumegroupCommand: map[string]string{
				"ls":     "[]",
				"create": "error",
			},
			expectedError: "failed to create CephFS test-cephfs subvolumegroup: failed to run command 'ceph fs subvolumegroup -f json create test-cephfs csi' (stdErr: ENOENT: error): ceph fs subvolumegroup -f json create command failed",
		},
		{
			name:               "create only cephfs subvolumegroup",
			sharedFs:           unitinputs.CephSharedFileSystemOk,
			cephFilesystemList: unitinputs.CephFilesystemListEmpty.DeepCopy(),
			expectedCephFsList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{unitinputs.TestCephFs}},
			subvolumegroupCommand: map[string]string{
				"ls": "[]",
			},
			stateChanged: true,
		},
		{
			name:               "fail to check cephfs subvolumegroup",
			sharedFs:           unitinputs.CephSharedFileSystemOk,
			cephFilesystemList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{*unitinputs.TestCephFs.DeepCopy()}},
			expectedCephFsList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{unitinputs.TestCephFs}},
			subvolumegroupCommand: map[string]string{
				"ls": "error",
			},
			expectedError: "failed to list CephFS test-cephfs subvolumegroup: failed to run command 'ceph fs subvolumegroup -f json ls test-cephfs' (stdErr: ENOENT: error): ceph fs subvolumegroup -f json ls command failed",
		},
		{
			name:               "fail to update existing cephfs",
			sharedFs:           updateSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: unitinputs.CephFSList,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
			apiErrors:     map[string]error{"update-cephfilesystems": errors.New("failed to update")},
			expectedError: "failed to update CephFS rook-ceph/test-cephfs: failed to update",
		},
		{
			name:               "update existing cephfs",
			sharedFs:           updateSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{*updatedCephFS}},
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
			stateChanged: true,
		},
		{
			name:               "update annotations for existing cephfs",
			sharedFs:           updateSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: &cephv1.CephFilesystemList{
				Items: []cephv1.CephFilesystem{
					func() cephv1.CephFilesystem {
						cephfs := updatedCephFS.DeepCopy()
						cephfs.Spec.MetadataServer.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "some-new-time"
						cephfs.Spec.MetadataServer.Annotations["cephdeployment.lcm.mirantis.com/config-mds-updated"] = "some-new-time"
						cephfs.Spec.MetadataServer.Annotations["cephdeployment.lcm.mirantis.com/config-mds.test-cephfs-updated"] = "some-new-time"
						return *cephfs
					}(),
				},
			},
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
			configUpdated: true,
			stateChanged:  true,
		},
		{
			name:               "do not update cephfs if nothing changed",
			sharedFs:           unitinputs.CephSharedFileSystemOk.DeepCopy(),
			cephFilesystemList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{unitinputs.TestCephFs}},
			expectedCephFsList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{unitinputs.TestCephFs}},
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
		},
		{
			name:               "fail to check cephfs subvolumegroup presence for delete",
			sharedFs:           delSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: unitinputs.CephFSList,
			subvolumegroupCommand: map[string]string{
				"ls": "error",
			},
			expectedError: "failed to list CephFS test-cephfs subvolumegroup: failed to run command 'ceph fs subvolumegroup -f json ls test-cephfs' (stdErr: ENOENT: error): ceph fs subvolumegroup -f json ls command failed",
		},
		{
			name:               "fail to remove cephfs subvolumegroup",
			sharedFs:           delSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: unitinputs.CephFSList,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
				"rm": "error",
			},
			expectedError: "failed to remove CephFS test-cephfs subvolumegroup: failed to run command 'ceph fs subvolumegroup -f json rm test-cephfs csi' (stdErr: ENOENT: error): ceph fs subvolumegroup -f json rm command failed",
		},
		{
			name:               "fail to delete existing cephfs",
			sharedFs:           delSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: unitinputs.CephFSList,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
			apiErrors:     map[string]error{"delete-cephfilesystems": errors.New("failed to delete")},
			expectedError: "failed to delete CephFS rook-ceph/test-cephfs: failed to delete",
		},
		{
			name:               "delete existing cephfs",
			sharedFs:           delSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: &unitinputs.CephFilesystemListEmpty,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
			stateChanged: true,
		},
		{
			name:               "delete existing cephfs skip removing cephfs subvolumegroup on cephfs delete",
			sharedFs:           delSharedFs,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: &unitinputs.CephFilesystemListEmpty,
			subvolumegroupCommand: map[string]string{
				"ls": `[]`,
				"rm": "error",
			},
			stateChanged: true,
		},
		{
			name:               "create few cephfs",
			sharedFs:           unitinputs.CephSharedFileSystemMultiple,
			cephFilesystemList: unitinputs.CephFSList.DeepCopy(),
			expectedCephFsList: &cephv1.CephFilesystemList{
				Items: []cephv1.CephFilesystem{
					unitinputs.TestCephFs,
					func() cephv1.CephFilesystem {
						cephfs := unitinputs.TestCephFs.DeepCopy()
						cephfs.Name = "second-test-cephfs"
						term := cephfs.Spec.MetadataServer.Placement.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0]
						term.LabelSelector.MatchExpressions[0].Values = []string{"second-test-cephfs"}
						cephfs.Spec.MetadataServer.Placement.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0] = term
						return *cephfs
					}(),
				},
			},
			subvolumegroupCommand: map[string]string{
				"ls": `[]`,
			},
			stateChanged: true,
		},
		{
			name:     "remove few cephfs",
			sharedFs: delSharedFs,
			cephFilesystemList: &cephv1.CephFilesystemList{
				Items: []cephv1.CephFilesystem{
					unitinputs.TestCephFs,
					func() cephv1.CephFilesystem {
						cephfs := unitinputs.TestCephFs.DeepCopy()
						cephfs.Name = "second-test-cephfs"
						return *cephfs
					}(),
				},
			},
			expectedCephFsList: &unitinputs.CephFilesystemListEmpty,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
			stateChanged: true,
		},
		{
			name:     "update existing cephfs and remove another",
			sharedFs: updateSharedFs,
			cephFilesystemList: &cephv1.CephFilesystemList{
				Items: []cephv1.CephFilesystem{
					unitinputs.TestCephFs,
					func() cephv1.CephFilesystem {
						cephfs := unitinputs.TestCephFs.DeepCopy()
						cephfs.Name = "second-test-cephfs"
						return *cephfs
					}(),
				},
			},
			expectedCephFsList: &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{*updatedCephFS}},
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
			},
			stateChanged: true,
		},
		{
			name:               "multiple errors during cephfs ensure",
			sharedFs:           unitinputs.CephSharedFileSystemMultiple,
			cephFilesystemList: unitinputs.CephFilesystemListEmpty.DeepCopy(),
			expectedCephFsList: &unitinputs.CephFilesystemListEmpty,
			apiErrors:          map[string]error{"create-cephfilesystems": errors.New("failed to create")},
			expectedError:      "multiple errors during cephFS ensure",
		},
	}

	oldCephCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: cephDpl}, nil)
			cephDpl.Spec.SharedFilesystem = test.sharedFs
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease
			if test.configUpdated {
				resourceUpdateTimestamps.cephConfigMap["global"] = "some-new-time"
				resourceUpdateTimestamps.cephConfigMap["mds"] = "some-new-time"
				resourceUpdateTimestamps.cephConfigMap["mds.test-cephfs"] = "some-new-time"
			}

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				commands := map[string]string{
					"ceph fs subvolumegroup -f json ls":     "ls",
					"ceph fs subvolumegroup -f json create": "create",
					"ceph fs subvolumegroup -f json rm":     "rm",
				}
				for cmd, op := range commands {
					if strings.HasPrefix(e.Command, cmd) {
						if test.subvolumegroupCommand[op] == "error" {
							t.Logf("ERROR ON CREATE")
							return "", "ENOENT: error", errors.Errorf("%s command failed", cmd)
						}
						return test.subvolumegroupCommand[op], "", nil
					}
				}
				return "", "", errors.Errorf("unexpected command '%v'", e.Command)
			}

			inputResources := map[string]runtime.Object{"cephfilesystems": test.cephFilesystemList}
			if test.cephFilesystemList == nil {
				inputResources = nil
			}
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephfilesystems"}, inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephfilesystems"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephfilesystems"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephfilesystems"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephfilesystems"}, inputResources, test.apiErrors)

			changed, err := c.ensureCephFS()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			assert.Equal(t, test.expectedCephFsList, test.cephFilesystemList)

			// revert updating timestamps of config is necessary
			if test.configUpdated {
				resourceUpdateTimestamps.cephConfigMap["global"] = "some-time"
				resourceUpdateTimestamps.cephConfigMap["mds"] = "some-time"
				delete(resourceUpdateTimestamps.cephConfigMap, "mds.test-cephfs")
			}
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCephCmdFunc
	unsetTimestampsVar()
}

func TestDeleteSharedFilesystems(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		cephConfigMap: map[string]string{
			"mds": "some-time",
		},
	}
	tests := []struct {
		name                  string
		cephDpl               *cephlcmv1alpha1.CephDeployment
		cephFsList            *cephv1.CephFilesystemList
		expectedList          *cephv1.CephFilesystemList
		removed               bool
		apiErrors             map[string]error
		subvolumegroupCommand map[string]string
		expectedError         string
	}{
		{
			name:    "fail list cephfs",
			cephDpl: &unitinputs.BaseCephDeployment,
			subvolumegroupCommand: map[string]string{
				"ls": "[]",
			},
			expectedError: "failed to get cephFS list: failed to list cephfilesystems",
		},
		{
			name:         "failed to delete ceph fs",
			cephDpl:      &unitinputs.BaseCephDeployment,
			cephFsList:   unitinputs.CephFSList.DeepCopy(),
			expectedList: unitinputs.CephFSList,
			apiErrors:    map[string]error{"delete-cephfilesystems": errors.New("failed to delete")},
			subvolumegroupCommand: map[string]string{
				"ls": "[]",
			},
			expectedError: "some CephFS failed to delete",
		},
		{
			name:    "shared filesystems removing",
			cephDpl: &unitinputs.BaseCephDeployment,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
				"rm": "",
			},
			cephFsList:   unitinputs.CephFSList.DeepCopy(),
			expectedList: &unitinputs.CephFilesystemListEmpty,
		},
		{
			name:    "multiple shared filesystems removing",
			cephDpl: &unitinputs.BaseCephDeployment,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
				"rm": "",
			},
			cephFsList: &cephv1.CephFilesystemList{
				Items: []cephv1.CephFilesystem{
					unitinputs.TestCephFs,
					func() cephv1.CephFilesystem {
						cephfs := unitinputs.TestCephFs.DeepCopy()
						cephfs.Name = "second-test-cephfs"
						return *cephfs
					}(),
				},
			},
			expectedList: &unitinputs.CephFilesystemListEmpty,
		},
		{
			name:    "fail to check cephfs subvolumegroup on cephfs removing",
			cephDpl: &unitinputs.BaseCephDeployment,
			subvolumegroupCommand: map[string]string{
				"ls": "error",
			},
			cephFsList:    unitinputs.CephFSList.DeepCopy(),
			expectedError: "some CephFS failed to delete",
			expectedList:  unitinputs.CephFSList,
		},
		{
			name:    "fail to delete cephfs subvolumegroup on cephfs removing",
			cephDpl: &unitinputs.BaseCephDeployment,
			subvolumegroupCommand: map[string]string{
				"ls": `[{"name":"csi"}]`,
				"rm": "error",
			},
			cephFsList:    unitinputs.CephFSList.DeepCopy(),
			expectedError: "some CephFS failed to delete",
			expectedList:  unitinputs.CephFSList,
		},
		{
			name:         "nothing to remove",
			cephDpl:      &unitinputs.BaseCephDeployment,
			cephFsList:   unitinputs.CephFilesystemListEmpty.DeepCopy(),
			expectedList: &unitinputs.CephFilesystemListEmpty,
			subvolumegroupCommand: map[string]string{
				"ls": `[]`,
				"rm": "",
			},
			removed: true,
		},
	}

	oldCephCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				commands := map[string]string{
					"ceph fs subvolumegroup -f json ls": "ls",
					"ceph fs subvolumegroup -f json rm": "rm",
				}
				for cmd, op := range commands {
					if strings.HasPrefix(e.Command, cmd) {
						if test.subvolumegroupCommand[op] == "error" {
							t.Logf("ERROR ON CREATE")
							return "", "ENOENT: error", errors.Errorf("%s command failed", cmd)
						}
						return test.subvolumegroupCommand[op], "", nil
					}
				}
				return "", "", errors.Errorf("unexpected command '%v'", e.Command)
			}

			inputResources := map[string]runtime.Object{"cephfilesystems": test.cephFsList}
			if test.cephFsList == nil {
				inputResources = nil
			}
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephfilesystems"}, inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephfilesystems"}, inputResources, test.apiErrors)

			done, err := c.deleteSharedFilesystems()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.removed, done)
			assert.Equal(t, test.expectedList, test.cephFsList)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCephCmdFunc
	unsetTimestampsVar()
}

func TestGenerateCephFilesystem(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		cephConfigMap: map[string]string{
			"global":          "some-time",
			"mds":             "some-time",
			"mds.test-cephfs": "some-time",
		},
	}
	simpleCephFS := unitinputs.BaseCephDeployment.DeepCopy()
	simpleCephFS.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
	// get tolerations from hyper and resources from metadata server spec
	cephFSHyperTolerations := unitinputs.BaseCephDeployment.DeepCopy()
	cephFSHyperTolerations.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{
		CephFS: []cephlcmv1alpha1.CephFS{*unitinputs.CephFSOkWithResources.DeepCopy()},
	}
	cephFSHyperTolerations.Spec.HyperConverge = unitinputs.HyperConvergeForExtraSVC.DeepCopy()
	// get tolerations and resources from hyper
	cephFSHyperTolerationsAndResources := unitinputs.BaseCephDeployment.DeepCopy()
	cephFSHyperTolerationsAndResources.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
	cephFSHyperTolerationsAndResources.Spec.HyperConverge = unitinputs.HyperConvergeForExtraSVC.DeepCopy()
	expectedCephFsHyper := unitinputs.TestCephFsWithTolerationsAndResources.DeepCopy()
	expectedCephFsHyper.Spec.MetadataServer.Resources = unitinputs.HyperConvergeForExtraSVC.DeepCopy().Resources["mds"]

	cephFSProbes := unitinputs.BaseCephDeployment.DeepCopy()
	cephFSProbes.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
	cephfs := cephFSProbes.Spec.SharedFilesystem.CephFS[0]
	probe := &cephv1.ProbeSpec{
		Disabled: true,
	}
	cephfs.MetadataServer.HealthCheck = &cephlcmv1alpha1.CephMdsHealthCheck{
		LivenessProbe: probe,
		StartupProbe:  probe,
	}
	cephFSProbes.Spec.SharedFilesystem.CephFS[0] = cephfs
	cephFsWithDaemonAnnotations := unitinputs.TestCephFs.DeepCopy()
	cephFsWithDaemonAnnotations.Spec.MetadataServer.Annotations["cephdeployment.lcm.mirantis.com/config-mds.test-cephfs-updated"] = "some-time"
	expectedCephFSProbes := cephFsWithDaemonAnnotations.DeepCopy()
	expectedCephFSProbes.Spec.MetadataServer.LivenessProbe = probe
	expectedCephFSProbes.Spec.MetadataServer.StartupProbe = probe
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		expectedCephFS *cephv1.CephFilesystem
	}{
		{
			name:           "generate cephfs no extra sections",
			cephDpl:        simpleCephFS,
			expectedCephFS: cephFsWithDaemonAnnotations,
		},
		{
			name:           "generate cephfs with resources from ceph fs",
			cephDpl:        cephFSHyperTolerations,
			expectedCephFS: &unitinputs.TestCephFsWithTolerationsAndResources,
		},
		{
			name:           "generate cephfs with resources from hyper",
			cephDpl:        cephFSHyperTolerationsAndResources,
			expectedCephFS: expectedCephFsHyper,
		},
		{
			name:           "generate cephfs with liveness startup probes",
			cephDpl:        cephFSProbes,
			expectedCephFS: expectedCephFSProbes,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultFS := generateCephFS(test.cephDpl.Spec.SharedFilesystem.CephFS[0], "rook-ceph", test.cephDpl.Spec.HyperConverge)
			assert.Equal(t, test.expectedCephFS, resultFS)
		})
	}
	unsetTimestampsVar()
}
