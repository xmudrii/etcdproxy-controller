package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FindEtcdStorageCondition returns the condition you're looking for, or nil if condition is not present.
func FindEtcdStorageCondition(es *EtcdStorage, conditionType EtcdStorageConditionType) *EtcdStorageCondition {
	for i := range es.Status.Conditions {
		if es.Status.Conditions[i].Type == conditionType {
			return &es.Status.Conditions[i]
		}
	}

	return nil
}

// IsEtcdStorageConditionPresentAndEqual checks is condition present with the status equal to passed argument.
func IsEtcdStorageConditionPresentAndEqual(es *EtcdStorage, conditionType EtcdStorageConditionType, status ConditionStatus) bool {
	for _, cond := range es.Status.Conditions {
		if cond.Type == conditionType {
			return cond.Status == status
		}
	}
	return false
}

// IsEtcdStorageConditionTrue checks is condition present and true.
func IsEtcdStorageConditionTrue(es *EtcdStorage, conditionType EtcdStorageConditionType) bool {
	return IsEtcdStorageConditionPresentAndEqual(es, conditionType, ConditionTrue)
}

// IsEtcdStorageConditionFalse checks is condition present and false.
func IsEtcdStorageConditionFalse(es *EtcdStorage, conditionType EtcdStorageConditionType) bool {
	return IsEtcdStorageConditionPresentAndEqual(es, conditionType, ConditionFalse)
}

// IsEtcdStorageConditionEquivalent returns true if conditions are same expect for LastTransitionTimes.
func IsEtcdStorageConditionEquivalent(l, r *EtcdStorageCondition) bool {
	if l == nil && r == nil {
		return true
	}
	if l == nil || r == nil {
		return false
	}

	return l.Message == r.Message && l.Reason == r.Reason && l.Status == r.Status && l.Type == r.Type
}

// SetEtcdStorageCondition applies Condition to the EtcdStorage instance provided as the argument.
// If the condition already exists in the EtcdStorage instance, it's overwritten.
func SetEtcdStorageCondition(es *EtcdStorage, newCondition EtcdStorageCondition) {
	existingCondition := FindEtcdStorageCondition(es, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		es.Status.Conditions = append(es.Status.Conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = newCondition.LastTransitionTime
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
}
