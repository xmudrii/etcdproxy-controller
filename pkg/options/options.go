package options

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/xmudrii/etcdproxy-controller/pkg/controller/etcdproxy"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/clientcmd"
)

// CoreEtcdOptions type is used to wire the core etcd information used by controller to create ReplicaSets.
type CoreEtcdOptions struct {
	// URLs contains the core etcd addresses.
	URLs []string

	// CAConfigMapName is the name of the ConfigMap in the controller namespace where CA certificates for
	// the core etcd are stored.
	CAConfigMapName string

	// CertSecretName is the name of the Secret in the controller namespace where Client certificate and key for
	// the core etcd are stored.
	CertSecretName string
}

// EtcdProxyControllerOptions type is used to pass information from cli to the controller.
type EtcdProxyControllerOptions struct {
	// CoreEtcd contains information needed to wire up ReplicaSets and the core etcd.
	CoreEtcd *CoreEtcdOptions

	// ControllerNamespace is name of namespace where controller is deployed.
	ControllerNamespace string

	// KubeconfigPath is used to obtain path to kubeconfig, used to create kubeclients.
	KubeconfigPath string

	// ProxyImage is name of the etcd image to be used for etcd-proxy ReplicaSet creation.
	ProxyImage string
}

// NewCoreEtcdOptions returns CoreEtcdOptions struct filled with default values.
func NewCoreEtcdOptions() *CoreEtcdOptions {
	return &CoreEtcdOptions{
		URLs:            []string{},
		CAConfigMapName: "etcd-coreserving-ca",
		CertSecretName:  "etcd-coreserving-cert",
	}
}

// NewEtcdProxyControllerOptions returns EtcdProxyControllerOptions struct filled with default values.
func NewEtcdProxyControllerOptions() *EtcdProxyControllerOptions {
	return &EtcdProxyControllerOptions{
		CoreEtcd:            NewCoreEtcdOptions(),
		ControllerNamespace: "kube-apiserver-storage",
		KubeconfigPath:      "",
		ProxyImage:          "quay.io/coreos/etcd:v3.2.24",
	}
}

// AddFlags adds flags to the root command.
func (e *EtcdProxyControllerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVarP(&e.CoreEtcd.URLs, "etcd-core-url", "u", e.CoreEtcd.URLs, "The address of the core etcd server.")
	fs.StringVar(&e.CoreEtcd.CAConfigMapName, "etcd-core-ca-configmap", e.CoreEtcd.CAConfigMapName, "The name of the ConfigMap where CA is stored.")
	fs.StringVar(&e.CoreEtcd.CertSecretName, "etcd-core-certs-secret", e.CoreEtcd.CertSecretName, "The name of the Secret where client certificates are stored.")

	fs.StringVarP(&e.ControllerNamespace, "namespace", "n", e.ControllerNamespace, "Name of the namespace where controller is deployed.")
	fs.StringVarP(&e.KubeconfigPath, "kubeconfig", "k", e.KubeconfigPath, "Path to kubeconfig (required only if running out-of-cluster).")
	fs.StringVar(&e.ProxyImage, "etcd-proxy-image", e.ProxyImage, "The image to be used for creating etcd proxy pods.")
}

// ApplyTo applies provided Options struct to the provided Config struct.
func (e *EtcdProxyControllerOptions) ApplyTo(c *etcdproxy.EtcdProxyControllerConfig) error {
	var err error

	c.CoreEtcd = &etcdproxy.CoreEtcdConfig{}
	c.CoreEtcd.URLs = append([]string{}, e.CoreEtcd.URLs...)
	c.CoreEtcd.CAConfigMapName = e.CoreEtcd.CAConfigMapName
	c.CoreEtcd.CertSecretName = e.CoreEtcd.CertSecretName

	c.ControllerNamespace = e.ControllerNamespace
	c.ProxyImage = e.ProxyImage

	c.Kubeconfig, err = clientcmd.BuildConfigFromFlags("", e.KubeconfigPath)
	if err != nil {
		return err
	}

	return nil
}

// Validate verifies are EtcdProxyControllerOptions and CoreEtcdOptions struct correctly populated.
func (e *EtcdProxyControllerOptions) Validate() error {
	errors := []error{}

	errors = append(errors, e.CoreEtcd.Validate())

	if e.ControllerNamespace == "" {
		errors = append(errors, fmt.Errorf("controller namespace name empty"))
	}

	if e.ProxyImage == "" {
		errors = append(errors, fmt.Errorf("etcd proxy image name empty"))
	}

	return utilerrors.NewAggregate(errors)
}

// Validate verifies is CoreEtcdOptions struct correctly populated.
func (c *CoreEtcdOptions) Validate() error {
	errors := []error{}

	if len(c.URLs) == 0 {
		errors = append(errors, fmt.Errorf("core etcd url empty"))
	}

	if c.CAConfigMapName == "" {
		errors = append(errors, fmt.Errorf("core etcd ca certificates configmap name empty"))
	}

	if c.CertSecretName == "" {
		errors = append(errors, fmt.Errorf("core etcd certificates secret name empty"))
	}

	return utilerrors.NewAggregate(errors)
}
