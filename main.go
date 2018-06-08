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

package main

import (
	"flag"
	"time"

	"github.com/golang/glog"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	clientset "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned"
	informers "github.com/xmudrii/etcdproxy-controller/pkg/client/informers/externalversions"
	"github.com/xmudrii/etcdproxy-controller/pkg/controller/etcdproxy"
	"github.com/xmudrii/etcdproxy-controller/pkg/signals"
)

var (
	kubeconfig string

	etcdURL             string
	etcdCAConfigMapName string
	etcdCertSecretName  string
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	etcdproxyClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	controllerNamespace := etcdproxy.GetControllerNamespace()

	kubeInformersNamespaced := kubeinformers.NewFilteredSharedInformerFactory(kubeClient, 10*time.Minute, controllerNamespace, nil)
	etcdproxyInformers := informers.NewSharedInformerFactory(etcdproxyClient, 10*time.Minute)

	etcdConnectionInfo := &etcdproxy.EtcdConnectionInfo{
		EtcdURL:             etcdURL,
		EtcdCAConfigMapName: etcdCAConfigMapName,
		EtcdCertSecretName:  etcdCertSecretName,
	}

	controller := etcdproxy.NewEtcdProxyController(kubeClient, etcdproxyClient,
		kubeInformersNamespaced.Apps().V1().ReplicaSets(),
		kubeInformersNamespaced.Core().V1().Services(),
		etcdproxyInformers.Etcd().V1alpha1().EtcdStorages(),
		etcdConnectionInfo,
		controllerNamespace)

	go kubeInformersNamespaced.Start(stopCh)
	go etcdproxyInformers.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")

	flag.StringVar(&etcdURL, "etcd-core-url", "", "The address of the core etcd server. Required.")
	flag.StringVar(&etcdCAConfigMapName, "etcd-ca-name", "etcd-coreserving-ca", "The name of the ConfigMap where CA is stored.")
	flag.StringVar(&etcdCertSecretName, "etcd-cert-name", "etcd-coreserving-cert", "The name of the Secret where client certificates are stored.")
}
