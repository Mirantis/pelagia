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
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestCephDeploymentVerification(t *testing.T) {
	nodesList := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}})
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		cephCliOutput  map[string]string
		daemonReport   map[string]string
		expectedStatus *lcmv1alpha1.CephDeploymentHealthReport
		foundIssues    []string
	}{
		{
			name: "no cephcluster present yet",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
				"deployments":  unitinputs.DeploymentListEmpty,
			},
			expectedStatus: &lcmv1alpha1.CephDeploymentHealthReport{
				RookOperator: unitinputs.RookOperatorStatusFailed,
			},
			foundIssues: []string{
				"failed to get 'rook-ceph-operator' deployment in 'rook-ceph' namespace",
				"cephcluster 'rook-ceph/cephcluster' object is not found",
			},
		},
		{
			name: "cephcluster external ok",
			inputResources: map[string]runtime.Object{
				"daemonsets":           unitinputs.DaemonSetListReady,
				"deployments":          unitinputs.DeploymentList,
				"configmaps":           unitinputs.ConfigMapList,
				"cephclusters":         &unitinputs.CephClusterListExternal,
				"cephblockpools":       &unitinputs.CephBlockPoolListEmpty,
				"cephclients":          &unitinputs.CephClientListEmpty,
				"cephobjectstores":     &unitinputs.CephObjectStoreListExternal,
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectrealms":     &unitinputs.CephObjectRealmListEmpty,
				"cephobjectzonegroups": &unitinputs.CephObjectZoneGroupListEmpty,
				"cephobjectzones":      &unitinputs.CephObjectZoneListEmpty,
				"cephfilesystems":      &unitinputs.CephFilesystemListEmpty,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json": unitinputs.CephDfBase,
				"ceph status -f json": unitinputs.BuildCliOutput(unitinputs.CephStatusTmpl, "status", map[string]string{
					"servicemap": `{"services": {"rgw": {"daemons": {"11556688": {"gid": 11556688},"12065099":{"gid": 12065099},"summary": ""}}}}`,
				}),
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
			},
			expectedStatus: unitinputs.CephExternalClusterReportOk,
			foundIssues:    []string{},
		},
		{
			name: "cephcluster external issues",
			inputResources: map[string]runtime.Object{
				"daemonsets":  unitinputs.DaemonSetListEmpty,
				"deployments": unitinputs.DeploymentList,
				"configmaps":  unitinputs.ConfigMapList,
				"cephclusters": func() *cephv1.CephClusterList {
					list := unitinputs.CephClusterListExternal.DeepCopy()
					list.Items[0].Status.Phase = cephv1.ConditionFailure
					return list
				}(),
				"cephblockpools":       &unitinputs.CephBlockPoolListEmpty,
				"cephclients":          &unitinputs.CephClientListEmpty,
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":   &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectrealms":     &unitinputs.CephObjectRealmListEmpty,
				"cephobjectzonegroups": &unitinputs.CephObjectZoneGroupListEmpty,
				"cephobjectzones":      &unitinputs.CephObjectZoneListEmpty,
				"cephfilesystems":      &unitinputs.CephFilesystemListEmpty,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfBase,
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
			},
			expectedStatus: func() *lcmv1alpha1.CephDeploymentHealthReport {
				report := unitinputs.CephExternalClusterReportOk.DeepCopy()
				report.RookCephObjects.CephCluster.Phase = cephv1.ConditionFailure
				report.RookCephObjects.ObjectStorage = nil
				report.ClusterDetails.CephEvents = nil
				report.ClusterDetails.RgwInfo = nil
				report.CephDaemons.CephDaemons = nil
				report.CephDaemons.CephCSIPluginDaemons = map[string]lcmv1alpha1.DaemonStatus{
					"csi-rbdplugin": {
						Status: lcmv1alpha1.DaemonStateFailed,
						Issues: []string{"daemonset 'rook-ceph/csi-rbdplugin' is not found"},
					},
					"csi-cephfsplugin": {
						Status: lcmv1alpha1.DaemonStateFailed,
						Issues: []string{"daemonset 'rook-ceph/csi-cephfsplugin' is not found"},
					},
				}
				return report
			}(),
			foundIssues: []string{
				"cephcluster 'rook-ceph/cephcluster' object state is 'Failure'",
				"daemonset 'rook-ceph/csi-cephfsplugin' is not found",
				"daemonset 'rook-ceph/csi-rbdplugin' is not found",
				"failed to list cephobjectstores in 'rook-ceph' namespace",
				"failed to run 'ceph status -f json' command to check daemons status",
				"failed to run 'ceph status -f json' command to check events details",
			},
		},
		{
			name: "cephcluster base ok",
			inputResources: map[string]runtime.Object{
				"daemonsets":           unitinputs.DaemonSetListReady,
				"deployments":          unitinputs.DeploymentList,
				"configmaps":           unitinputs.ConfigMapList,
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
				"nodes":                &nodesList,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfBase,
				"ceph status -f json":              unitinputs.CephStatusBaseHealthy,
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
				"ceph osd metadata -f json":        unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":            unitinputs.CephOsdInfoOutput,
			},
			daemonReport: map[string]string{
				"node-1": unitinputs.CephDiskDaemonDiskReportStringNode1,
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			expectedStatus: unitinputs.CephBaseClusterReportOk,
			foundIssues:    []string{},
		},
		{
			name: "cephcluster base issues",
			inputResources: map[string]runtime.Object{
				"daemonsets":           unitinputs.DaemonSetListNotReady,
				"deployments":          unitinputs.DeploymentList,
				"configmaps":           unitinputs.ConfigMapList,
				"cephclusters":         &unitinputs.CephClusterListHealthIssues,
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
			cephCliOutput: map[string]string{
				"ceph df -f json":           unitinputs.CephDfBase,
				"ceph status -f json":       unitinputs.CephStatusBaseUnhealthy,
				"ceph mgr dump -f json":     unitinputs.CephMgrDumpBaseUnhealthy,
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
			},
			expectedStatus: unitinputs.CephBaseClusterReportNotOk,
			foundIssues:    unitinputs.CephDeploymentHealthStatusNotOk.Status.Issues,
		},
		{
			name: "cephcluster extra multisite ok",
			inputResources: map[string]runtime.Object{
				"daemonsets":           unitinputs.DaemonSetListReady,
				"deployments":          unitinputs.DeploymentList,
				"configmaps":           unitinputs.ConfigMapList,
				"ingresses":            &unitinputs.IngressesList,
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
				"nodes":                &nodesList,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfExtraPools,
				"ceph status -f json":              unitinputs.CephStatusCephFewFsRgwHealthy,
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
				"ceph osd metadata -f json":        unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":            unitinputs.CephOsdInfoOutput,
				"radosgw-admin sync status --rgw-zonegroup=zonegroup-1 --rgw-zone=zone-1": unitinputs.RadosgwAdminMasterSyncStatusOk,
			},
			daemonReport: map[string]string{
				"node-1": unitinputs.CephDiskDaemonDiskReportStringNode1,
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			expectedStatus: unitinputs.CephMultisiteClusterReportOk,
			foundIssues:    []string{},
		},
		{
			name: "cephcluster multisite issues",
			inputResources: map[string]runtime.Object{
				"daemonsets":           unitinputs.DaemonSetListReady,
				"deployments":          unitinputs.DeploymentList,
				"configmaps":           unitinputs.ConfigMapList,
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
				"nodes":                &nodesList,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfExtraPools,
				"ceph status -f json":              unitinputs.CephStatusCephFewFsRgwUnhealthy,
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
				"ceph osd metadata -f json":        unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":            unitinputs.CephOsdInfoOutput,
			},
			daemonReport: map[string]string{
				"node-1": "{||}",
				"node-2": "{||}",
			},
			expectedStatus: func() *lcmv1alpha1.CephDeploymentHealthReport {
				report := unitinputs.CephMultisiteClusterReportOk.DeepCopy()
				report.RookCephObjects = unitinputs.RookCephObjectsReportReadyOnlyCephCluster
				report.CephDaemons.CephDaemons["mds"] = lcmv1alpha1.DaemonStatus{
					Status: lcmv1alpha1.DaemonStateFailed,
					Messages: []string{
						"mds active: 0/1 (cephfs 'cephfs-1')", "mds active: 1/1, standby-replay: 0/1 (cephfs 'cephfs-2')",
					},
					Issues: []string{
						"unexpected mds daemons running (CephFS 'cephfs-3')",
						"unexpected number (0/1) of mds active are running for CephFS 'cephfs-1'",
						"unexpected number (0/1) of mds standby are running",
						"unexpected number (0/1) of mds standby-replay are running for CephFS 'cephfs-2'",
					},
				}
				report.CephDaemons.CephDaemons["rgw"] = lcmv1alpha1.DaemonStatus{
					Status:   lcmv1alpha1.DaemonStateFailed,
					Messages: []string{"4 rgws running, daemons: [10223488 11556688 12065099 12065109]"},
					Issues:   []string{"unexpected rgws (4/3) rgws are running"},
				}
				report.OsdAnalysis = unitinputs.OsdSpecAnalysisNotOk
				report.ClusterDetails.RgwInfo.PublicEndpoint = ""
				report.ClusterDetails.RgwInfo.MultisiteDetails = unitinputs.CephMultisiteStateFailed
				// TODO: drop after full merge cephdeployment/rook resources inputs
				report.ClusterDetails.RgwInfo.MultisiteDetails.Messages = []string{"failed to run 'radosgw-admin sync status --rgw-zonegroup=zonegroup-1 --rgw-zone=zone-1' command to check multisite status for zone 'zone-1'"}
				return report
			}(),
			foundIssues: []string{
				"cephblockpool 'rook-ceph/pool1' is not ready",
				"cephblockpool 'rook-ceph/pool2' status is not available yet",
				"cephclient 'rook-ceph/client1' is not ready",
				"cephclient 'rook-ceph/client2' status is not available yet",
				"cephfilesystem 'rook-ceph/cephfs-1' is not ready",
				"cephfilesystem 'rook-ceph/cephfs-2' status is not available yet",
				"cephobjectrealm 'rook-ceph/realm-1' is not ready",
				"cephobjectrealm 'rook-ceph/realm-2' status is not available yet",
				"cephobjectstore 'rook-ceph/rgw-store' endpoint is not found",
				"cephobjectstore 'rook-ceph/rgw-store' is not ready",
				"cephobjectstore 'rook-ceph/rgw-store-sync' status is not available yet",
				"cephobjectuser 'rook-ceph/rgw-user-1' is not ready",
				"cephobjectuser 'rook-ceph/rgw-user-2' status is not available yet",
				"cephobjectzone 'rook-ceph/zone-1' is not ready",
				"cephobjectzone 'rook-ceph/zone-2' status is not available yet",
				"cephobjectzonegroup 'rook-ceph/zonegroup-1' is not ready",
				"cephobjectzonegroup 'rook-ceph/zonegroup-2' status is not available yet",
				"failed to check ingresses in 'rook-ceph' namespace",
				"failed to run 'radosgw-admin sync status --rgw-zonegroup=zonegroup-1 --rgw-zone=zone-1' command to check multisite status for zone 'zone-1'",
				"node 'node-1' has failed spec analyse",
				"node 'node-2' has failed spec analyse",
				"objectbucketclaim 'rook-ceph/bucket-1' is not ready",
				"unexpected mds daemons running (CephFS 'cephfs-3')",
				"unexpected number (0/1) of mds active are running for CephFS 'cephfs-1'",
				"unexpected number (0/1) of mds standby are running",
				"unexpected number (0/1) of mds standby-replay are running for CephFS 'cephfs-2'",
				"unexpected rgws (4/3) rgws are running",
			},
		},
	}

	oldRunFunc := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hc := getEmtpyHealthConfig()
			c := fakeCephReconcileConfig(&hc, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxAndDiskDaemonPodsList}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"ingresses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", rookListResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Claimclientset, "list", claimListResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", rookGetResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments", "daemonsets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, test.inputResources, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --full-report --port 9999" {
					if test.daemonReport != nil {
						return test.daemonReport[e.Nodename], "", nil
					}
				} else if output, ok := test.cephCliOutput[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("failed command")
			}

			status, issues := c.cephDeploymentVerification()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.foundIssues, issues)

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Claimclientset)
		})
	}
	lcmcommon.RunPodCommand = oldRunFunc
}

