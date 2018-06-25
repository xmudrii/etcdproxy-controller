package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	clientset "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// newKubeClient creates new Kubernetes client instance based on kubeconfig file.
func newKubeClient() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// newEtcdProxyClient creates new client for etcdstorage resources.
func newEtcdProxyClient() (*clientset.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}

	return clientset.NewForConfig(config)
}

// TestDeployEtcdStorage deploys EtcdStorage resource and then tests are all relevant resources created and running.
func TestDeployEtcdStorage(t *testing.T) {
	// Initialize clients.
	client, err := newKubeClient()
	if err != nil {
		t.Fatalf("unable to create kubernetes client from provided kubeconfig: %v", err)
	}

	etcdproxyClient, err := newEtcdProxyClient()
	if err != nil {
		t.Fatalf("unable to create etcdproxy client from provided kubeconfig: %v", err)
	}

	tests := []struct {
		name                   string
		etcdStorage            *v1alpha1.EtcdStorage
		etcdStorageValid       bool
		expectedReplicaSetName string
		expectedReplicas       int32
		expectedServiceName    string
	}{
		{
			name: "test simple etcdstorage creation",
			etcdStorage: &v1alpha1.EtcdStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name: "es-test-1",
				},
			},
			expectedReplicaSetName: "etcd-rs-es-test-1",
			expectedReplicas:       int32(1),
			expectedServiceName:    "etcd-es-test-1",
			etcdStorageValid:       true,
		},
		{
			name: "test etcdstorage creation - name too long",
			etcdStorage: &v1alpha1.EtcdStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name: "es-test-1-this-name-is-too-long-limit-foo-bar-baz-foo-bar-baz-123",
				},
			},
			etcdStorageValid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create an EtcdStorage instance.
			es, err := etcdproxyClient.EtcdV1alpha1().EtcdStorages().Create(tc.etcdStorage)
			if err != nil && tc.etcdStorageValid {
				t.Fatalf("unable to create etcdstorage '%s': %v", tc.etcdStorage.Name, err)
			}

			// Only continue if etcdStorage is expected to be valid and there is no error.
			if tc.etcdStorageValid && err == nil {
				// It takes short amount time for the ReplicaSet and Service to be created, so we need to poll.
				err = wait.Poll(500*time.Millisecond, wait.ForeverTestTimeout, func() (bool, error) {
					es, err = etcdproxyClient.EtcdV1alpha1().EtcdStorages().Get(es.Name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					if es.Status.Conditions != nil {
						return true, nil
					}
					return false, nil
				})
				if err != nil {
					t.Fatalf("deployed condition for etcdstorage '%s' not set, and is expected: %v", tc.etcdStorage.Name, err)
				}

				// We currently have only one condition, so we're making sure that one is set.
				for _, cond := range es.Status.Conditions {
					if cond.Type != "Deployed" {
						t.Fatalf("expected 'Deployed' condition, but got: %s", cond.Type)
					}
					if cond.Status != v1alpha1.ConditionTrue {
						t.Fatalf("expected condition 'Deployed' true, but got: %v", cond.Status)
					}
				}

				// Check is the ReplicaSet created and wait for pods to become ready.
				err = wait.Poll(500*time.Millisecond, wait.ForeverTestTimeout, func() (bool, error) {
					rs, err := client.AppsV1().ReplicaSets("kube-apiserver-storage").Get(tc.expectedReplicaSetName, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					if rs.Status.ReadyReplicas != tc.expectedReplicas {
						return false, nil
					}
					return true, nil
				})
				if err != nil {
					t.Fatalf("expected replicaset '%s', but got: %v", tc.expectedReplicaSetName, err)
				}

				// Check is Service created.
				_, err = client.CoreV1().Services("kube-apiserver-storage").Get(tc.expectedServiceName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("expected service '%s', but got: %v", tc.expectedServiceName, err)
				}
			}
		})
	}
}
