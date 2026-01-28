package v1alpha1

import (
	bktv1alpha1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:resource:path=cephdeploymenthealths,scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Cluster health state"
// +kubebuilder:printcolumn:name="Last check",type=string,JSONPath=`.status.lastHealthCheck`,description="Last state check"
// +kubebuilder:printcolumn:name="Last update",type=string,JSONPath=`.status.lastHealthUpdate`,description="Last state update"
// +kubebuilder:resource:shortName={cdh}
// +kubebuilder:subresource:status
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CephDeploymentHealth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status represents cluster health state
	// +optional
	Status CephDeploymentHealthStatus `json:"status,omitempty"`
}

type CephDeploymentHealthState string
type DaemonState string

const (
	HealthStateOk     CephDeploymentHealthState = "Ok"
	HealthStateFailed CephDeploymentHealthState = "Failed"

	DaemonStateOk      DaemonState = "ok"
	DaemonStateFailed  DaemonState = "failed"
	DaemonStateSkipped DaemonState = "skipped"
)

type CephDeploymentHealthStatus struct {
	// State represents the state for overall status
	State CephDeploymentHealthState `json:"state"`
	// FullClusterStatus represents overall Ceph cluster status info
	// +optional
	HealthReport *CephDeploymentHealthReport `json:"healthReport,omitempty"`
	// LastCheck is a last time when cluster was verified
	// +nullable
	LastHealthCheck string `json:"lastHealthCheck,omitempty"`
	// LastUpdate is a last time when CephDeploymentHealthStatus was updated
	// +nullable
	LastHealthUpdate string `json:"lastHealthUpdate,omitempty"`
	// Messages is a list with any possible error/warning messages
	// +optional
	Issues []string `json:"issues,omitempty"`
}

type CephDeploymentHealthReport struct {
	// RookOperator contains Rook operator status
	RookOperator DaemonStatus `json:"rookOperator"`
	// CephDaemons contains status of ceph daemons
	// +optional
	CephDaemons *CephDaemonsStatus `json:"cephDaemons,omitempty"`
	// Status of rook ceph cluster objects
	// +optional
	RookCephObjects *RookCephObjectsStatus `json:"rookCephObjects,omitempty"`
	// ClusterDetails contains additional Ceph cluster information, such as disk usage, device class usage
	// +optional
	ClusterDetails *ClusterDetails `json:"clusterDetails,omitempty"`
	// Osd spec analyse based on info from disk daemons for osd nodes
	// +optional
	OsdAnalysis *OsdSpecAnalysisState `json:"osdAnalysis,omitempty"`
}

type CephDaemonsStatus struct {
	// CephDaemonsStatus contains Ceph daemon's overall information.
	// For now supported statuses for next Ceph daemons: mon, mgr, osd, rgw, mds.
	// +optional
	CephDaemons map[string]DaemonStatus `json:"cephDaemons,omitempty"`
	// CephCSIDaemons contains Ceph CSI related daemons status
	// Supported statuses for CSI operator, CephFS, RBD node plugins and controllers.
	// +optional
	CephCSIDaemons map[string]DaemonStatus `json:"cephCSIDaemons,omitempty"`
	// CephCSIPluginDaemons is deprecated in favor cephCSIDaemons section
	CephCSIPluginDaemons map[string]DaemonStatus `json:"cephCSIPluginDaemons,omitempty"`
}

type DaemonStatus struct {
	// Status contains short state of current daemon
	// +optional
	Status DaemonState `json:"status,omitempty"`
	// Issues represents found Ceph daemon issues, otherwise it is empty
	// +optional
	Issues []string `json:"issues,omitempty"`
	// Messages contains human-readable additional information about current daemon state
	// +optional
	Messages []string `json:"info,omitempty"`
}

