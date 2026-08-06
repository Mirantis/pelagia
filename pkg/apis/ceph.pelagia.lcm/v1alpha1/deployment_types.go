package v1alpha1

import (
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Validation",type=string,JSONPath=`.status.validation.result`,description="Validation status"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="Deployment phase"
// +kubebuilder:printcolumn:name="Last run",type=string,JSONPath=`.status.lastRun`,description="Last reconcile run"
// +kubebuilder:printcolumn:name="Cluster version",type=string,JSONPath=`.status.clusterVersion`,description="Current Ceph cluster version"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="Cluster status message"
// +kubebuilder:resource:path=cephdeployments,scope=Namespaced
// +kubebuilder:resource:shortName={cephdpl}
// +kubebuilder:subresource:status
// +genclient

// CephDeployment is the Schema for the cephdeployments API which contains
// a valid Ceph configuration which is handled by Pelagia controller and
// produce all related objects and daemons in Rook (K8S based Ceph)
type CephDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration of resulting Ceph Cluster
	// and all corresponding resources
	Spec CephDeploymentSpec `json:"spec"`
	// Status represents current status of handling Ceph Cluster configuration
	// +optional
	Status CephDeploymentStatus `json:"status,omitempty"`
}

// CephDeploymentSpec defines the desired configuration of resulting Ceph Cluster
// and all corresponding resources
type CephDeploymentSpec struct {
	// Cluster stands for main Ceph cluster configuration
	// Required to be specified.
	Cluster *CephCluster `json:"cluster,omitempty"`
	// BlockStorage stands for configuration Ceph block storage,
	// such as rbd pool and rbd mirroring
	// +optional
	BlockStorage *CephBlockStorage `json:"blockStorage,omitempty"`
	// Clients is a list of Ceph Clients used for Ceph Cluster connection by
	// consumer services
	// +optional
	Clients []CephClient `json:"clients,omitempty"`
	// ExtraOpts contains some extra options for managing Ceph cluster, like devices labels
	// +optional
	ExtraOpts *CephDeploymentExtraOpts `json:"extraOpts,omitempty"`
	// Nodes contains full cluster nodes configuration to use as Ceph Nodes
	Nodes []CephDeploymentNode `json:"nodes"`
	// ObjectStorage contains full RadosGW Object Storage configurations: RGW itself
	// and RGW multisite feature
	// +optional
	ObjectStorage *CephObjectStorage `json:"objectStorage,omitempty"`
	// RBDMirror allows to configure RBD mirroring between two Ceph Clusters
	// +optional
	RBDMirror *CephRBDMirrorSpec `json:"rbdMirror,omitempty"`
	// RookConfig is a key-value mapping which contains ceph config keys with a specified values
	// +optional
	RookConfig map[string]string `json:"rookConfig,omitempty"`
	// SharedFilesystem enables such system as CephFS
	// +optional
	SharedFilesystem *CephSharedFilesystem `json:"sharedFilesystem,omitempty"`
	// CSI provides an ability to specify CephCSI Drivers and OperatorConfig objects
	// +optional
	CSIResources *CephCSI `json:"csi,omitempty"`

	// Deprecated parameter, objectStorage.gatewayHTTPRoutes should be used instead.
	// Ingress became deprecated and going to be replaced by Gateway API, for more information
	// follow https://gateway-api.sigs.k8s.io/guides/getting-started/migrating-from-ingress/
	// +optional
	IngressConfig *CephDeploymentIngressConfig `json:"ingressConfig,omitempty"`
}

// CephCluster represents cluster specification
// Follow https://rook.io/docs/rook/v1.19/CRDs/Cluster/ceph-cluster-crd/
// for available options
type CephCluster struct {
	runtime.RawExtension `json:",inline"`
}

type CephBlockStorage struct {
	// Pools is a list of Ceph RBD Pools configurations
	// +optional
	Pools []CephPool `json:"pools,omitempty"`
}