func TestCheckRookOperator(t *testing.T) {
	tests := []struct {
		name           string
		deploy         *appsv1.Deployment
		maintenance    *lcmv1alpha1.CephDeploymentMaintenance
		expectedStatus lcmv1alpha1.DaemonStatus
	}{
		{
			name:           "failed to get rook-ceph-operator deployment",
			expectedStatus: unitinputs.RookOperatorStatusFailed,
		},
		{
			name:   "failed to check maintenance status",
			deploy: unitinputs.RookDeploymentNotScaled,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{"failed to check CephDeploymentMaintenance state"},
			},
		},
		{
			name:        "rook operator deployment is not ready",
			deploy:      unitinputs.RookDeploymentNotScaled,
			maintenance: &unitinputs.CephDeploymentMaintenanceIdle,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{"deployment 'rook-ceph-operator' is not ready"},
			},
		},
		{
			name:        "maintenance is in progress",
			deploy:      unitinputs.RookDeploymentNotScaled,
			maintenance: &unitinputs.CephDeploymentMaintenanceActing,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status:   lcmv1alpha1.DaemonStateFailed,
				Messages: []string{"deployment 'rook-ceph-operator' is scaled down due to maintenance mode"},
			},
		},
		{
			name:           "rook operator deployment is ready",
			deploy:         &unitinputs.DeploymentList.Items[0],
			expectedStatus: unitinputs.RookOperatorStatusOk,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			res := map[string]runtime.Object{"deployments": &appsv1.DeploymentList{}, "cephdeploymentmaintenances": &lcmv1alpha1.CephDeploymentMaintenanceList{}}
			if test.deploy != nil {
				res["deployments"] = &appsv1.DeploymentList{Items: []appsv1.Deployment{*test.deploy}}
			}
			if test.maintenance != nil {
				res["cephdeploymentmaintenances"] = &lcmv1alpha1.CephDeploymentMaintenanceList{Items: []lcmv1alpha1.CephDeploymentMaintenance{*test.maintenance}}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, res, nil)
			faketestclients.FakeReaction(c.api.Lcmclientset, "get", []string{"cephdeploymentmaintenances"}, res, nil)

			status := c.checkRookOperator()
			assert.Equal(t, test.expectedStatus, status)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Lcmclientset)
		})
	}
}
