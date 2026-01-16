package v1alpha1

import (
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	DashboardEnabled bool `json:"dashboard"`
	// Clients is a list of Ceph Clients used for Ceph Cluster connection by
	// consumer services
	// +optional
	Clients []CephClient `json:"clients,omitempty"`
	// DataDirHostPath is a default hostPath directory where Rook stores all
	// valuable info. Equals to '/var/lib/rook' by default
	// +nullable
	DataDirHostPath string `json:"dataDirHostPath,omitempty"`
	// External enables usage of external Ceph Cluster connected to pkg
	// Container Cloud cluster instead of local Ceph Cluster
	// +optional
	External bool `json:"external,omitempty"`
	// ExtraOpts contains some extra options for managing Ceph cluster, like devices labels
	// +optional
	ExtraOpts *CephDeploymentExtraOpts `json:"extraOpts,omitempty"`
	// HealthCheck provides an ability to configure pkg daemon healthchecks
	// and liveness probe settings for mon,mgr,osd daemons
	// +optional
	HealthCheck *CephClusterHealthCheckSpec `json:"healthCheck,omitempty"`
	// HyperConverge provides an ability to configure resources requests and limitations
	// for Ceph Daemons. Also provides an ability to spawn those Ceph Daemons on a tainted
	// nodes
	// +optional
	HyperConverge *CephDeploymentHyperConverge `json:"hyperconverge,omitempty"`
	// IngressConfig provides ability to configure custom ingress rule for an external
	// access to Ceph Cluster resources, for example, public endpoint
	// for Ceph Object Store access.
	// +optional
	IngressConfig *CephDeploymentIngressConfig `json:"ingressConfig,omitempty"`
	// Mgr contains a list of Ceph Manager modules to enable in Ceph Cluster
	// +optional
	Mgr *Mgr `json:"mgr,omitempty"`
	// Network is a section which defines the specific network range(s)
	// for Ceph daemons to communicate with each other and the an external
	// connections
	Network CephNetworkSpec `json:"network"`
	// Nodes contains full cluster nodes configuration to use as Ceph Nodes
	Nodes []CephDeploymentNode `json:"nodes"`
	// ObjectStorage contains full RadosGW Object Storage configurations: RGW itself
	// and RGW multisite feature
	// +optional
	ObjectStorage *CephObjectStorage `json:"objectStorage,omitempty"`
	// Pools is a list of Ceph RBD Pools configurations
	// +optional
	Pools []CephPool `json:"pools,omitempty"`
	// RBDMirror allows to configure RBD mirroring between two Ceph Clusters
	// +optional
	RBDMirror *CephRBDMirrorSpec `json:"rbdMirror,omitempty"`
	// RookConfig is a key-value mapping which contains ceph config keys with a specified values
	// +optional
	RookConfig map[string]string `json:"rookConfig,omitempty"`
	// SharedFilesystem enables such system as CephFS
	// +optional
	SharedFilesystem *CephSharedFilesystem `json:"sharedFilesystem,omitempty"`
}

type CephClient struct {
	ClientSpec `json:",inline"`
}

