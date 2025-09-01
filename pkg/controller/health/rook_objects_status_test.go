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

package health

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestRookObjectsVerification(t *testing.T) {
	basehc := getEmtpyHealthConfig()
	tests := []struct {
		name                 string
		inputResources       map[string]runtime.Object
		expectedStatus       *lcmv1alpha1.RookCephObjectsStatus
		expectedIssues       []string
		expectedHealthConfig *healthConfig
	}{
		{
			name:           "cant check cephcluster version",
			inputResources: map[string]runtime.Object{},
			expectedIssues: []string{"failed to get cephcluster 'rook-ceph/cephcluster' object"},
		},
		{
			name: "failed to list rook ceph resources",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			expectedStatus: &lcmv1alpha1.RookCephObjectsStatus{
				CephCluster: &unitinputs.ReefCephClusterReady.Status,
			},
			expectedIssues: []string{
				"failed to list cephblockpools in 'rook-ceph' namespace",
				"failed to list cephclients in 'rook-ceph' namespace",
				"failed to list cephobjectstores in 'rook-ceph' namespace",
				"failed to list cephobjectusers in 'rook-ceph' namespace",
				"failed to list objectbucketclaims in 'rook-ceph' namespace",
				"failed to list cephobjectrealms in 'rook-ceph' namespace",
				"failed to list cephobjectzonegroups in 'rook-ceph' namespace",
				"failed to list cephobjectzones in 'rook-ceph' namespace",
				"failed to list cephfilesystems in 'rook-ceph' namespace",
			},
			expectedHealthConfig: &healthConfig{
				name:                 "cephcluster",
				namespace:            "lcm-namespace",
				cephCluster:          &unitinputs.ReefCephClusterReady,
				sharedFilesystemOpts: basehc.sharedFilesystemOpts,
			},
		},
		{
			name: "no ceph rook resources are present, except ceph cluster",
			inputResources: map[string]runtime.Object{
				"cephclusters":         &unitinputs.CephClusterListReady,
				"cephblockpools":       &unitinputs.CephBlockPoolListEmpty,
				"cephclients":          &unitinputs.CephClientListEmpty,
				"cephobjectstores":     &unitinputs.CephObjectStoreListEmpty,
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectrealms":     &unitinputs.CephObjectRealmListEmpty,
				"cephobjectzonegroups": &unitinputs.CephObjectZoneGroupListEmpty,
				"cephobjectzones":      &unitinputs.CephObjectZoneListEmpty,
				"cephfilesystems":      &unitinputs.CephFilesystemListEmpty,
			},
			expectedStatus: unitinputs.RookCephObjectsReportOnlyCephCluster,
			expectedIssues: []string{},
			expectedHealthConfig: &healthConfig{
				name:                 "cephcluster",
				namespace:            "lcm-namespace",
				cephCluster:          &unitinputs.ReefCephClusterReady,
				sharedFilesystemOpts: basehc.sharedFilesystemOpts,
			},
		},
		{
			name: "ceph rook resources are not ready",
			inputResources: map[string]runtime.Object{
				"cephclusters":         &unitinputs.CephClusterListReady,
				"cephblockpools":       &unitinputs.CephBlockPoolListNotReady,
				"cephclients":          &unitinputs.CephClientListNotReady,
				"cephobjectstores":     &unitinputs.CephObjectStoresMultisiteSyncDaemonPhaseNotReady,
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListNotReady,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListNotReady,
				"cephobjectrealms":     &unitinputs.CephObjectRealmListNotReady,
				"cephobjectzonegroups": &unitinputs.CephObjectZoneGroupListNotReady,
				"cephobjectzones":      &unitinputs.CephObjectZoneListNotReady,
				"cephfilesystems":      &unitinputs.CephFilesystemListMultipleNotReady,
			},
			expectedStatus: unitinputs.RookCephObjectsReportReadyOnlyCephCluster,
			expectedIssues: []string{
				"cephblockpool 'rook-ceph/pool1' is not ready",
				"cephblockpool 'rook-ceph/pool2' status is not available yet",
				"cephclient 'rook-ceph/client1' is not ready",
				"cephclient 'rook-ceph/client2' status is not available yet",
				"cephobjectstore 'rook-ceph/rgw-store' is not ready",
				"cephobjectstore 'rook-ceph/rgw-store-sync' status is not available yet",
				"cephobjectuser 'rook-ceph/rgw-user-1' is not ready",
				"cephobjectuser 'rook-ceph/rgw-user-2' status is not available yet",
				"objectbucketclaim 'rook-ceph/bucket-1' is not ready",
				"cephobjectrealm 'rook-ceph/realm-1' is not ready",
				"cephobjectrealm 'rook-ceph/realm-2' status is not available yet",
				"cephobjectzonegroup 'rook-ceph/zonegroup-1' is not ready",
				"cephobjectzonegroup 'rook-ceph/zonegroup-2' status is not available yet",
				"cephobjectzone 'rook-ceph/zone-1' is not ready",
				"cephobjectzone 'rook-ceph/zone-2' status is not available yet",
				"cephfilesystem 'rook-ceph/cephfs-1' is not ready",
				"cephfilesystem 'rook-ceph/cephfs-2' status is not available yet",
			},
			expectedHealthConfig: &healthConfig{
				name:        "cephcluster",
				namespace:   "lcm-namespace",
				cephCluster: &unitinputs.ReefCephClusterReady,
				rgwOpts: rgwOpts{
					storeName:         "rgw-store",
					desiredRgwDaemons: 3,
					multisite:         true,
				},
				sharedFilesystemOpts: sharedFilesystemOpts{
					mdsStandbyDesired: 1,
					mdsDaemonsDesired: map[string]map[string]int{
						"cephfs-1": {"up:active": 1},
						"cephfs-2": {"up:active": 1, "up:standby-replay": 1},
					},
				},
			},
		},
		{
			name: "ceph rook resources are ready",
			inputResources: map[string]runtime.Object{
				"cephclusters":         &unitinputs.CephClusterListReady,
				"cephblockpools":       &unitinputs.CephBlockPoolListReady,
				"cephclients":          &unitinputs.CephClientListReady,
				"cephobjectstores":     &unitinputs.CephObjectStoresMultisiteSyncDaemonPhaseReady,
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListReady,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListReady,
				"cephobjectrealms":     &unitinputs.CephObjectRealmListReady,
				"cephobjectzonegroups": &unitinputs.CephObjectZoneGroupListReady,
				"cephobjectzones":      &unitinputs.CephObjectZoneListReady,
				"cephfilesystems":      &unitinputs.CephFilesystemListMultipleReady,
			},
			expectedStatus: unitinputs.RookCephObjectsReportReadyFull,
			expectedIssues: []string{},
			expectedHealthConfig: &healthConfig{
				name:        "cephcluster",
				namespace:   "lcm-namespace",
				cephCluster: &unitinputs.ReefCephClusterReady,
				rgwOpts: rgwOpts{
					storeName:         "rgw-store",
					desiredRgwDaemons: 3,
					multisite:         true,
				},
				sharedFilesystemOpts: sharedFilesystemOpts{
					mdsStandbyDesired: 1,
					mdsDaemonsDesired: map[string]map[string]int{
						"cephfs-1": {"up:active": 1},
						"cephfs-2": {"up:active": 1, "up:standby-replay": 1},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", rookListResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", rookGetResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Claimclientset, "list", claimListResources, test.inputResources, nil)

			status, issues := c.rookObjectsVerification()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)

			hc := basehc
			if test.expectedHealthConfig != nil {
				hc = *test.expectedHealthConfig
			}
			assert.Equal(t, hc, c.healthConfig)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Claimclientset)
		})
	}
}

