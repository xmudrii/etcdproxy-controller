package etcdproxy

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
)

func TestGetReplicaSetName(t *testing.T) {
	es := v1alpha1.EtcdStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "test-1"},
	}

	name := replicaSetName(&es)
	expectedName := "etcd-rs-test-1"
	if name != expectedName {
		t.Fatalf("incorrect replicaset name. expected: %s, got: %s", name, expectedName)
	}
}

func TestGetServiceName(t *testing.T) {
	es := v1alpha1.EtcdStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "test-1"},
	}

	name := serviceName(&es)
	expectedName := "etcd-test-1"
	if name != expectedName {
		t.Fatalf("incorrect replicaset name. expected: %s, got: %s", name, expectedName)
	}
}

func TestGetFlagFromString(t *testing.T) {
	cases := []struct {
		name         string
		key          string
		value        string
		expectedFlag string
	}{
		{
			name:         "basic flag",
			key:          "test",
			value:        "testing",
			expectedFlag: "--test=testing",
		},
		{
			name:         "multipart flag",
			key:          "test-multipart",
			value:        "testing",
			expectedFlag: "--test-multipart=testing",
		},
		{
			name:         "flag with escaped value",
			key:          "test",
			value:        "\"testing test\"",
			expectedFlag: "--test=\"testing test\"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flag := flagfromString(tc.key, tc.value)
			if flag != tc.expectedFlag {
				t.Fatalf("incorrect flag. expected: %s, got: %s", tc.expectedFlag, flag)
			}
		})
	}
}