type ClientSpec struct {
	// +optional
	Name string `json:"name,omitempty"`
	// SecretName is the name of the secret created for this ceph client.
	// If not specified, the default name is "rook-ceph-client-" as a prefix to the CR name.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// RemoveSecret indicates whether the current secret for this ceph client should be removed or not.
	// If true, the K8s secret will be deleted, but the cephx keyring will remain until the CR is deleted.
	// +optional
	RemoveSecret bool `json:"removeSecret,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Caps map[string]string `json:"caps"`
}

type LabeledDevices map[string]string

// CephDeploymentExtraOpts contains extra options, used for cluster configuration and management
type CephDeploymentExtraOpts struct {
	// Custom devices classes different from default classes (ssd, hdd, nvme)
	// +optional
	CustomDeviceClasses []string `json:"customDeviceClasses,omitempty"`
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

type CephClusterHealthCheckSpec struct {
	// DaemonHealth contains health check settings for ceph daemons
	// +optional
	DaemonHealth cephv1.DaemonHealthSpec `json:"daemonHealth,omitempty"`
	// LivenessProbe allows changing the livenessProbe configuration for ceph daemons
	// +optional
	LivenessProbe map[cephv1.KeyType]*cephv1.ProbeSpec `json:"livenessProbe,omitempty"`
	// StartupProbe allows changing the startupProbe configuration for ceph daemons
	// +optional
	StartupProbe map[cephv1.KeyType]*cephv1.ProbeSpec `json:"startupProbe,omitempty"`
}

// CephDeploymentHyperConverge represents hyperconverge parameters for Ceph daemons
type CephDeploymentHyperConverge struct {
	// Resources requirements for ceph daemons, such as: mon, mgr, mds, rgw, osd, osd-hdd, osd-ssd, osd-nvme, prepareosd
	// +optional
	Resources cephv1.ResourceSpec `json:"resources,omitempty"`
	// Tolerations rules for ceph daemons: osd, mon, mgr.
	// +optional
	Tolerations map[string]CephDeploymentToleration `json:"tolerations,omitempty"`
}

// CephDeploymentToleration represents kubernetes toleration rules
type CephDeploymentToleration struct {
	// Rules is a list of kubernetes tolerations defined for some
	// Ceph daemon
	Rules []v1.Toleration `json:"rules"`
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

// Mgr contains a list of Ceph Manager modules to enable in Ceph Cluster
type Mgr struct {
	// MgrModules is a list of Ceph Manager modules names to enable in Ceph
	// +optional
	MgrModules []CephMgrModule `json:"mgrModules,omitempty"`
}

// CephMgrModule represents mgr modules that the user wants to enable or disable
type CephMgrModule struct {
	// Name is the name of the ceph manager module
	// +nullable
	Name string `json:"name,omitempty"`
	// Enabled determines whether a module should be enabled or not
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// Settings reflects mgr module settings if required
	// +optional
	Settings *CephMgrModuleSettings `json:"settings,omitempty"`
}

// CephMgrModuleSettings represents mgr modules settings
type CephMgrModuleSettings struct {
	// BalancerMode sets the `balancer` module with different modes like `upmap`, `crush-compact` etc
	BalancerMode string `json:"balancerMode,omitempty"`
}

// CephNetworkSpec is a section which defines the specific network range(s)
// for Ceph daemons to communicate with each other and the an external
// connections
type CephNetworkSpec struct {
	// ClusterNet defines pkg network for Ceph Daemons intra-communication
	ClusterNet string `json:"clusterNet"`
	// ClusterNet defines public network for an external access to Ceph Cluster
	PublicNet string `json:"publicNet"`
	// MonOnPublicNet defines monitors on public network instead of LCM network
	// +optional
	MonOnPublicNet bool `json:"monOnPublicNet,omitempty"`
	// Provider specifies the network provider that will be used to connect the network interface
	// +nullable
	Provider string `json:"provider,omitempty"`
	// Selector is used for multus provider only. Select NetworkAttachmentDefinitions to use for Ceph networks
	// +optional
	Selector map[cephv1.CephNetworkType]string `json:"selector,omitempty"`
	// HostNetwork is deprecated field, always true to have persistan mons ips
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty"`
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
	// Rgw represents Ceph RadosGW settings
	Rgw CephRGW `json:"rgw"`
	// MultiSite represents Ceph RadosGW multisite/multizone feature settings
	// +optional
	MultiSite *CephMultiSite `json:"multiSite,omitempty"`
}

