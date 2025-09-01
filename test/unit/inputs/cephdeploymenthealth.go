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

package input

import (
	bktv1alpha1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var CephDeploymentHealth = lcmv1alpha1.CephDeploymentHealth{
	ObjectMeta: LcmObjectMeta,
}

var CephDeploymentHealthStatusOk = lcmv1alpha1.CephDeploymentHealth{
	ObjectMeta: LcmObjectMeta,
	Status: lcmv1alpha1.CephDeploymentHealthStatus{
		State:            lcmv1alpha1.HealthStateOk,
		HealthReport:     CephBaseClusterReportOk,
		LastHealthCheck:  "time-now-fake",
		LastHealthUpdate: "time-now-fake",
	},
}

var CephDeploymentHealthStatusNotOk = lcmv1alpha1.CephDeploymentHealth{
	ObjectMeta: LcmObjectMeta,
	Status: lcmv1alpha1.CephDeploymentHealthStatus{
		State:            lcmv1alpha1.HealthStateFailed,
		HealthReport:     CephBaseClusterReportNotOk,
		LastHealthCheck:  "time-now-fake",
		LastHealthUpdate: "time-now-fake",
		Issues: []string{
			"RECENT_MGR_MODULE_CRASH: 2 mgr modules have recently crashed",
			"cephcluster 'rook-ceph/cephcluster' object state is 'Failure'",
			"cephcluster 'rook-ceph/cephcluster' object status is not updated for last 5 minutes",
			"daemonset 'lcm-namespace/pelagia-disk-daemon' is not ready",
			"daemonset 'rook-ceph/csi-cephfsplugin' is not ready",
			"daemonset 'rook-ceph/csi-rbdplugin' is not ready",
			"failed to run 'ceph osd tree -f json' command to check replicas sizing",
			"no active mgr",
			"not all (2/3) mons are running",
			"not all osds are in",
			"not all osds are up",
		},
	},
}

var CephExternalClusterReportOk = &lcmv1alpha1.CephDeploymentHealthReport{
	RookOperator: RookOperatorStatusOk,
	RookCephObjects: &lcmv1alpha1.RookCephObjectsStatus{
		CephCluster: &CephClusterExternal.Status,
		ObjectStorage: &lcmv1alpha1.ObjectStorageStatus{
			CephObjectStores: map[string]*cephv1.ObjectStoreStatus{
				"rgw-store-external": CephObjectStoreExternalReady.Status,
			},
		},
	},
	CephDaemons: func() *lcmv1alpha1.CephDaemonsStatus {
		daemonsStatus := CephDaemonsStatusHealthy.DeepCopy()
		daemonsStatus.CephDaemons["rgw"] = CephDaemonsCephFsRgwHealthy["rgw"]
		return daemonsStatus
	}(),
	ClusterDetails: &lcmv1alpha1.ClusterDetails{
		UsageDetails: CephBaseUsageDetails,
		CephEvents:   CephEventsIdle,
		RgwInfo: &lcmv1alpha1.RgwInfo{
			PublicEndpoint: "https://127.0.0.1:8443",
		},
	},
}

var CephBaseClusterReportOk = &lcmv1alpha1.CephDeploymentHealthReport{
	RookOperator:    RookOperatorStatusOk,
	RookCephObjects: RookCephObjectsReportOnlyCephCluster,
	CephDaemons:     CephDaemonsStatusHealthy,
	ClusterDetails:  CephDetailsStatusNoIssues,
	OsdAnalysis:     OsdSpecAnalysisOk,
}

var CephBaseClusterReportNotOk = &lcmv1alpha1.CephDeploymentHealthReport{
	RookOperator: RookOperatorStatusOk,
	RookCephObjects: func() *lcmv1alpha1.RookCephObjectsStatus {
		status := RookCephObjectsReportOnlyCephCluster.DeepCopy()
		status.CephCluster = &ReefCephClusterHasHealthIssues.Status
		return status
	}(),
	CephDaemons:    CephDaemonsStatusUnhealthy,
	ClusterDetails: CephDetailsStatusNoIssues,
	OsdAnalysis: &lcmv1alpha1.OsdSpecAnalysisState{
		DiskDaemon: lcmv1alpha1.DaemonStatus{
			Status:   lcmv1alpha1.DaemonStateFailed,
			Messages: []string{"0/2 ready"},
			Issues:   []string{"daemonset 'lcm-namespace/pelagia-disk-daemon' is not ready"},
		},
	},
}

var CephMultisiteClusterReportOk = &lcmv1alpha1.CephDeploymentHealthReport{
	RookOperator:    RookOperatorStatusOk,
	RookCephObjects: RookCephObjectsReportReadyFull,
	CephDaemons: &lcmv1alpha1.CephDaemonsStatus{
		CephDaemons: map[string]lcmv1alpha1.DaemonStatus{
			"mon": CephDaemonsCephFsRgwHealthy["mon"],
			"mgr": CephDaemonsCephFsRgwHealthy["mgr"],
			"osd": CephDaemonsCephFsRgwHealthy["osd"],
			"mds": {
				Status: lcmv1alpha1.DaemonStateOk,
				Messages: []string{
					"mds active: 1/1 (cephfs 'cephfs-1')", "mds active: 1/1, standby-replay: 1/1 (cephfs 'cephfs-2')",
				},
			},
			"rgw": {
				Status:   lcmv1alpha1.DaemonStateOk,
				Messages: []string{"3 rgws running, daemons: [10223488 11556688 12065099]"},
			},
		},
		CephCSIPluginDaemons: CephCSIDaemonsReady,
	},
	ClusterDetails: &lcmv1alpha1.ClusterDetails{
		UsageDetails: CephExtraUsageDetails,
		CephEvents:   CephEventsIdle,
		RgwInfo: &lcmv1alpha1.RgwInfo{
			PublicEndpoint:   "https://rgw-store.example.com",
			MultisiteDetails: CephMultisiteStateOk,
		},
	},
	OsdAnalysis: OsdSpecAnalysisOk,
}

var RookOperatorStatusOk = lcmv1alpha1.DaemonStatus{
	Status: lcmv1alpha1.DaemonStateOk,
}

var RookOperatorStatusFailed = lcmv1alpha1.DaemonStatus{
	Status: lcmv1alpha1.DaemonStateFailed,
	Issues: []string{"failed to get 'rook-ceph-operator' deployment in 'rook-ceph' namespace"},
}

var RookCephObjectsReportOnlyCephCluster = &lcmv1alpha1.RookCephObjectsStatus{
	CephCluster: &ReefCephClusterReady.Status,
}

var RookCephObjectsReportReadyOnlyCephCluster = &lcmv1alpha1.RookCephObjectsStatus{
	CephCluster: &ReefCephClusterReady.Status,
	BlockStorage: &lcmv1alpha1.BlockStorageStatus{
		CephBlockPools: map[string]*cephv1.CephBlockPoolStatus{
			"pool1": CephBlockPoolListNotReady.Items[0].Status,
			"pool2": nil,
		},
	},
	CephClients: map[string]*cephv1.CephClientStatus{
		"client1": CephClientListNotReady.Items[0].Status,
		"client2": nil,
	},
	ObjectStorage: &lcmv1alpha1.ObjectStorageStatus{
		CephObjectStores: map[string]*cephv1.ObjectStoreStatus{
			"rgw-store":      CephObjectStoresMultisiteSyncDaemonPhaseNotReady.Items[0].Status,
			"rgw-store-sync": nil,
		},
		CephObjectStoreUsers: map[string]*cephv1.ObjectStoreUserStatus{
			"rgw-user-1": CephObjectStoreUserListNotReady.Items[0].Status,
			"rgw-user-2": nil,
		},
		ObjectBucketClaims: map[string]bktv1alpha1.ObjectBucketClaimStatus{
			"bucket-1": ObjectBucketClaimListNotReady.Items[0].Status,
		},
		CephObjectRealms: map[string]*cephv1.Status{
			"realm-1": CephObjectRealmListNotReady.Items[0].Status,
			"realm-2": nil,
		},
		CephObjectZoneGroups: map[string]*cephv1.Status{
			"zonegroup-1": CephObjectZoneGroupListNotReady.Items[0].Status,
			"zonegroup-2": nil,
		},
		CephObjectZones: map[string]*cephv1.Status{
			"zone-1": CephObjectZoneListNotReady.Items[0].Status,
			"zone-2": nil,
		},
	},
	SharedFilesystem: &lcmv1alpha1.SharedFilesystemStatus{
		CephFilesystems: map[string]*cephv1.CephFilesystemStatus{
			"cephfs-1": CephFilesystemListMultipleNotReady.Items[0].Status,
			"cephfs-2": nil,
		},
	},
}

var RookCephObjectsReportReadyFull = &lcmv1alpha1.RookCephObjectsStatus{
	CephCluster: &ReefCephClusterReady.Status,
	BlockStorage: &lcmv1alpha1.BlockStorageStatus{
		CephBlockPools: map[string]*cephv1.CephBlockPoolStatus{
			"pool1": CephBlockPoolListReady.Items[0].Status,
			"pool2": CephBlockPoolListReady.Items[1].Status,
		},
	},
	CephClients: map[string]*cephv1.CephClientStatus{
		"client1": CephClientListReady.Items[0].Status,
		"client2": CephClientListReady.Items[1].Status,
	},
	ObjectStorage: &lcmv1alpha1.ObjectStorageStatus{
		CephObjectStores: map[string]*cephv1.ObjectStoreStatus{
			"rgw-store":      CephObjectStoresMultisiteSyncDaemonPhaseReady.Items[0].Status,
			"rgw-store-sync": CephObjectStoresMultisiteSyncDaemonPhaseReady.Items[1].Status,
		},
		CephObjectStoreUsers: map[string]*cephv1.ObjectStoreUserStatus{
			"rgw-user-1": CephObjectStoreUserListReady.Items[0].Status,
			"rgw-user-2": CephObjectStoreUserListReady.Items[1].Status,
		},
		ObjectBucketClaims: map[string]bktv1alpha1.ObjectBucketClaimStatus{
			"bucket-1": ObjectBucketClaimListReady.Items[0].Status,
		},
		CephObjectRealms: map[string]*cephv1.Status{
			"realm-1": CephObjectRealmListReady.Items[0].Status,
		},
		CephObjectZoneGroups: map[string]*cephv1.Status{
			"zonegroup-1": CephObjectZoneGroupListReady.Items[0].Status,
		},
		CephObjectZones: map[string]*cephv1.Status{
			"zone-1": CephObjectZoneListReady.Items[0].Status,
		},
	},
	SharedFilesystem: &lcmv1alpha1.SharedFilesystemStatus{
		CephFilesystems: map[string]*cephv1.CephFilesystemStatus{
			"cephfs-1": CephFilesystemListMultipleReady.Items[0].Status,
			"cephfs-2": CephFilesystemListMultipleReady.Items[1].Status,
		},
	},
}

var CephDaemonsStatusHealthy = &lcmv1alpha1.CephDaemonsStatus{
	CephDaemons:          CephDaemonsBaseHealthy,
	CephCSIPluginDaemons: CephCSIDaemonsReady,
}

var CephDaemonsStatusUnhealthy = &lcmv1alpha1.CephDaemonsStatus{
	CephDaemons:          CephDaemonsBaseUnhealthy,
	CephCSIPluginDaemons: CephCSIDaemonsNotReady,
}

var CephDaemonsBaseHealthy = map[string]lcmv1alpha1.DaemonStatus{
	"mon": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3 mons, quorum [a b c]"},
	},
	"mgr": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"a is active mgr"},
	},
	"osd": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3 osds, 3 up, 3 in"},
	},
}

