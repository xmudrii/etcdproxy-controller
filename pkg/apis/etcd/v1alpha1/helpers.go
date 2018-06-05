/*
Copyright 2018 The Kubernetes Authors.

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

// IsEtcdStorageConditionTrue checks is condition present and false.
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