// CephRGW represents Ceph RadosGW settings
type CephRGW struct {
	// Name represents the name of specified object storage
	// +kubebuilder:validation:MaxLength:=25
	Name string `json:"name"`
	// Users is a list of user names to create for object storage
	// with radosgw-admin
	// +optional
	ObjectUsers []CephRGWUser `json:"objectUsers,omitempty"`
	// Buckets is a list of initial buckets to create in object storage
	// with radosgw-admin
	// +optional
	Buckets []string `json:"buckets,omitempty"`
	// Replicas is a number of replicas for each Ceph RadosGW instance.
	// Not used in a product currently
	// +optional
	Replicas *int `json:"replicas,omitempty"`
	// Whether host networking is enabled for the rgw daemon.
	// If not set, the network settings from the cluster CR will be applied.
	// +optional
	// +nullable
	RgwUseHostNetwork *bool `json:"rgwUseHostNetwork,omitempty"`
	// MetadataPool represents Ceph Pool's settings which stores RGW metadata.
	// Mutually exclusive with Zone
	// +optional
	MetadataPool *CephPoolSpec `json:"metadataPool,omitempty"`
	// DataPool represents Ceph Pool's settings which stores RGW data.
	// Mutually exclusive with Zone
	// +optional
	DataPool *CephPoolSpec `json:"dataPool,omitempty"`
	// PreservePoolsOnDelete is a flag whether keep RGW metadata/data pools
	// on RGW delete or not
	// +optional
	PreservePoolsOnDelete bool `json:"preservePoolsOnDelete"`
	// Gateway represents Ceph RGW daemons settings
	Gateway CephRGWGateway `json:"gateway"`
	// Disable auto update allowed hostnames for zone group
	// By default is enabled, but ignored for multisite.
	// +optional
	SkipAutoZoneGroupHostnameUpdate bool `json:"skipAutoZoneGroupHostnameUpdate,omitempty"`
	// SSLCert used for access to RGW Gateway endpoint, if not specified will be generated self-signed
	// +optional
	SSLCert *CephDeploymentCert `json:"SSLCert,omitempty"`
	// SSLCertInRef is a flag, whether RGW SSL certs are provided internally,
	// without exposing in spec in base default 'rgw-ssl-certificate' secret.
	// +optional
	SSLCertInRef bool `json:"SSLCertInRef,omitempty"`
	// Zone represents RGW zone if multisite feature enabled
	// +optional
	Zone *cephv1.ZoneSpec `json:"zone,omitempty"`
	// HealthCheck represents Ceph RGW daemons healthchecks
	// +optional
	HealthCheck *cephv1.ObjectHealthCheckSpec `json:"healthCheck,omitempty"`
}

// CephRGWUser represents Ceph RadosGW user
type CephRGWUser struct {
	// Represent a user name
	Name string `json:"name"`
	// The display name for the user
	// +nullable
	DisplayName string `json:"displayName,omitempty"`
	// User capabilities
	// +optional
	Capabilities *cephv1.ObjectUserCapSpec `json:"capabilities,omitempty"`
	// User quotas
	// +optional
	Quotas *cephv1.ObjectUserQuotaSpec `json:"quotas,omitempty"`
}

// CephRGWGateway represents Ceph RGW daemon settings
type CephRGWGateway struct {
	// Port the rgw service will be listening on (http)
	Port int32 `json:"port"`
	// SecurePort the rgw service will be listening on (https)
	SecurePort int32 `json:"securePort"`
	// Instances is the number of pods in the rgw replicaset.
	// If AllNodes is specified, a daemonset will be created.
	Instances int32 `json:"instances"`
	// AllNodes is a flag whether the rgw pods should be
	// started as a daemonset on all nodes
	AllNodes bool `json:"allNodes"`
	// SplitDaemonForMultisiteTrafficSync is a flag for multisite, which allows
	// to split daemon responsible for sync between zones and daemon for serving clients requests
	// +optional
	SplitDaemonForMultisiteTrafficSync bool `json:"splitDaemonForMultisiteTrafficSync,omitempty"`
	// Port the rgw multisite traffic service will be listening on (http). Optional.
	// Has effect only for multisite configuration.
	// +optional
	RgwSyncPort int32 `json:"rgwSyncPort,omitempty"`
	// Resources requirements for RGW instances
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// ExternalRgwEndpoint represents external RGW Endpoint to use, when external Ceph cluster is used.
	// Has effect only for external cluster setup.
	// +optional
	ExternalRgwEndpoint *cephv1.EndpointAddress `json:"externalRgwEndpoint,omitempty"`
}

// CephMultiSite represents Ceph RadosGW multisite/multizone feature settings
type CephMultiSite struct {
	// Realms is a list of Ceph Object storage multisite realms
	Realms []CephRGWRealm `json:"realms"`
	// ZoneGroups is a list of Ceph Object storage multisite zonegroups
	ZoneGroups []CephRGWZoneGroup `json:"zoneGroups"`
	// Zones is a list of Ceph Object storage multisite zones
	Zones []CephRGWZone `json:"zones"`
}

