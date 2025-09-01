package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="Phase"
// +kubebuilder:printcolumn:name="Additinal info",type=string,JSONPath=`.status.phaseInfo`,description="Extra phase Info"
// +kubebuilder:printcolumn:name="Approve",type=boolean,JSONPath=`.spec.approve`,description="Approve"
// +kubebuilder:resource:path=cephosdremovetasks,scope=Namespaced
// +kubebuilder:resource:shortName={osdlcm}
// +kubebuilder:subresource:status
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CephOsdRemoveTask stands for handling tasks for removing osds from cluster
type CephOsdRemoveTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// CephOsdRemoveTaskSpec contains main remove task options
	// +optional
	Spec *CephOsdRemoveTaskSpec `json:"spec,omitempty"`
	// CephOsdRemoveTaskStatus contains remove info for task
	// +optional
	Status *CephOsdRemoveTaskStatus `json:"status,omitempty"`
}

// CephOsdRemoveTaskSpec contains approval flag,
// map of nodes with osd to-remove list and flag to mark failed
// request as completed to keep in history
type CephOsdRemoveTaskSpec struct {
	// Nodes is a map of nodes, which contains specification how osds
	// should be removed: by devices or osd ids
	// +optional
	Nodes map[string]NodeCleanUpSpec `json:"nodes,omitempty"`
	// Approve is a ceph team emergency break to ask operator to
	// think twice before removing OSD. Could be only manually be
	// enabled by user.
	// +optional
	Approve bool `json:"approve,omitempty"`
	// Resolved allows to keep task in history when it is failed and
	// do not block any further operations.
	// +optional
	Resolved bool `json:"resolved,omitempty"`
}

// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:MaxProperties:=1

// NodeCleanUpSpec describes how should be OSD cleaned up on particular node
// Can be set only one field at time.
type NodeCleanUpSpec struct {
	// CompleteCleanUp is a flag for total node cleanup and drop from crush map
	// Node will be cleaned up with all its osd and devices if possible
	// +optional
	CompleteCleanup bool `json:"completeCleanup,omitempty"`
	// DropFromCrush is the same to CompleteCleanup, but without devices cleanup on host
	// May be useful when host is going to be reprovisioned and
	// no need to spent time for devices clean up
	// +optional
	DropFromCrush bool `json:"dropFromCrush,omitempty"`
	// CleanupStrayPartitions is a flag for cleaning disks with osd lvm partitions
	// which are not belong to current cluster, for example, when disk was not cleaned
	// after previous setup
	// +optional
	CleanupStrayPartitions bool `json:"cleanupStrayPartitions,omitempty"`
	// CleanupByDevice describes devices or it pathes and osd on it to cleanup
	// +optional
	// +kubebuilder:validation:MinItems:=1
	CleanupByDevice []DeviceCleanupSpec `json:"cleanupByDevice,omitempty"`
	// CleanupByOsdID is a list of osd's id, placed on node to cleanup
	// +optional
	// +kubebuilder:validation:MinItems:=1
	CleanupByOsd []OsdCleanupSpec `json:"cleanupByOsd,omitempty"`
}

// DeviceCleanupSpec is a spec describing dev names or pathes to cleanup
// and whether to cleanup it or not
type DeviceCleanupSpec struct {
	// Device represents either physical dev names on a node, used for osd, e.g. 'sdb', '/dev/nvme1e0'.
	// Either full dev path (by-path or by-id) on a node, where osd lives, e.g. '/dev/disk/by-path/...' or '/dev/disk/by-id/...'
	// +kubebuilder:validation:Pattern:=`^((\/dev\/)?[\w]+$)|(\/dev\/disk\/by-(path|id)\/.+)`
	Device string `json:"device"`
	// SkipDeviceCleanup is a flag, whether to skip device/osd partitions cleanup
	// related to all osd found on specified device, including partitions on other related devices.
	// +optional
	SkipDeviceCleanup bool `json:"skipDeviceCleanup,omitempty"`
}

type OsdCleanupSpec struct {
	// Osd id to remove from cluster
	ID int `json:"id"`
	// SkipDeviceCleanup is a flag, whether to skip device/osd partitions cleanup
	// related to specified osd id
	// +optional
	SkipDeviceCleanup bool `json:"skipDeviceCleanup,omitempty"`
}

