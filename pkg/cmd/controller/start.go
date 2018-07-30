package controller

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	clientset "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned"
	informers "github.com/xmudrii/etcdproxy-controller/pkg/client/informers/externalversions"
	"github.com/xmudrii/etcdproxy-controller/pkg/controller/etcdproxy"
	"github.com/xmudrii/etcdproxy-controller/pkg/options"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// NewCommandEtcdProxyControllerStart returns controller root command.
func NewCommandEtcdProxyControllerStart(stopCh <-chan struct{}) *cobra.Command {
	o := options.NewEtcdProxyControllerOptions()

	cmd := &cobra.Command{
		Short: "Start EtcdProxyController",
		Long:  "Start EtcdProxyController",
		Run: func(c *cobra.Command, args []string) {
			cfg := &etcdproxy.EtcdProxyControllerConfig{}

			if err := o.Validate(); err != nil {
				glog.Fatal(err)
			}

			if err := o.ApplyTo(cfg); err != nil {
				glog.Fatal(err)
			}

			if err := RunController(cfg, stopCh); err != nil {
				glog.Fatal(err)
			}
		},
	}

	o.AddFlags(cmd.Flags())
	return cmd
}

// RunController runs EtcdProxyController.
func RunController(config *etcdproxy.EtcdProxyControllerConfig, stopCh <-chan struct{}) error {
	controllerNamespace, err := controllerNamespace(config.ControllerNamespace)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(config.Kubeconfig)
	if err != nil {
		return err
	}
	kubeInformersNamespaced := kubeinformers.NewFilteredSharedInformerFactory(kubeClient, 10*time.Minute, controllerNamespace, nil)

	etcdproxyClient, err := clientset.NewForConfig(config.Kubeconfig)
	if err != nil {
		return err
	}
	etcdproxyInformers := informers.NewSharedInformerFactory(etcdproxyClient, 10*time.Minute)

	controller := etcdproxy.NewEtcdProxyController(kubeClient, etcdproxyClient,
		kubeInformersNamespaced.Apps().V1().Deployments(),
		kubeInformersNamespaced.Core().V1().Services(),
		etcdproxyInformers.Etcd().V1alpha1().EtcdStorages(), config)

	go kubeInformersNamespaced.Start(stopCh)
	go etcdproxyInformers.Start(stopCh)

	return controller.Run(2, stopCh)
}

// controllerNamespace returns name of the namespace where controller is located. The namespace name is obtained
// from the "/var/run/secrets/kubernetes.io/serviceaccount/namespace" file. In case it's not possible to obtain it
// from that file, the function resorts to the default name, `kube-apiserver-storage`.
func controllerNamespace(namespace string) (string, error) {
	if namespace != "" {
		return namespace, nil
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	}

	return "", fmt.Errorf("unable to detect controller namespace")
}