var CephDaemonsBaseUnhealthy = map[string]lcmv1alpha1.DaemonStatus{
	"mon": {
		Status:   lcmv1alpha1.DaemonStateFailed,
		Messages: []string{"2 mons, quorum [a b]"},
		Issues:   []string{"not all (2/3) mons are running"},
	},
	"mgr": {
		Status: lcmv1alpha1.DaemonStateFailed,
		Issues: []string{"no active mgr"},
	},
	"osd": {
		Status:   lcmv1alpha1.DaemonStateFailed,
		Messages: []string{"3 osds, 2 up, 2 in"},
		Issues:   []string{"not all osds are in", "not all osds are up"},
	},
}

var CephDaemonsCephFsRgwHealthy = map[string]lcmv1alpha1.DaemonStatus{
	"mon": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3 mons, quorum [a b c]"},
	},
	"mgr": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"a is active mgr"},
	},
	"osd": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3 osds, 3 up, 3 in"},
	},
	"mds": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"mds active: 1/1 (cephfs 'cephfs-1')"},
	},
	"rgw": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"2 rgws running, daemons: [11556688 12065099]"},
	},
}

var CephDaemonsCephFsRgwUnhealthy = map[string]lcmv1alpha1.DaemonStatus{
	"mon": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3 mons, quorum [a b c]"},
	},
	"mgr": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"a is active mgr"},
	},
	"osd": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3 osds, 3 up, 3 in"},
	},
	"mds": {
		Status: lcmv1alpha1.DaemonStateFailed,
		Issues: []string{
			"unexpected number (0/1) of mds active are running for CephFS 'cephfs-1'", "unexpected number (0/1) of mds standby are running",
		},
		Messages: []string{"mds active: 0/1 (cephfs 'cephfs-1')"},
	},
	"rgw": {
		Status:   lcmv1alpha1.DaemonStateFailed,
		Issues:   []string{"not all (0/2) rgws are running"},
		Messages: []string{"0 rgws running, daemons: []"},
	},
}

