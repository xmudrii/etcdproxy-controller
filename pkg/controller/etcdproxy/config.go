package etcdproxy

import (
	restclient "k8s.io/client-go/rest"
)

// EtcdProxyControllerConfig type is used to pass information from cli to the controller.
type EtcdProxyControllerConfig struct {
	// CoreEtcd contains information needed to wire up ReplicaSets and the core etcd.
	CoreEtcd *CoreEtcdConfig

	// ControllerNamespace is name of namespace where controller is deployed.
	ControllerNamespace string

	// Kubeconfig is the cluster configuration.
	Kubeconfig *restclient.Config

	// ProxyImage is name of the etcd image to be used for etcd-proxy ReplicaSet creation.
	ProxyImage string
}

// CoreEtcdConfig type is used to wire the core etcd information used by controller to create ReplicaSets.
type CoreEtcdConfig struct {
	// URLs contains the core etcd addresses.
	URLs []string

	// CAConfigMapName is the name of the ConfigMap in the controller namespace where CA certificates for
	// the core etcd are stored.
	CAConfigMapName string

	// CertSecretName is the name of the Secret in the controller namespace where Client certificate and key for
	// the core etcd are stored.
	CertSecretName string
}
