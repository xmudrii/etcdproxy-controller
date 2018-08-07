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

// TestDeployEtcdStorage deploys EtcdStorage resource and then tests are all relevant resources created and running.
func TestDeployEtcdStorage(t *testing.T) {
	// Initialize clients.
	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Fatal(err)
	}

	client := kubernetes.NewForConfigOrDie(cfg)
	etcdproxyClient := clientset.NewForConfigOrDie(cfg)

	tests := []struct {
		name                   string
		etcdStorage            *v1alpha1.EtcdStorage
		expectedDeploymentName string
		expectedReplicas       int32
		expectedServiceName    string
	}{
		{
			name: "test simple etcdstorage creation",
			etcdStorage: &v1alpha1.EtcdStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name: "es-test-1",
				},
				Spec: v1alpha1.EtdcStorageSpec{
					CACertConfigMaps: []v1alpha1.CABundleDestination{
						{
							Name:      "etcd-serving-ca",
							Namespace: "k8s-sample-apiserver",
						},
					},
					ClientCertSecrets: []v1alpha1.ClientCertificateDestination{
						{
							Name:      "etcd-client-cert",
							Namespace: "k8s-sample-apiserver",
						},
					},
				},
			},
			expectedDeploymentName: "etcd-es-test-1",
			expectedReplicas:       int32(3),
			expectedServiceName:    "etcd-es-test-1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create an EtcdStorage instance.
			es, err := etcdproxyClient.EtcdV1alpha1().EtcdStorages().Create(tc.etcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			// It takes short amount time for the Deployment and Service to be created, so we need to poll.
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
				err = etcdproxyClient.EtcdV1alpha1().EtcdStorages().Delete(es.Name, &metav1.DeleteOptions{})
				if err != nil {
					t.Logf("etcdstorage cleanup failed: %v", err)
				}
				t.Fatal(err)
			}

			// We currently have only one condition, so we're making sure that one is set.
			for _, cond := range es.Status.Conditions {
				if cond.Type != "Deployed" {
					err = etcdproxyClient.EtcdV1alpha1().EtcdStorages().Delete(es.Name, &metav1.DeleteOptions{})
					if err != nil {
						t.Logf("etcdstorage cleanup failed: %v", err)
					}
					t.Fatalf("expected 'Deployed' condition, but got: %s", cond.Type)
				}
				if cond.Status != v1alpha1.ConditionTrue {
					err = etcdproxyClient.EtcdV1alpha1().EtcdStorages().Delete(es.Name, &metav1.DeleteOptions{})
					if err != nil {
						t.Logf("etcdstorage cleanup failed: %v", err)
					}
					t.Fatalf("expected condition 'Deployed' true, but got: %v", cond.Status)
				}
			}

			// Check is the Deployment created and wait for pods to become ready.
			err = wait.Poll(500*time.Millisecond, wait.ForeverTestTimeout, func() (bool, error) {
				ds, err := client.AppsV1().Deployments("kube-apiserver-storage").Get(tc.expectedDeploymentName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if ds.Status.ReadyReplicas != tc.expectedReplicas {
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				err = etcdproxyClient.EtcdV1alpha1().EtcdStorages().Delete(es.Name, &metav1.DeleteOptions{})
				if err != nil {
					t.Logf("etcdstorage cleanup failed: %v", err)
				}
				t.Fatal(err)
			}

			// Check is Service created.
			_, err = client.CoreV1().Services("kube-apiserver-storage").Get(tc.expectedServiceName, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			err = etcdproxyClient.EtcdV1alpha1().EtcdStorages().Delete(es.Name, &metav1.DeleteOptions{})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