// CephRGWRealm represents RGW multisite realm namespace
type CephRGWRealm struct {
	// Name represents realm's name
	Name string `json:"name"`
	// Pull stands for the Endpoint, the access key and the system key
	// of the system user from the realm being pulled from
	// +optional
	Pull *CephRGWRealmPull `json:"pullEndpoint,omitempty"`
	// Set this realm as the default in Ceph. Only one realm should be default.
	// +optional
	DefaultRealm bool `json:"defaultRealm,omitempty"`
}

// CephRGWRealmPull stands for the Endpoint, the access key and the system key
// of the system user from the realm being pulled from
type CephRGWRealmPull struct {
	// Endpoint represents an endpoint from the master zone in the master zone group
	Endpoint string `json:"endpoint"`
	// AccessKey is an access key of the system user from the realm being pulled from
	AccessKey string `json:"accessKey"`
	// SecretKey is a system key of the system user from the realm being pulled from
	SecretKey string `json:"secretKey"`
}

// CephRGWZoneGroup represents multisite zone group
type CephRGWZoneGroup struct {
	// Name represents zone group's name
	Name string `json:"name"`
	// Realm is a name of the realm for which zone group belongs to
	Realm string `json:"realmName"`
}

// CephRGWZone represents multisite zone
type CephRGWZone struct {
	// Name represents zone's name
	Name string `json:"name"`
	// MetadataPool represents Ceph Pool's setting which contains
	// RGW zone metadata
	MetadataPool CephPoolSpec `json:"metadataPool"`
	// DataPool represents Ceph Pool's setting which contains
	// RGW zone data
	DataPool CephPoolSpec `json:"dataPool"`
	// ZoneGroup is a name of the zone group for which zone belongs to
	ZoneGroup string `json:"zoneGroupName"`
	// Custom endpoints for zone, which should be used in zone config
	// +optional
	EndpointsForZone []string `json:"endpointsForZone,omitempty"`
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

	CephPoolSpec `json:",inline"`
}

