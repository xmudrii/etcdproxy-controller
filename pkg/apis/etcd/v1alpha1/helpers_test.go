package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsEtcdStorageConditionPresentAndEqual(t *testing.T) {
	tests := []struct {
		name            string
		esConditions    []EtcdStorageCondition
		conditionType   EtcdStorageConditionType
		conditionStatus ConditionStatus
		expectedResult  bool
	}{
		{
			name: "etcdstorage condition present and true",
			esConditions: []EtcdStorageCondition{
				{
					Type:    Deployed,
					Status:  ConditionTrue,
					Reason:  "EtcdProxyDeployed",
					Message: "EtcdProxy ReplicaSet and Service created",
				},
			},
			conditionType:   Deployed,
			conditionStatus: ConditionTrue,
			expectedResult:  true,
		},
		{
			name: "etcdstorage condition present but not equal",
			esConditions: []EtcdStorageCondition{
				{
					Type:    Deployed,
					Status:  ConditionTrue,
					Reason:  "EtcdProxyDeployed",
					Message: "EtcdProxy ReplicaSet and Service created",
				},
			},
			conditionType:   Deployed,
			conditionStatus: ConditionFalse,
			expectedResult:  false,
		},
		{
			name:            "etcdstorage condition not present",
			esConditions:    []EtcdStorageCondition{},
			conditionType:   Deployed,
			conditionStatus: ConditionTrue,
			expectedResult:  false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			es := newEtcdStorageWithConditions(tc.esConditions)
			res := IsEtcdStorageConditionPresentAndEqual(es, tc.conditionType, tc.conditionStatus)
			if res != tc.expectedResult {
				t.Fatalf("expected %v but got %v instead", tc.expectedResult, res)
			}
		})
	}
}

func TestSetEtcdStorageCondition(t *testing.T) {
	tests := []struct {
		name               string
		esConditions       []EtcdStorageCondition
		newCondition       EtcdStorageCondition
		expectedConditions []EtcdStorageCondition
	}{
		{
			name:         "etcdstorage with no conditions",
			esConditions: []EtcdStorageCondition{},
			newCondition: EtcdStorageCondition{
				Type:    Deployed,
				Status:  ConditionTrue,
				Reason:  "EtcdProxyDeployed",
				Message: "EtcdProxy ReplicaSet and Service created",
			},
			expectedConditions: []EtcdStorageCondition{
				{
					Type:    Deployed,
					Status:  ConditionTrue,
					Reason:  "EtcdProxyDeployed",
					Message: "EtcdProxy ReplicaSet and Service created",
				},
			},
		},
		{
			name: "etcdstorage with deployed false condition",
			esConditions: []EtcdStorageCondition{
				{
					Type:    Deployed,
					Status:  ConditionFalse,
					Reason:  "Deploying",
					Message: "EtcdProxy ReplicaSet and Service are creating",
				},
			},
			newCondition: EtcdStorageCondition{
				Type:    Deployed,
				Status:  ConditionTrue,
				Reason:  "EtcdProxyDeployed",
				Message: "EtcdProxy ReplicaSet and Service created",
			},
			expectedConditions: []EtcdStorageCondition{
				{
					Type:    Deployed,
					Status:  ConditionTrue,
					Reason:  "EtcdProxyDeployed",
					Message: "EtcdProxy ReplicaSet and Service created",
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			es := newEtcdStorageWithConditions(tc.esConditions)
			SetEtcdStorageCondition(es, tc.newCondition)
			if len(es.Status.Conditions) != len(tc.expectedConditions) {
				t.Fatalf("expected and present number of conditions missmatch. expected %v, got %v",
					len(tc.expectedConditions), len(es.Status.Conditions))
			}
			for i := range tc.expectedConditions {
				if !IsEtcdStorageConditionEquivalent(&tc.expectedConditions[i], &es.Status.Conditions[i]) {
					t.Fatalf("expected %v, got %v", tc.expectedConditions, es.Status.Conditions)
				}
			}
		})
	}
}

func newEtcdStorageWithConditions(conditions []EtcdStorageCondition) *EtcdStorage {
	return &EtcdStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-es",
		},
		Status: EtcdStorageStatus{
			Conditions: conditions,
		},
	}
}
