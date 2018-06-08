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

package controller

import (
	"time"

	clientset "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned"
	informers "github.com/xmudrii/etcdproxy-controller/pkg/client/informers/externalversions"
	"github.com/xmudrii/etcdproxy-controller/pkg/controller/etcdproxy"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

// EtcdConnectionInfo type is used to wire information with etcdproxy controller and CLI.
type EtcdProxyControllerOptions struct {
	EtcdCoreURL             string
	EtcdCoreCAConfigMapName string
	EtcdCoreCertSecretName  string

	EtcdProxyImage string

	KubeconfigPath string

	ControllerNamespace string
}

func NewCommandEtcdProxyControllerStart(stopCh <-chan struct{}) *cobra.Command {
	o := &EtcdProxyControllerOptions{}

	cmd := &cobra.Command{
		Short: "Start EtcdProxyController",
		Long:  "Start EtcdProxyController",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.RunController(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&o.EtcdCoreURL, "core-etcd-url", "", "The address of the core etcd server.")
	cmd.MarkFlagRequired("core-etcd-url")

	flags.StringVar(&o.KubeconfigPath, "kubeconfig", "", "Path to kubeconfig (required only if running out-cluster).")

	flags.StringVar(&o.EtcdCoreCAConfigMapName, "core-etcd-ca-configmap-name", "etcd-coreserving-ca", "The name of the ConfigMap where CA is stored.")
	flags.StringVar(&o.EtcdCoreCertSecretName, "core-etcd-cert-secret-name", "etcd-coreserving-cert", "The name of the Secret where client certificates are stored.")
	flags.StringVar(&o.EtcdProxyImage, "proxy-etcd-image", "quay.io/coreos/etcd:v3.2.18", "The image to be used for creating etcd proxy pods.")
	return cmd
}

// initControllerClientSets returns kubernetes clientset and etcdproxy clientset.
func (o EtcdProxyControllerOptions) initControllerClientSets() (*kubernetes.Clientset, *clientset.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", o.KubeconfigPath)
	if err != nil {
		return nil, nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	etcdproxyClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return kubeClient, etcdproxyClient, nil
}

// initControllerInformers returns informer namespaced to controller's namespace and etcdproxy informer.
func (o EtcdProxyControllerOptions) initControllerInformers(kubeClientset *kubernetes.Clientset, etcdproxyClientset *clientset.Clientset,
	defaultResync time.Duration) (kubeinformers.SharedInformerFactory, informers.SharedInformerFactory) {
	kubeInformersNamespaced := kubeinformers.NewFilteredSharedInformerFactory(kubeClientset, defaultResync, o.ControllerNamespace, nil)
	etcdproxyInformers := informers.NewSharedInformerFactory(etcdproxyClientset, defaultResync)

	return kubeInformersNamespaced, etcdproxyInformers
}

func (o *EtcdProxyControllerOptions) RunController(stopCh <-chan struct{}) error {
	o.ControllerNamespace = etcdproxy.ControllerNamespace()

	kubeClient, etcdproxyClient, err := o.initControllerClientSets()
	if err != nil {
		return err
	}

	etcdConnectionInfo := &etcdproxy.EtcdConnectionInfo{
		EtcdCoreURL:             o.EtcdCoreURL,
		EtcdCoreCAConfigMapName: o.EtcdCoreCAConfigMapName,
		EtcdCoreCertSecretName:  o.EtcdCoreCertSecretName,

		EtcdProxyImage: o.EtcdProxyImage,
	}

	kubeInformersNamespaced, etcdproxyInformers := o.initControllerInformers(kubeClient, etcdproxyClient, 10*time.Minute)

	controller := etcdproxy.NewEtcdProxyController(kubeClient, etcdproxyClient,
		kubeInformersNamespaced.Apps().V1().ReplicaSets(),
		kubeInformersNamespaced.Core().V1().Services(),
		etcdproxyInformers.Etcd().V1alpha1().EtcdStorages(),
		etcdConnectionInfo,
		o.ControllerNamespace)

	go kubeInformersNamespaced.Start(stopCh)
	go etcdproxyInformers.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		return err
	}

	return nil
}