var CephCSIDaemonsReady = map[string]lcmv1alpha1.DaemonStatus{
	"csi-rbdplugin": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3/3 ready"},
	},
	"csi-cephfsplugin": {
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"3/3 ready"},
	},
}

var CephCSIDaemonsNotReady = map[string]lcmv1alpha1.DaemonStatus{
	"csi-rbdplugin": {
		Status:   lcmv1alpha1.DaemonStateFailed,
		Messages: []string{"1/3 ready"},
		Issues:   []string{"daemonset 'rook-ceph/csi-rbdplugin' is not ready"},
	},
	"csi-cephfsplugin": {
		Status:   lcmv1alpha1.DaemonStateFailed,
		Messages: []string{"1/3 ready"},
		Issues:   []string{"daemonset 'rook-ceph/csi-cephfsplugin' is not ready"},
	},
}

var CephDetailsStatusNoIssues = &lcmv1alpha1.ClusterDetails{
	UsageDetails: CephBaseUsageDetails,
	CephEvents:   CephEventsIdle,
}

var CephBaseUsageDetails = &lcmv1alpha1.UsageDetails{
	PoolsDetail: map[string]lcmv1alpha1.PoolUsageStats{
		"pool-hdd": {UsedBytes: "12288", UsedBytesPercentage: "0.000", TotalBytes: "104807096320", AvailableBytes: "104807084032"},
		".mgr":     {UsedBytes: "1388544", UsedBytesPercentage: "0.000", TotalBytes: "104807096320", AvailableBytes: "104805707776"},
	},
	ClassesDetail: map[string]lcmv1alpha1.ClassUsageStats{
		"hdd": {UsedBytes: "81630961664", TotalBytes: "509981204480", AvailableBytes: "428350242816"},
	},
}