// TaskPhase is a enum for all supported
// handle task phases
type TaskPhase string

// Phases are moving in next order:
// Pending -> Validating -> ValidationFailed
// Pending -> Validating -> ApproveWaiting -> Processing -> Failed
// Pending -> Validating -> ApproveWaiting -> Processing -> Complete
const (
	TaskPhaseApproveWaiting        TaskPhase = "ApproveWaiting"
	TaskPhaseAborted               TaskPhase = "Aborted"
	TaskPhaseCompleted             TaskPhase = "Completed"
	TaskPhaseCompletedWithWarnings TaskPhase = "CompletedWithWarnings"
	TaskPhaseFailed                TaskPhase = "Failed"
	TaskPhasePending               TaskPhase = "Pending"
	TaskPhaseWaitingOperator       TaskPhase = "WaitingOperator"
	TaskPhaseProcessing            TaskPhase = "Processing"
	TaskPhaseValidating            TaskPhase = "Validating"
	TaskPhaseValidationFailed      TaskPhase = "ValidationFailed"
)

// CephOsdRemoveTaskStatus contains status of removing osds process
// and possible info/error messages found on during process
type CephOsdRemoveTaskStatus struct {
	// Phase is a current task phase
	Phase TaskPhase `json:"phase"`
	// Additional state info
	// +nullable
	PhaseInfo string `json:"phaseInfo,omitempty"`
	// RemoveInfo contains map, describing on what is going to be removed
	// in next view: node -> osd ID -> associated devices info,
	// issues found during validation/processing phases
	// and warnings which user should pay attention to
	// +optional
	RemoveInfo *TaskRemoveInfo `json:"removeInfo,omitempty"`
	// Messages is a list of info messages describing what's a reason
	// of moving task to next phase
	// +optional
	Messages []string `json:"messages,omitempty"`
	// Conditions is a history list of changing task itself
	// +optional
	Conditions []CephOsdRemoveTaskCondition `json:"conditions"`
}

type TaskRemoveInfo struct {
	// CleanupMap is a map of cleanup from host-osdId to device
	// based on this map user will decide whether approve current task or not
	// after that it will contain all remove statuses and errors during remove
	// +optional
	CleanupMap map[string]HostMapping `json:"cleanupMap"`
	// Issues found during validation/processing phases, describing occured problem
	// +optional
	Issues []string `json:"issues,omitempty"`
	// Warnings found during validation/processing phases, user attention required
	// +optional
	Warnings []string `json:"warnings,omitempty"`
}

type HostMapping struct {
	// CompleteCleanUp is a flag whether make complete host cleanup from crush map
	// +optional
	CompleteCleanup bool `json:"completeCleanup,omitempty"`
	// DropFromCrush is a flag whether make complete host cleanup from
	// crush map, but do not cleanup used devices
	// +optional
	DropFromCrush bool `json:"dropFromCrush,omitempty"`
	// OsdMapping represents a mapping from osdID -> devices, also contains
	// osd remove statuses such as osd remove itself, deployment remove,
	// device clean up job
	// +optional
	OsdMapping map[string]OsdMapping `json:"osdMapping"`
	// NodeIsDown indicates host availability
	// +optional
	NodeIsDown bool `json:"nodeIsDown,omitempty"`
	// VolumesInfoMissed indicates volume info unavailability for host
	// +optional
	VolumesInfoMissed bool `json:"volumeInfoMissed,omitempty"`
	// HostRemoveStatus represents host remove status, if node marked for complete clean up
	// +optional
	HostRemoveStatus *RemoveStatus `json:"hostRemoveStatus,omitempty"`
}

type OsdMapping struct {
	// osd UUID in cluster
	// +nullable
	UUID string `json:"uuid,omitempty"`
	// ceph cluster FSID
	// +nullable
	ClusterFSID string `json:"clusterFSID,omitempty"`
	// host directory rook path
	// +nullable
	HostDirectory string `json:"hostDirectory,omitempty"`
	// marker that osd is present in crush map
	// +optional
	InCrushMap bool `json:"inCrushMap"`
	// DeviceMapping is a mapping device -> device info, with short device info
	// such as path, class, partition, etc
	// +optional
	DeviceMapping map[string]DeviceInfo `json:"deviceMapping,omitempty"`
	// Whether to skip devices cleanup for current osd
	// +optional
	SkipDeviceCleanupJob bool `json:"skipDevicesCleanup,omitempty"`
	// RemoveStatus describing current phase and errors if happened
	// for osd, deployment or device clean up
	// +optional
	RemoveStatus *RemoveResult `json:"removeStatus,omitempty"`
}