// CephPool stands for specified Ceph RBD Pool configuration
type CephPool struct {
	// Name represents Ceph RBD pool name
	Name string `json:"name"`
	// UseAsFullName uses Name as a resulting pool name instead of "<Name>-<DeviceClass>"
	// +optional
	UseAsFullName bool `json:"useAsFullName,omitempty"`
	// Role represents pool role. The following values are reserved for
	// MOS managed clusters: vms, images, backup, volumes
	// +nullable
	Role string `json:"role,omitempty"`
	// PreserveOnDelete prevents related CephBlockPool object removal
	// +optional
	PreserveOnDelete bool `json:"preserveOnDelete,omitempty"`
	// StorageClassOpts represents options to set on related storage class
	// +optional
	StorageClassOpts CephStorageClassSpec `json:"storageClassOpts,omitempty"`
	// PoolSpec represents pool specification
	// Follow https://rook.io/docs/rook/v1.19/CRDs/Block-Storage/ceph-block-pool-crd
	// for available options
	PoolSpec runtime.RawExtension `json:"spec"`
}

// CephClient represents client specification
// Follow https://rook.io/docs/rook/v1.19/CRDs/ceph-client-crd/
// for available options
type CephClient struct {
	runtime.RawExtension `json:",inline"`
}

type LabeledDevices map[string]string

// CephDeploymentExtraOpts contains extra options, used for cluster configuration and management
type CephDeploymentExtraOpts struct {
	// Mark some device by-id, by-path or name with label
	// +optional
	DeviceLabels map[string]LabeledDevices `json:"deviceLabels,omitempty"`
	// Enable progress events module. Disabled by default to due to CPU overhead
	// +optional
	EnableProgressEvents bool `json:"enableProgressEvents,omitempty"`
	// PreventClusterDestroy option is used to avoid occasional cluster remove.
	// Option should be dropped in case of real cluster remove.
	// +optional
	PreventClusterDestroy bool `json:"preventClusterDestroy,omitempty"`
	// OsdRestartReason option is used for restarting ALL osds on config changes,
	// which are requires daemon restart.
	// Should contain description why it is required.
	// +nullable
	OsdRestartReason string `json:"osdRestartReason,omitempty"`
	// DisableOsKeys disables automatic generating of openstack-ceph-keys secret.
	// Valuable only for MOS managed clusters
	// +optional
	DisableOsKeys bool `json:"disableOsSharedKeys,omitempty"`
}

// CephDeploymentNode contains specific node configuration to use it in Ceph Cluster
type CephDeploymentNode struct {
	cephv1.Node `json:",inline"`
	// Roles is a list of control daemons to spawn on the defined node: Ceph Monitor,
	// Ceph Manager and/or Ceph RadosGW daemons. Possible values are: mon, mgr, rgw
	Roles []string `json:"roles"`
	// Crush represents ceph crush topology rules to apply on
	// the defined node
	Crush map[string]string `json:"crush,omitempty"`
	// NodeGroup is a list of kubernetes node names
	// which allows to specify defined spec to a group of nodes
	// instead of one node defined with Name parameter. Name should be
	// interpreted as a node group name instead of node name if specified
	// +optional
	NodeGroup []string `json:"nodeGroup,omitempty"`
	// NodesByLabel is a valid kubernetes label selector expression
	// which allows to specify defined spec to a group of selected nodes
	// instead of one node defined with Name parameter. Name should be
	// interpreted as a node group name instead of node name if specified
	// +nullable
	NodesByLabel string `json:"nodesByLabel,omitempty"`
	// MonitorIP represents custom static endpoint for monitor daemon on a node.
	// Updates have no effect on that parameter, could be used only on monitor create
	// +nullable
	MonitorIP string `json:"monitorIP,omitempty"`
}