var CephExtraUsageDetails = &lcmv1alpha1.UsageDetails{
	PoolsDetail: map[string]lcmv1alpha1.PoolUsageStats{
		"pool-hdd":                     {UsedBytes: "251719680", UsedBytesPercentage: "0.080", TotalBytes: "104710103040", AvailableBytes: "104458383360"},
		".mgr":                         {UsedBytes: "2777088", UsedBytesPercentage: "0.001", TotalBytes: "104710103040", AvailableBytes: "104707325952"},
		".rgw.root":                    {UsedBytes: "196608", UsedBytesPercentage: "0.000", TotalBytes: "104710103040", AvailableBytes: "104709906432"},
		"rgw-store.rgw.log":            {UsedBytes: "1990656", UsedBytesPercentage: "0.001", TotalBytes: "104710103040", AvailableBytes: "104708112384"},
		"rgw-store.rgw.buckets.non-ec": {UsedBytes: "0", UsedBytesPercentage: "0.000", TotalBytes: "104710103040", AvailableBytes: "104710103040"},
		"rgw-store.rgw.buckets.index":  {UsedBytes: "0", UsedBytesPercentage: "0.000", TotalBytes: "104710103040", AvailableBytes: "104710103040"},
		"rgw-store.rgw.otp":            {UsedBytes: "0", UsedBytesPercentage: "0.000", TotalBytes: "104710103040", AvailableBytes: "104710103040"},
		"rgw-store.rgw.control":        {UsedBytes: "0", UsedBytesPercentage: "0.000", TotalBytes: "104710103040", AvailableBytes: "104710103040"},
		"rgw-store.rgw.meta":           {UsedBytes: "49152", UsedBytesPercentage: "0.000", TotalBytes: "104710103040", AvailableBytes: "104710053888"},
		"rgw-store.rgw.buckets.data":   {UsedBytes: "0", UsedBytesPercentage: "0.000", TotalBytes: "209420206080", AvailableBytes: "209420206080"},
		"my-cephfs-metadata":           {UsedBytes: "114688", UsedBytesPercentage: "0.000", TotalBytes: "157065150464", AvailableBytes: "157065035776"},
		"my-cephfs-data-1":             {UsedBytes: "0", UsedBytesPercentage: "0.000", TotalBytes: "157065150464", AvailableBytes: "157065150464"},
		"my-cephfs-data-2":             {UsedBytes: "0", UsedBytesPercentage: "0.000", TotalBytes: "209420206080", AvailableBytes: "209420206080"},
	},
	ClassesDetail: map[string]lcmv1alpha1.ClassUsageStats{
		"hdd": {UsedBytes: "82097160192", TotalBytes: "509981204480", AvailableBytes: "427884044288"},
		"ssd": {UsedBytes: "77127680", TotalBytes: "53682896896", AvailableBytes: "53605769216"},
	},
}