// DeviceInfo represents short device info which provide all
// needed info for clean up procedure
type DeviceInfo struct {
	// Whether device is rotational (hdd or ssd/nvme)
	// +optional
	Rotational bool `json:"rotational,omitempty"`
	// Path is a full device path by-path to remove
	// +nullable
	Path string `json:"devicePath,omitempty"`
	// ID is a device id
	// +nullable
	ID string `json:"deviceID,omitempty"`
	// Partition used for removing osd on device
	// +nullable
	Partition string `json:"osdPartition,omitempty"`
	// Type is a osd partition type, e.g. db or block
	// +nullable
	Type string `json:"osdPartitionType,omitempty"`
	// ZapDevice is a flag whether to cleanup disk at all or only partitions on it
	// +optional
	Zap bool `json:"deviceCleanup,omitempty"`
	// Alive is a marker whether device lost or alive
	// +optional
	Alive bool `json:"deviceAlive,omitempty"`
}

// RemovePhase is a enum for handling remove during processing phase
type RemovePhase string

const (
	RemovePending          RemovePhase = "Pending"
	RemoveWaitingRebalance RemovePhase = "Rebalancing"
	RemoveInProgress       RemovePhase = "Removing"
	RemoveStray            RemovePhase = "RemovingStray"
	RemoveCompleted        RemovePhase = "Completed"
	RemoveFinished         RemovePhase = "Removed"
	RemoveFailed           RemovePhase = "Failed"
	RemoveSkipped          RemovePhase = "Skipped"
)

// RemoveResult keeps all osd remove related statuses in one place
type RemoveResult struct {
	// OsdRemoveStatus represents Ceph OSD remove status itself
	// +optional
	OsdRemoveStatus *RemoveStatus `json:"osdRemoveStatus,omitempty"`
	// DeployRemoveStatus represents osd related deployment remove status
	// +optional
	DeployRemoveStatus *RemoveStatus `json:"deploymentRemoveStatus,omitempty"`
	// DeviceCleanUpJob represents osd-device related clean up job status
	// +optional
	DeviceCleanUpJob *RemoveStatus `json:"deviceCleanUpJob,omitempty"`
}

// RemoveStatus handling status description
type RemoveStatus struct {
	// Status is a current remove status
	Status RemovePhase `json:"status"`
	// Name is an object name for handling, optional
	// +nullable
	Name string `json:"name,omitempty"`
	// Error faced during handling
	// +nullable
	Error string `json:"error,omitempty"`
	// Start time for remove action
	// +nullable
	StartedAt string `json:"startedAt,omitempty"`
	// Finish time for remove action
	// +nullable
	FinishedAt string `json:"finishedAt,omitempty"`
}

// CephOsdRemoveTaskCondition contains history of changes/updates for task
type CephOsdRemoveTaskCondition struct {
	// Timestamp is a timestamp when this condition appeared
	Timestamp string `json:"timestamp"`
	// Phase is a current task handling phase
	Phase TaskPhase `json:"phase"`
	// Nodes is a mapping of nodes within their Ceph OSDs / devices to clean up
	// +optional
	Nodes map[string]NodeCleanUpSpec `json:"nodes,omitempty"`
	// CephClusterSpecVersion is a version of cephcluster used for that
	// condition in format <generation>-<resourceVersion>
	// +optional
	CephClusterSpecVersion *CephClusterSpecVersion `json:"cephClusterVersion,omitempty"`
}

type CephClusterSpecVersion struct {
	// ResourceVersion is a CephCluster resource version
	// +nullable
	ResourceVersion string `json:"cephClusterResourceVersion"`
	// Generation is a CephCluster generation ID
	// +optional
	Generation int64 `json:"cephClusterGeneration"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CephOsdRemoveTaskList contains a list of CephOsdRemoveTask objects
type CephOsdRemoveTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains a list of CephOsdRemoveTask objects
	Items []CephOsdRemoveTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CephOsdRemoveTask{}, &CephOsdRemoveTaskList{})
}