type RookCephObjectsStatus struct {
	// CephCluster contains Ceph cluster status information
	// +optional
	CephCluster *cephv1.ClusterStatus `json:"cephCluster,omitempty"`
	// BlockStorage contains status of block-storage related objects status information
	// +optional
	BlockStorage *BlockStorageStatus `json:"blockStorage,omitempty"`
	// CephClients represents a key-value mapping of Ceph Client's name and it's status
	// +optional
	CephClients map[string]*cephv1.CephClientStatus `json:"cephClients,omitempty"`
	// ObjectStorage contains Ceph Object storage's related object status information
	// +optional
	ObjectStorage *ObjectStorageStatus `json:"objectStorage,omitempty"`
	// SharedFilesystem contains Ceph Filesystem's related objects status information
	// +optional
	SharedFilesystem *SharedFilesystemStatus `json:"sharedFilesystem,omitempty"`
}

type BlockStorageStatus struct {
	// CephBlockPools represents a key-value mapping of Ceph Pool's name and it's status
	// +optional
	CephBlockPools map[string]*cephv1.CephBlockPoolStatus `json:"cephBlockPools,omitempty"`
}

type ObjectStorageStatus struct {
	// ObjectStoreStatus represents a key-value mapping of Ceph Object Store's name and it's status
	// +optional
	CephObjectStores map[string]*cephv1.ObjectStoreStatus `json:"cephObjectStore,omitempty"`
	// CephObjectStoreUsers represents a key-value mapping of Ceph object storage user's name and it's status
	// +optional
	CephObjectStoreUsers map[string]*cephv1.ObjectStoreUserStatus `json:"cephObjectStoreUsers,omitempty"`
	// ObjectBucketClaims represents a key-value mapping of Ceph object storage bucket's name and it's status
	// +optional
	ObjectBucketClaims map[string]bktv1alpha1.ObjectBucketClaimStatus `json:"objectBucketClaims,omitempty"`
	// CephObjectRealm represents a key-value mapping of Ceph object storage gateway realm's name and it's status
	// +optional
	CephObjectRealms map[string]*cephv1.Status `json:"cephObjectRealms,omitempty"`
	// CephObjectZoneGroups represents a key-value mapping of Ceph object storage gateway zone group's name and it's status
	// +optional
	CephObjectZoneGroups map[string]*cephv1.Status `json:"cephObjectZoneGroups,omitempty"`
	// CephObjectZones represents a key-value mapping of Ceph object storage gateway zone's name and it's status
	// +optional
	CephObjectZones map[string]*cephv1.Status `json:"cephObjectZones,omitempty"`
}

type SharedFilesystemStatus struct {
	// CephFilesystems represents a key-value mapping of CephFilesystem's name and it's status
	// +optional
	CephFilesystems map[string]*cephv1.CephFilesystemStatus `json:"cephFilesystems,omitempty"`
}

type ClusterDetails struct {
	// UsageDetails contains verbose info about usage/capacity cluster per class/pools
	// +optional
	UsageDetails *UsageDetails `json:"usageDetails,omitempty"`
	// CephEvents contains info about current ceph events happen in Ceph cluster
	// if progress events module is enabled
	// +optional
	CephEvents *CephEvents `json:"cephEvents,omitempty"`
	// RgwInfo represents additional Ceph Multiste Object storage info
	// +optional
	RgwInfo *RgwInfo `json:"rgwInfo,omitempty"`
}

type UsageDetails struct {
	// ClassesDetail represents info based on device classes usage
	// +optional
	ClassesDetail map[string]ClassUsageStats `json:"deviceClasses,omitempty"`
	// PoolsDetail represents info based on pools usage
	// +optional
	PoolsDetail map[string]PoolUsageStats `json:"pools,omitempty"`
}

type ClassUsageStats struct {
	// UsedBytes overall used bytes for device class
	// +optional
	UsedBytes string `json:"usedBytes,omitempty"`
	// AvailableBytes available not used bytes for device class
	// +optional
	AvailableBytes string `json:"availableBytes,omitempty"`
	// TotalBytes total capacity in bytes for device class
	// +optional
	TotalBytes string `json:"totalBytes,omitempty"`
}