var CephEventsIdle = &lcmv1alpha1.CephEvents{
	RebalanceDetails:    lcmv1alpha1.CephEventDetails{State: lcmv1alpha1.CephEventIdle},
	PgAutoscalerDetails: lcmv1alpha1.CephEventDetails{State: lcmv1alpha1.CephEventIdle},
}

var CephEventsProgressing = &lcmv1alpha1.CephEvents{
	RebalanceDetails: lcmv1alpha1.CephEventDetails{
		State:    lcmv1alpha1.CephEventProgressing,
		Progress: "almost done",
		Messages: []lcmv1alpha1.CephEventMessage{
			{
				Message:  "Rebalancing after osd.3 marked in (33s)",
				Progress: "0.948051929473877",
			},
		},
	},
	PgAutoscalerDetails: lcmv1alpha1.CephEventDetails{
		State:    lcmv1alpha1.CephEventProgressing,
		Progress: "more than a half done",
		Messages: []lcmv1alpha1.CephEventMessage{
			{
				Message:  "PG autoscaler increasing pool 9 PGs from 32 to 128 (0s)",
				Progress: "0.5294585938568447",
			},
		},
	},
}

var CephMultisiteStateFailed = &lcmv1alpha1.MultisiteState{
	MetadataSyncState: lcmv1alpha1.MultiSiteFailed,
	DataSyncState:     lcmv1alpha1.MultiSiteFailed,
	Messages:          []string{"failed to run 'radosgw-admin sync status --rgw-zonegroup=zonegroup1 --rgw-zone=zone1' command to check multisite status for zone 'zone1'"},
}

var CephMultisiteStateOk = &lcmv1alpha1.MultisiteState{
	MetadataSyncState: lcmv1alpha1.MultiSiteSyncing,
	DataSyncState:     lcmv1alpha1.MultiSiteSyncing,
	MasterZone:        true,
}

var OsdSpecAnalysisOk = &lcmv1alpha1.OsdSpecAnalysisState{
	DiskDaemon: lcmv1alpha1.DaemonStatus{
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"2/2 ready"},
	},
	CephClusterSpecGeneration: &ReefCephClusterReady.Generation,
	SpecAnalysis:              OsdStorageSpecAnalysisOk,
}

var OsdSpecAnalysisNotOk = &lcmv1alpha1.OsdSpecAnalysisState{
	DiskDaemon: lcmv1alpha1.DaemonStatus{
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{"2/2 ready"},
	},
	CephClusterSpecGeneration: &ReefCephClusterReady.Generation,
	SpecAnalysis:              OsdStorageSpecAnalysisFailed,
}

var OsdStorageSpecAnalysisOk = map[string]lcmv1alpha1.DaemonStatus{
	"node-1": {
		Status: lcmv1alpha1.DaemonStateOk,
	},
	"node-2": {
		Status: lcmv1alpha1.DaemonStateOk,
		Messages: []string{
			"found ceph block partition '/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca', belongs to osd '2' (osd fsid '61869d90-2c45-4f02-b7c3-96955f41e2ca'), placed on '/dev/vde' device, which seems to be stray, can be cleaned up",
			"found ceph block partition '/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19', belongs to osd '0' (osd fsid '06bf4d7c-9603-41a4-b250-284ecf3ecb2f'), placed on '/dev/vdc' device, which seems to be stray, can be cleaned up",
		},
	},
}

var OsdStorageSpecAnalysisFailed = map[string]lcmv1alpha1.DaemonStatus{
	"node-1": {
		Status: lcmv1alpha1.DaemonStateFailed,
		Issues: []string{"failed to run 'pelagia-disk-daemon --full-report --port 9999' command to get disk report from pelagia-disk-daemon"},
	},
	"node-2": {
		Status: lcmv1alpha1.DaemonStateFailed,
		Issues: []string{"failed to run 'pelagia-disk-daemon --full-report --port 9999' command to get disk report from pelagia-disk-daemon"},
	},
}