func TestСheckCephCluster(t *testing.T) {
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		apiError       bool
		expectedStatus *cephv1.ClusterStatus
		expectedIssues []string
	}{
		{
			name:           "failed to get cephcluster object",
			apiError:       true,
			expectedIssues: []string{"failed to get cephcluster 'rook-ceph/cephcluster' object"},
		},
		{
			name: "cephcluster object is not found",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			expectedIssues: []string{"cephcluster 'rook-ceph/cephcluster' object is not found"},
		},
		{
			name: "cephcluster object has no status version",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListNotReady,
			},
			expectedIssues: []string{"cephcluster is creating, no valid cephcluster version in status"},
		},
		{
			name: "cephcluster object has unsupported ceph version",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListNotSupported,
			},
			expectedIssues: []string{"verification is supported since Ceph Pacific versions (v16.2), current is '15.2.8-0'"},
		},
		{
			name: "cephcluster object has health info is not available",
			inputResources: map[string]runtime.Object{
				"cephclusters": func() *cephv1.CephClusterList {
					list := unitinputs.CephClusterListHealthIssues.DeepCopy()
					list.Items[0].Status.CephStatus = nil
					return list
				}(),
			},
			expectedStatus: func() *cephv1.ClusterStatus {
				status := unitinputs.ReefCephClusterHasHealthIssues.DeepCopy().Status
				status.CephStatus = nil
				return &status
			}(),
			expectedIssues: []string{
				"cephcluster 'rook-ceph/cephcluster' object state is 'Failure'",
				"cephcluster 'rook-ceph/cephcluster' object health info is not available",
			},
		},
		{
			name: "cephcluster object has health issues and not ready",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListHealthIssues,
			},
			expectedStatus: &unitinputs.ReefCephClusterHasHealthIssues.Status,
			expectedIssues: []string{
				"cephcluster 'rook-ceph/cephcluster' object state is 'Failure'",
				"RECENT_MGR_MODULE_CRASH: 2 mgr modules have recently crashed",
				"cephcluster 'rook-ceph/cephcluster' object status is not updated for last 5 minutes",
			},
		},
		{
			name: "cephcluster object has health issues which are ignored",
			inputResources: map[string]runtime.Object{
				"cephclusters": func() *cephv1.CephClusterList {
					list := unitinputs.CephClusterListReady.DeepCopy()
					list.Items[0].Status.CephStatus.Details = map[string]cephv1.CephHealthMessage{
						"RECENT_CRASH": {
							Message:  "4 daemons have recently crashed",
							Severity: "HEALTH_WARN",
						},
					}
					return list
				}(),
			},
			expectedStatus: func() *cephv1.ClusterStatus {
				status := unitinputs.ReefCephClusterReady.DeepCopy().Status
				status.CephStatus.Details = map[string]cephv1.CephHealthMessage{
					"RECENT_CRASH": {
						Message:  "4 daemons have recently crashed",
						Severity: "HEALTH_WARN",
					},
				}
				return &status
			}(),
			expectedIssues: []string{},
		},
		{
			name: "cephcluster object is ready",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			expectedStatus: &unitinputs.ReefCephClusterReady.Status,
			expectedIssues: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			apiErrors := map[string]error{}
			if test.apiError {
				apiErrors = map[string]error{"get-cephclusters": errors.New("failed to get cephcluster")}
			}
			faketestclients.FakeReaction(c.api.Rookclientset, "get", rookGetResources, test.inputResources, apiErrors)

			status, issues := c.checkCephCluster()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)
			hc := getEmtpyHealthConfig()
			if list, ok := test.inputResources["cephclusters"]; ok && list != nil {
				items := list.(*cephv1.CephClusterList).Items
				if len(items) > 0 && test.expectedStatus != nil {
					hc.cephCluster = &items[0]
				}
			}
			assert.Equal(t, hc, c.healthConfig)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