// CephObjectStorage contains full RadosGW Object Storage configurations:
// RGW itself and RGW multisite feature
type CephObjectStorage struct {
	// Rgws is a list of Ceph object stores, representing Ceph RadosGW
	// with its configuration
	// +optional
	Rgws []CephObjectStore `json:"objectStores,omitempty"`
	// Users is a list of user to create for object storage with radosgw-admin
	// +optional
	Users []CephObjectStoreUser `json:"users,omitempty"`
	// GatewayHTTPRoutes stands for adding Gateway API HTTPRoutes for Ceph RGW
	// public access
	// +optional
	GatewayHTTPRoutes []CephDeploymentHTTPRoute `json:"gatewayHTTPRoutes,omitempty"`
	// Realms is a list of Ceph Object storage multisite realms.
	// Currently is possible to specify only 1 realm.
	// +kubebuilder:validation:MaxItems:=1
	// +optional
	Realms []CephObjectRealm `json:"realms,omitempty"`
	// Zonegroups is a list of Ceph Object storage multisite zonegroups.
	// Currently is possible to specify only 1 zonegroup.
	// +kubebuilder:validation:MaxItems:=1
	// +optional
	Zonegroups []CephObjectZonegroup `json:"zonegroups,omitempty"`
	// Zones is a list of Ceph Object storage multisite zones.
	// Currently is possible to specify only 1 zone.
	// +kubebuilder:validation:MaxItems:=1
	// +optional
	Zones []CephObjectZone `json:"zones,omitempty"`
}

// CephObjectStore stands for configuration of object store.
type CephObjectStore struct {
	Name string `json:"name"`
	// Deprecated option, since Ingress is depreated in favor of Gateway API.
	// ObjectStore has ingress frontend.
	// +optional
	ServedByIngress bool `json:"servedByIngress,omitempty"`
	// ObjectStore will be used for Openstack.
	// +optional
	UsedForOpenstack bool `json:"usedForOpenstack,omitempty"`
	// AuxiliaryService stands for marking object store as non-client serving.
	// Storage class and external service will not be created.
	// +optional
	AuxiliaryService bool `json:"auxiliaryService,omitempty"`
	// Spec represents CephObjectStore configuration
	// Follow https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-crd/
	// for available options
	Spec runtime.RawExtension `json:"spec"`
}

// CephObjectStoreUser stands for configuration of object store users.
type CephObjectStoreUser struct {
	Name string `json:"name"`
	// Spec represents CephObjectStoreUser configuration
	// https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-user-crd/
	// for available options
	Spec runtime.RawExtension `json:"spec"`
}

// CephDeploymentHTTPRoute represents Gateway API HTTPRoute specification
type CephDeploymentHTTPRoute struct {
	// Name of httproute
	Name string `json:"name"`
	// Name of related object store object, which will be routed by httproute
	ObjectStoreName string `json:"objectStoreName"`
	// Spec represents HTTPRoute specification
	// Follow https://gateway-api.sigs.k8s.io/api-types/httproute/
	// for details
	Spec runtime.RawExtension `json:"spec"`
}

// CephObjectRealm stands for object store multisite realm creation and configuration.
type CephObjectRealm struct {
	// Name of realm
	Name string `json:"name"`
	// Spec stands for realm configuration.
	// see https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-realm-crd/
	// for available options
	Spec runtime.RawExtension `json:"spec,omitempty"`
}

// CephObjectZonegroup stands for object store multisite zonegroup creation and configuration.
type CephObjectZonegroup struct {
	// Name of zonegroup
	Name string `json:"name"`
	// Spec stands for zonegroup configuration.
	// https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-zonegroup-crd/
	// for available options
	Spec runtime.RawExtension `json:"spec"`
}

// CephObjectZone stands for object store multisite zone creation and configuration.
type CephObjectZone struct {
	// Name of zone
	Name string `json:"name"`
	// Spec stands for zone configuration.
	// https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-zone-crd/
	// for available options
	Spec runtime.RawExtension `json:"spec"`
}

