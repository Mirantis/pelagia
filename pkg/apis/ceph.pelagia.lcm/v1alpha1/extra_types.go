package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:resource:path=cephdeploymentsecrets,scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Secret state"
// +kubebuilder:printcolumn:name="Last check",type=string,JSONPath=`.status.lastSecretCheck`,description="Last secret update"
// +kubebuilder:printcolumn:name="Last update",type=string,JSONPath=`.status.lastSecretUpdate`,description="Last secret update"
// +kubebuilder:resource:shortName={cephdplsecret}
// +kubebuilder:subresource:status
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CephDeploymentSecret aggregates secrets created by Ceph
type CephDeploymentSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status contains overall status and secrets info for MiraCephSecret
	// +optional
	Status *CephDeploymentSecretStatus `json:"status,omitempty"`
}

// CephDeploymentSecretsInfo contains secret names for client and rgw users
type CephDeploymentSecretsInfo struct {
	// ClientSecrets list of secrets info for ceph clients
	// +optional
	ClientSecrets []CephDeploymentSecretInfo `json:"clientSecrets,omitempty"`
	// RgwUserSecrets list of secrets info for ceph radosgw users
	// +optional
	RgwUserSecrets []CephDeploymentSecretInfo `json:"rgwUserSecrets,omitempty"`
}

// CephDeploymentSecretInfo contains secret name, namespace and object name, associated with secret
type CephDeploymentSecretInfo struct {
	ObjectName      string `json:"name"`
	SecretName      string `json:"secretName"`
	SecretNamespace string `json:"secretNamespace"`
}

// CephDeploymentSecretStatus detects issues for MiraCephSecret object
type CephDeploymentSecretStatus struct {
	// State represents the state for overall secret status
	State CephDeploymentHealthState `json:"state"`
	// LastUpdate is a last time when MiraCephSecret was updated
	// +nullable
	LastSecretUpdate string `json:"lastSecretUpdate,omitempty"`
	// LastUpdate is a last time when MiraCephSecret was updated
	// +nullable
	LastSecretCheck string `json:"lastSecretCheck,omitempty"`
	// SecretsInfo contains Ceph secret names for related objects
	// +optional
	SecretsInfo *CephDeploymentSecretsInfo `json:"secretInfo,omitempty"`
	// Messages is a list with any possible error/warning messages
	// +optional
	Messages []string `json:"messages,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CephDeploymentSecretList represents a list of CephDeploymentSecret objects
type CephDeploymentSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items represents a list of CephDeploymentSecret objects
	Items []CephDeploymentSecret `json:"items"`
}

// +kubebuilder:resource:path=cephdeploymentmaintenances,scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="Message for current state"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Current maintenance state"
// +kubebuilder:printcolumn:name="Last check",type=string,JSONPath=`.status.lastStateCheck`,description="Last state check time"
// +kubebuilder:resource:shortName={cephdplmnt}
// +kubebuilder:subresource:status
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CephDeploymentMaintenance aggregates info about miraceph maintenance
type CephDeploymentMaintenance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status contains overall status info for CephDeploymentMaintenance
	Status *CephDeploymentMaintenanceStatus `json:"status,omitempty"`
}

type CephDeploymentMaintenanceState string

const (
	MaintenanceIdle    CephDeploymentMaintenanceState = "Idle"
	MaintenanceActing  CephDeploymentMaintenanceState = "Acting"
	MaintenanceFailing CephDeploymentMaintenanceState = "Failing"
)

type CephDeploymentMaintenanceStatus struct {
	// current state of maintenance
	State CephDeploymentMaintenanceState `json:"state"`
	// last state check timestamp
	LastStateCheck string `json:"lastStateCheck"`
	// message for current state
	// +nullable
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// CephDeploymentMaintenanceList represents a list of CephDeploymentMaintenance objects
type CephDeploymentMaintenanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items represents a list of CephDeploymentMaintenance objects
	Items []CephDeploymentMaintenance `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&CephDeploymentSecret{}, &CephDeploymentSecretList{}, &CephDeploymentMaintenance{}, &CephDeploymentMaintenanceList{})
}
