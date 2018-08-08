package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionStatus represents status of the EtcdStorage condition.
type ConditionStatus string

// These are valid condition statuses: ConditionTrue, ConditionFalse, ConditionUnknown.
const (
	// ConditionTrue means a resource is in the condition.
	ConditionTrue ConditionStatus = "True"
	// ConditionFalse means a resource is not in the condition.
	ConditionFalse ConditionStatus = "False"
	// ConditionUnknown means controller can't decide if a EtcdStorage resource is in the condition or not.
	ConditionUnknown ConditionStatus = "Unknown"
)

// EtcdStorageConditionType represents condition of the EtcdStorage resource.
type EtcdStorageConditionType string

const (
	// Deployed means EtcdProxy Deployment and Service for exposing EtcdProxy are created.
	Deployed EtcdStorageConditionType = "Deployed"
)

// CABundleDestination contains name and namespace of configmap where CA bundle is stored.
type CABundleDestination struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ClientCertificateDestination contains name and namespace of secret where client certificate and key are stored.
type ClientCertificateDestination struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EtcdStorage is a specification for a EtcdStorage resource
type EtcdStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtdcStorageSpec   `json:"spec"`
	Status EtcdStorageStatus `json:"status"`
}

// EtdcStorageSpec is the spec for a EtcdStorage resource
type EtdcStorageSpec struct {
	// CACertConfigMaps contains name and namespace of ConfigMap where CA serving certificate for etcdproxy pod
	// is supposed to be deployed. Usually it is in aggregated API server namespace.
	CACertConfigMaps []CABundleDestination `json:"caCertConfigMap"`

	// ClientCertSecrets contains name and namespace of Secret where client certificate and key for etcdproxy pod
	// is supposed to be deployed. Usually it is in aggregated API server namespace.
	ClientCertSecrets []ClientCertificateDestination `json:"clientCertSecret"`

	// SigningCertificateValidity is number of minutes for how long self-generated signing certificate is valid.
	SigningCertificateValidity metav1.Duration `json:"signingCertificateValidity"`

	// ServingCertificateValidity is number of minutes for how long serving certificate/key pair is valid.
	ServingCertificateValidity metav1.Duration `json:"servingCertificateValidity"`

	// ClientCertificateValidity is number of minutes for how long client certificate/key pair is valid.
	ClientCertificateValidity metav1.Duration `json:"clientCertificateValidity"`
}

// EtcdStorageStatus is the status for a EtcdStorage resource
type EtcdStorageStatus struct {
	// Conditions indicates states of the EtcdStroageStatus,
	Conditions []EtcdStorageCondition
}

// EtcdStorageCondition contains details for the current condition of this EtcdStorage instance.
type EtcdStorageCondition struct {
	// Type is the type of the condition.
	Type EtcdStorageConditionType
	// Status is the status of the condition (true, false, unknown).
	Status ConditionStatus
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time
	// Unique, one-word, CamelCase reason for the condition's last transition.
	Reason string
	// Human-readable message indicating details about last transition.
	Message string
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EtcdStorageList is a list of EtcdStorage resources
type EtcdStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EtcdStorage `json:"items"`
}