type CephStorageClassSpec struct {
	// Default represents whether Ceph Pool's StorageClass would be default or not
	// +optional
	Default bool `json:"default,omitempty"`
	// MapOptions is a comma-separated list of kernel RBD map options
	// +nullable
	MapOptions string `json:"mapOptions,omitempty"`
	// UnmapOptions is a comma-separated list of kernel RBD unmap options
	// +nullable
	UnmapOptions string `json:"unmapOptions,omitempty"`
	// ImageFeatures is a comma-separated list of RBD image features,
	// see: https://docs.ceph.com/en/latest/man/8/rbd/#cmdoption-rbd-image-feature
	// Default is layering.
	// +nullable
	ImageFeatures string `json:"imageFeatures,omitempty"`
	// ReclaimPolicy stands for underlying StorageClass reclaimPolicy parameter.
	// Default is 'Delete' if not set.
	// +nullable
	ReclaimPolicy string `json:"reclaimPolicy,omitempty"`
	// AllowVolumeExpansion allows to extend volumes sizes in pool
	// +optional
	AllowVolumeExpansion bool `json:"allowVolumeExpansion,omitempty"`
}

// CephRBDMirrorSpec allows to configure RBD mirroring between two Ceph Clusters
type CephRBDMirrorSpec struct {
	// Count of rbd-mirror daemons to spawn
	Count int `json:"daemonsCount"`

	// Peers is a list of secret's names defined in kubernetes.
	// Currently, (Ceph Octopus release) only a single peer is supported
	// +optional
	Peers []CephRBDMirrorSecret `json:"peers,omitempty"`
}

type CephRBDMirrorSecret struct {
	// Site is a name of remote site associated with the token
	Site string `json:"site"`
	// Token represents base64 encoded information about
	// remote cluster; contains fsid,client_id,key,mon_host
	Token string `json:"token"`
	// Pools is a list of Ceph Pools names to mirror
	// +optional
	Pools []string `json:"pools,omitempty"`
}

type CephSharedFilesystem struct {
	// CephFilesystems to create and configure
	// +optional
	Filesystems []CephFilesystem `json:"cephFilesystems,omitempty"`
}

// CephFilesystem stands for object store multisite zone creation and configuration.
type CephFilesystem struct {
	// CephFilesystem name
	Name string `json:"name"`
	// FsSpec represents CephFilesystem configuration
	// https://rook.io/docs/rook/v1.19/CRDs/Shared-Filesystem/ceph-filesystem-crd/
	// for available options
	FsSpec runtime.RawExtension `json:"spec"`
}

// CSI provides an ability to specify CephCSI Drivers and OperatorConfig objects
type CephCSI struct {
	// OperatorConfig provides CephCSI OperatorConfig spec description
	// +optional
	OperatorConfig *CephCSIOperatorConfig `json:"operatorConfig,omitempty"`
	// Drivers provides CephCSI Drivers spec description
	// +optional
	Drivers []CephCSIDriver `json:"drivers,omitempty"`
}

type CephCSIOperatorConfig struct {
	// FullOverride fully override default or manually created OperatorConfig
	// if present. Otherwise, provided spec will be merged with existed.
	// +optional
	FullOverride bool `json:"fullOverride,omitempty"`
	// Spec represents OperatorConfig configuration
	// https://github.com/ceph/ceph-csi-operator/blob/v1.0.4/docs/design/operator.md#operatorconfig-crd
	Spec runtime.RawExtension `json:"spec"`
}

type CephCSIDriver struct {
	// Driver type, could be one of "nvmeof,rbd,cephfs,nfs"
	// +kubebuilder:validation:Enum=rbd;cephfs;nfs;nvmeof
	Type CSIDriverType `json:"type"`
	// FullOverride fully override default or manually created Driver
	// if present. Otherwise, provided spec will be merged with existed.
	// +optional
	FullOverride bool `json:"fullOverride,omitempty"`
	// Spec represents Driver configuration
	// https://github.com/ceph/ceph-csi-operator/blob/v1.0.4/docs/design/operator.md#driver-crd
	Spec runtime.RawExtension `json:"spec"`
}

type CSIDriverType string

