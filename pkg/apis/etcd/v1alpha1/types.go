package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means controller
// can't decide if a resource is in the condition or not.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

type EtcdStorageConditionType string

const (
	// Deployed means EtcdProxy ReplicaSet and Service for exposing EtcdProxy are created.
	Deployed EtcdStorageConditionType = "Deployed"
)

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
