package etcdproxy

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	etcdclient "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned/fake"
	etcdlisters "github.com/xmudrii/etcdproxy-controller/pkg/client/listers/etcd/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes/fake"
	dslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

func TestSyncHandler(t *testing.T) {
	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
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
		}
	}
	etcdStorageNoCerts := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
	}
	configMap := func(name, namespace string) *v1.ConfigMap {
		return &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}
	secret := func(name, namespace string) *v1.Secret {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			// TODO: add a test case when secret is not v1.SecretTypeTLS.
			Type: v1.SecretTypeTLS,
		}
	}

	tests := []struct {
		name                   string
		startingEtcdStorage    *v1alpha1.EtcdStorage
		startingConfigMap      *v1.ConfigMap
		startingSecret         *v1.Secret
		etcdProxyConfig        *EtcdProxyControllerConfig
		expectedDeploymentName string
		expectedServiceName    string
	}{
		{
			name:                "test simple creation",
			startingEtcdStorage: etcdStorage("test-1"),
			startingConfigMap:   configMap("etcd-serving-ca", "k8s-sample-apiserver"),
			startingSecret:      secret("etcd-client-cert", "k8s-sample-apiserver"),
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "kube-apiserver-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedDeploymentName: "etcd-test-1",
			expectedServiceName:    "etcd-test-1",
		},
		{
			name:                "test simple creation with non-default namespace",
			startingEtcdStorage: etcdStorage("test-2"),
			startingConfigMap:   configMap("etcd-serving-ca", "k8s-sample-apiserver"),
			startingSecret:      secret("etcd-client-cert", "k8s-sample-apiserver"),
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "test-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedDeploymentName: "etcd-test-2",
			expectedServiceName:    "etcd-test-2",
		},
		{
			name:                "test simple creation without apiserver configmap and secret provided in spec",
			startingEtcdStorage: etcdStorageNoCerts("test-3"),
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "test-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedDeploymentName: "etcd-test-3",
			expectedServiceName:    "etcd-test-3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
			etcdObjs := []runtime.Object{}
			objs := []runtime.Object{}

			if tc.startingConfigMap != nil {
				objs = append(objs, tc.startingConfigMap)
				indexer.Add(tc.startingConfigMap)
			}
			if tc.startingSecret != nil {
				objs = append(objs, tc.startingSecret)
				indexer.Add(tc.startingSecret)
			}

			etcdObjs = append(etcdObjs, tc.startingEtcdStorage)
			indexer.Add(tc.startingEtcdStorage)

			etcdstorageClient := etcdclient.NewSimpleClientset(etcdObjs...)
			kubeClient := kubeclient.NewSimpleClientset(objs...)

			c := EtcdProxyController{
				etcdProxyClient:    etcdstorageClient,
				etcdstoragesLister: etcdlisters.NewEtcdStorageLister(indexer),

				kubeclientset:     kubeClient,
				deploymentsLister: dslisters.NewDeploymentLister(indexer),
				servicesLister:    corelisters.NewServiceLister(indexer),
				recorder:          &record.FakeRecorder{},

				config: tc.etcdProxyConfig,
			}
			err := c.syncHandler(tc.startingEtcdStorage.Name)
			if err != nil {
				t.Fatal(err)
			}

			// Check is Deployment created.
			_, err = kubeClient.Apps().Deployments(tc.etcdProxyConfig.ControllerNamespace).Get(tc.expectedDeploymentName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				t.Fatalf("deployment not found: %v", err)
			}
			if err != nil {
				t.Fatal(err)
			}

			// Check is Service created.
			_, err = kubeClient.Core().Services(tc.etcdProxyConfig.ControllerNamespace).Get(tc.expectedServiceName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				t.Fatalf("service not found: %v", err)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSyncHandlerFailure(t *testing.T) {
	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
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
		}
	}
	configMap := func(name, namespace string) *v1.ConfigMap {
		return &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}
	secret := func(name, namespace string) *v1.Secret {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}

	tests := []struct {
		name                string
		startingEtcdStorage *v1alpha1.EtcdStorage
		startingConfigMap   *v1.ConfigMap
		startingSecret      *v1.Secret
		etcdProxyConfig     *EtcdProxyControllerConfig
		expectedErrors      []error
	}{
		{
			name:                "test creation without ca cert configmap created",
			startingConfigMap:   nil,
			startingSecret:      secret("etcd-client-cert", "k8s-sample-apiserver"),
			startingEtcdStorage: etcdStorage("test-1"),
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "kube-apiserver-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedErrors: []error{fmt.Errorf("configmaps \"etcd-serving-ca\" not found")},
		},
		{
			name:                "test creation without clinet cert secret created",
			startingEtcdStorage: etcdStorage("test-2"),
			startingConfigMap:   configMap("etcd-serving-ca", "k8s-sample-apiserver"),
			startingSecret:      nil,
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "kube-apiserver-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedErrors: []error{fmt.Errorf("secrets \"etcd-client-cert\" not found")},
		},
		{
			name:                "test creation without ca cert configmap and client cert secret created",
			startingEtcdStorage: etcdStorage("test-3"),
			startingConfigMap:   nil,
			startingSecret:      nil,
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "kube-apiserver-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedErrors: []error{fmt.Errorf("configmaps \"etcd-serving-ca\" not found"),
				fmt.Errorf("secrets \"etcd-client-cert\" not found")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
			etcdObjs := []runtime.Object{}
			objs := []runtime.Object{}

			if tc.startingConfigMap != nil {
				objs = append(objs, tc.startingConfigMap)
				indexer.Add(tc.startingConfigMap)
			}
			if tc.startingSecret != nil {
				objs = append(objs, tc.startingSecret)
				indexer.Add(tc.startingSecret)
			}

			etcdObjs = append(etcdObjs, tc.startingEtcdStorage)
			indexer.Add(tc.startingEtcdStorage)

			etcdstorageClient := etcdclient.NewSimpleClientset(etcdObjs...)
			kubeClient := kubeclient.NewSimpleClientset(objs...)

			c := EtcdProxyController{
				etcdProxyClient:    etcdstorageClient,
				etcdstoragesLister: etcdlisters.NewEtcdStorageLister(indexer),

				kubeclientset:     kubeClient,
				deploymentsLister: dslisters.NewDeploymentLister(indexer),
				servicesLister:    corelisters.NewServiceLister(indexer),
				recorder:          &record.FakeRecorder{},

				config: tc.etcdProxyConfig,
			}
			errs := c.syncHandler(tc.startingEtcdStorage.Name)
			if reflect.DeepEqual(errs, tc.expectedErrors) {
				t.Fatalf("expected error(s): '%v',\nbut got error(s): '%v'", tc.expectedErrors, errs)
			}
		})
	}
}