type PoolUsageStats struct {
	// UsedBytes overall used bytes in pool
	// +nullable
	UsedBytes string `json:"usedBytes,omitempty"`
	// UsedBytesPercentage percent of used bytes in pool
	// +nullable
	UsedBytesPercentage string `json:"usedBytesPercentage,omitempty"`
	// AvailableBytes available not used bytes for pool
	// +nullable
	AvailableBytes string `json:"availableBytes,omitempty"`
	// TotalBytes total capacity in bytes for pool
	// +nullable
	TotalBytes string `json:"totalBytes,omitempty"`
}

const (
	CephEventIdle        CephEventState = "Idle"
	CephEventProgressing CephEventState = "Progressing"
)

type CephEventState string

type CephEvents struct {
	// RebalanceDetails contains info about current rebalancing processes happen in Ceph cluster
	// +optional
	RebalanceDetails CephEventDetails `json:"rebalanceDetails,omitempty"`
	// PgAutoscalerDetails contains info about current pg autoscaler events happen in Ceph cluster
	// +optional
	PgAutoscalerDetails CephEventDetails `json:"PgAutoscalerDetails,omitempty"`
}

type CephEventDetails struct {
	State    CephEventState     `json:"state,omitempty"`
	Messages []CephEventMessage `json:"messages,omitempty"`
	Progress string             `json:"progress,omitempty"`
}

type CephEventMessage struct {
	Message  string `json:"message,omitempty"`
	Progress string `json:"progress,omitempty"`
}

type RgwInfo struct {
	// PublicEndpoint represents external endpoint to access object storage
	// +nullable
	PublicEndpoint string `json:"publicEndpoint,omitempty"`
	// MultisiteDetails represents overall multisite state info
	// +optional
	MultisiteDetails *MultisiteState `json:"multisiteDetails,omitempty"`
}

type MultisiteState struct {
	// Current Multisite metadata sync state
	MetadataSyncState MultiSiteState `json:"metadataSyncState"`
	// Current Multisite data sync state
	DataSyncState MultiSiteState `json:"dataSyncState"`
	// MasterZone whether current zone is master
	// +optional
	MasterZone bool `json:"masterZone,omitempty"`
	// Additional messages about current state
	// +optional
	Messages []string `json:"messages,omitempty"`
}

type MultiSiteState string

const (
	MultiSiteSyncing   MultiSiteState = "syncing"
	MultiSiteOutOfSync MultiSiteState = "out of sync"
	MultiSiteFailed    MultiSiteState = "failed"
)

type OsdSpecAnalysisState struct {
	// Disk-daemon status
	DiskDaemon DaemonStatus `json:"diskDaemon"`
	// CephClusterSpecGeneration is a last validated rook cephcluster spec
	// +optional
	CephClusterSpecGeneration *int64 `json:"cephClusterSpecGeneration,omitempty"`
	// SpecAnalysis is a spec analysis status for nodes in cephcluster storage spec
	// +optional
	SpecAnalysis map[string]DaemonStatus `json:"specAnalysis,omitempty"`
}

type OsdDetails struct {
	DeviceName       string `json:"deviceName,omitempty"`
	DeviceByID       string `json:"deviceByID,omitempty"`
	DeviceByPath     string `json:"deviceByPath,omitempty"`
	DeviceClass      string `json:"deviceClass,omitempty"`
	BlockPartition   string `json:"blockPartition,omitempty"`
	MetaDeviceName   string `json:"metadataDeviceName,omitempty"`
	MetaDeviceByID   string `json:"metadataDeviceByID,omitempty"`
	MetaDeviceByPath string `json:"metadataDeviceByPath,omitempty"`
	MetaDeviceClass  string `json:"metadataDeviceClass,omitempty"`
	MetaPartition    string `json:"metaPartition,omitempty"`
	UUID             string `json:"osdUUID,omitempty"`
	Up               bool   `json:"up,omitempty"`
	In               bool   `json:"in,omitempty"`
}

type DeviceMapping map[string]OsdDetails

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CephDeploymentHealthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CephDeploymentHealth `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CephDeploymentHealthList{}, &CephDeploymentHealth{})
}
