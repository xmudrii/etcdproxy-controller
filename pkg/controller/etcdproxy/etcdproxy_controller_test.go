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

package etcdproxy

import (
	"testing"

	"github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes/fake"
	rslisters "k8s.io/client-go/listers/apps/v1"
	svclisters "k8s.io/client-go/listers/core/v1"

	etcdclient "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned/fake"
	etcdlisters "github.com/xmudrii/etcdproxy-controller/pkg/client/listers/etcd/v1alpha1"
)

func TestSyncHandler(t *testing.T) {
	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
	}

	tests := []struct {
		name                   string
		startingEtcdStorage    *v1alpha1.EtcdStorage
		etcdProxyOptions       *EtcdProxyOptions
		expectedReplicaSetName string
		expectedServiceName    string
	}{
		{
			name:                "test simple creation",
			startingEtcdStorage: etcdStorage("test-1"),
			etcdProxyOptions: &EtcdProxyOptions{
				CoreEtcd: CoreEtcdOptions{
					URL:             "https://test.etcd.svc:2379",
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "kube-apiserver-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedReplicaSetName: "etcd-rs-test-1",
			expectedServiceName:    "etcd-test-1",
		},
		{
			name:                "test simple creation with non-default namespace",
			startingEtcdStorage: etcdStorage("test-2"),
			etcdProxyOptions: &EtcdProxyOptions{
				CoreEtcd: CoreEtcdOptions{
					URL:             "https://test.etcd.svc:2379",
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "test-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			expectedReplicaSetName: "etcd-rs-test-2",
			expectedServiceName:    "etcd-test-2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
			etcdObjs := []runtime.Object{}
			objs := []runtime.Object{}

			etcdObjs = append(etcdObjs, tc.startingEtcdStorage)
			indexer.Add(tc.startingEtcdStorage)

			etcdstorageClient := etcdclient.NewSimpleClientset(etcdObjs...)
			kubeClient := kubeclient.NewSimpleClientset(objs...)

			c := EtcdProxyController{
				etcdProxyClient:    etcdstorageClient,
				etcdstoragesLister: etcdlisters.NewEtcdStorageLister(indexer),

				kubeclientset:     kubeClient,
				replicasetsLister: rslisters.NewReplicaSetLister(indexer),
				servicesLister:    svclisters.NewServiceLister(indexer),
				recorder:          &record.FakeRecorder{},

				etcdProxyOptions: tc.etcdProxyOptions,
			}
			err := c.syncHandler(tc.startingEtcdStorage.Name)
			if err != nil {
				t.Fatal(err)
			}

			// Check is ReplicaSet created.
			_, err = kubeClient.Apps().ReplicaSets(tc.etcdProxyOptions.ControllerNamespace).Get(tc.expectedReplicaSetName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				t.Fatalf("replicaset not found: %v", err)
			}
			if err != nil {
				t.Fatal(err)
			}

			// Check is Service created.
			_, err = kubeClient.Core().Services(tc.etcdProxyOptions.ControllerNamespace).Get(tc.expectedServiceName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				t.Fatalf("service not found: %v", err)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