type CephPoolSpec struct {
	// Replicated represents Ceph Pool's replica settings
	// +optional
	Replicated *CephPoolReplicatedSpec `json:"replicated,omitempty"`
	// FailureDomain represents level of cluster fault-tolerance.
	// Possible values are: osd, host, region or zone if available;
	// technically also any type in the crush map
	// +nullable
	FailureDomain string `json:"failureDomain,omitempty"`
	// CrushRoot is the root of the crush hierarchy utilized by the pool
	// +nullable
	CrushRoot string `json:"crushRoot,omitempty"`
	// DeviceClass is the device class the OSD should set to (options are: hdd, ssd, or nvme)
	DeviceClass string `json:"deviceClass"`
	// ErasureCoded represents Ceph Pool's erasure coding settings
	// +optional
	ErasureCoded *CephPoolErasureCodedSpec `json:"erasureCoded,omitempty"`
	// Mirroring allows to enable RBD mirroring feature in modes: pool, image
	// +optional
	Mirroring *CephPoolMirrorSpec `json:"mirroring,omitempty"`
	// Parameters is a key-value mapping of all supported ceph pool parameters such
	// as pg_num, compression_mode etc.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
	// EnableCrushUpdates enables rook to update the pool crush rule using Pool Spec.
	// Can cause data remapping if crush rule changes, Defaults to false.
	// +optional
	// +nullable
	EnableCrushUpdates *bool `json:"enableCrushUpdates,omitempty"`
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

// CephPoolErasureCodedSpec represents the spec for erasure code in a pool
type CephPoolErasureCodedSpec struct {
	// CodingChunks is a number of coding chunks per object
	// in an erasure coded storage pool (required for erasure-coded pool type)
	CodingChunks uint `json:"codingChunks"`
	// DataChunks is a number of data chunks per object
	// in an erasure coded storage pool (required for erasure-coded pool type)
	DataChunks uint `json:"dataChunks"`
	// Algorithm represents the algorithm for erasure coding
	// +nullable
	Algorithm string `json:"algorithm,omitempty"`
}

// CephPoolReplicatedSpec represents the spec for replication in a pool
type CephPoolReplicatedSpec struct {
	// Size - Number of copies per object in a replicated storage pool, including the object itself (required for replicated pool type)
	Size uint `json:"size"`
	// TargetSizeRatio gives a hint (%) to Ceph in terms of expected consumption of the total cluster capacity
	// +optional
	TargetSizeRatio float64 `json:"targetSizeRatio,omitempty"`
}

// CephPoolMirrorSpec spec represents RBD mirroring
// settings for a specific Ceph RBD Pool
type CephPoolMirrorSpec struct {
	// Mode - mirroring mode to run
	Mode string `json:"mode"`
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
	// CephFS to create. Multiple CephFS available to deploy for clusters with Ceph Reef or above
	// +optional
	CephFS []CephFS `json:"cephFS,omitempty"`
}

type CephFS struct {
	// CephFS name
	Name string `json:"name"`
	// The settings used to create the filesystem metadata pool. Must use replication.
	MetadataPool CephPoolSpec `json:"metadataPool"`
	// The settings to create the filesystem data pools. Must use replication.
	// +optional
	DataPools []CephFSPool `json:"dataPools,omitempty"`
	// When set to ‘true’ the filesystem will remain when the CephFilesystem resource is deleted
	// This is a security measure to avoid loss of data if the CephFilesystem resource is deleted accidentally.
	// +optional
	PreserveFilesystemOnDelete bool `json:"preserveFilesystemOnDelete,omitempty"`
	// Metadata server settings correspond to the MDS daemon settings
	MetadataServer CephMetadataServer `json:"metadataServer"`
}

// CephFSPool stands for specified CephFS Pool configuration
type CephFSPool struct {
	// Name represents CephFS pool name
	Name string `json:"name"`

	CephPoolSpec `json:",inline"`
}

type CephMetadataServer struct {
	// The number of active MDS instances. As load increases, CephFS will automatically
	// partition the filesystem across the MDS instances. Rook will create double the
	// number of MDS instances as requested by the active count. The extra instances will
	// be in standby mode for failover
	ActiveCount int32 `json:"activeCount"`
	// If true, the extra MDS instances will be in active standby mode and will keep
	// a warm cache of the filesystem metadata for faster failover. The instances will
	// be assigned by CephFS in failover pairs. If false, the extra MDS instances will
	// all be on passive standby mode and will not maintain a warm cache of the metadata.
	// +optional
	ActiveStandby bool `json:"activeStandby,omitempty"`
	// Resources represents kubernetes resource requirements for mds instances
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// HealthCheck provides an ability to configure mds daemon healthchecks
	// +optional
	HealthCheck *CephMdsHealthCheck `json:"healthCheck,omitempty"`
}

type CephMdsHealthCheck struct {
	// LivenessProbe allows changing the livenessProbe configuration for ceph mds daemon
	// +optional
	LivenessProbe *cephv1.ProbeSpec `json:"livenessProbe,omitempty"`
	// StartupProbe allows changing the startupProbe configuration for ceph mds daemon
	// +optional
	StartupProbe *cephv1.ProbeSpec `json:"startupProbe,omitempty"`
}

// MiraIngress provides an ability to configure custom ingress rule for an external
// access to Ceph Cluster resources, for example, public endpoint
// for Ceph Object Store access
type MiraIngress struct {
	// Domain is a public domain used for ingress public endpoint
	Domain string `json:"publicDomain"`

	CephDeploymentCert `json:",inline"`

	// CustomIngress represents Extra/Custom Ingress configuration
	// +optional
	CustomIngress *CephDeploymentCustomIngress `json:"customIngress,omitempty"`
}

// CephDeploymentCustomIngress represents custom Ingress Controller configuration
type CephDeploymentCustomIngress struct {
	// ClassName is a name of Ingress Controller class. Default for
	// MOS cloud is 'openstack-ingress-nginx'
	// +nullable
	ClassName string `json:"className,omitempty"`
	// Annotations is an extra annotations set to proxy
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

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