const (
	RBDCSIDriver    CSIDriverType = "rbd"
	CephFSCSIDriver CSIDriverType = "cephfs"
	NFSCSIDriver    CSIDriverType = "nfs"
	NVMEoFCSIDriver CSIDriverType = "nvmeof"
)

type CephDeploymentPhase string

const (
	PhaseCreating    CephDeploymentPhase = "Creating"
	PhaseDeploying   CephDeploymentPhase = "Deploying"
	PhaseValidation  CephDeploymentPhase = "Validation"
	PhaseReady       CephDeploymentPhase = "Ready"
	PhaseOnHold      CephDeploymentPhase = "OnHold"
	PhaseMaintenance CephDeploymentPhase = "Maintenance"
	PhaseDeleting    CephDeploymentPhase = "Deleting"
	PhaseFailed      CephDeploymentPhase = "Failed"
)

// CephDeploymentStatus defines the observed state of MiraCeph
type CephDeploymentStatus struct {
	//+kubebuilder:default=Creating
	// Phase is a current MiraCeph handling phase
	Phase CephDeploymentPhase `json:"phase"`
	// Message is a description of a current phase if exists
	// +nullable
	Message string `json:"message,omitempty"`
	// Validation reflects validation result for spec
	// +optional
	Validation CephDeploymentValidation `json:"validation,omitempty"`
	// Current Ceph cluster version(s)
	// +nullable
	ClusterVersion string `json:"clusterVersion,omitempty"`
	// Last MiraCeph reconcile run time
	// +nullable
	LastRun string `json:"lastRun,omitempty"`
	// objects refs
	// +optional
	ObjectsRefs []v1.ObjectReference `json:"objRefs,omitempty"`
}

type ValidationResult string

const (
	ValidationFailed  ValidationResult = "Failed"
	ValidationSucceed ValidationResult = "Succeed"
)

// CephDeploymentValidation reflects validation result for MiraCeph spec
type CephDeploymentValidation struct {
	// Result is a spec validation result, which could be Succeed or Failed
	Result ValidationResult `json:"result,omitempty"`
	// Last validated miraCeph generation version
	// +optional
	LastValidatedGeneration int64 `json:"lastValidatedGeneration,omitempty"`
	// Messages represents a list of possible issues or validation messages
	// found during spec validating
	// +optional
	Messages []string `json:"messages,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CephDeploymentList contains a list of CephDeployment
type CephDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains a list of CephDeployment objects
	Items []CephDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CephDeployment{}, &CephDeploymentList{})
}

type CephDeploymentIngressConfig struct {
	// Annotations is an extra annotations set to proxy
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// ClassName is a name of Ingress Controller class. Rockoon default
	// is 'openstack-ingress-nginx'
	// +nullable
	ControllerClassName string `json:"controllerClassName,omitempty"`
	// TLSConfig represents tls configuration: certs, public domain
	// +optional
	TLSConfig *CephDeploymentIngressTLSConfig `json:"tlsConfig,omitempty"`
}

type CephDeploymentIngressTLSConfig struct {
	// TLSCerts contains TLS certs for ingress
	// +optional
	TLSCerts *CephDeploymentCert `json:"certs,omitempty"`
	// TLSSecretRefName is a name of secret, where tls certs for ingress is stored
	// +optional
	TLSSecretRefName string `json:"tlsSecretRefName,omitempty"`
	// Domain is a public domain used for ingress public endpoint
	Domain string `json:"publicDomain"`
	// Ingress hostname different from RGW Objectstore name
	// +optional
	Hostname string `json:"hostname,omitempty"`
}

// CephDeploymentCert represents custom certificate settings
type CephDeploymentCert struct {
	// Cacert represents CA certificate
	Cacert string `json:"cacert"`
	// TLSCert represents SSL certificate based on the defined Cacert and TLSKey
	TLSCert string `json:"tlsCert"`
	// TLSKey represents SSL secret key used for TLSCert generate
	TLSKey string `json:"tlsKey"`
}
