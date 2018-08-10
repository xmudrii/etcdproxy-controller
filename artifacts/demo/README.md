# EtcdProxyController demo

This directory contains manifest and scripts for deploying the EtcdProxyController.

In this directory, you'll find manifests for deploying:

* core etcd, located in the `etcd` subdirectory,
* etcdproxy-controller, with all required resources, such as ServiceAccounts and RBAC roles,
* [sample-apiserver](https://github.com/kubernetes/sample-apiserver).

## Requirements

On your local machine, you need:

* kubectl

On your Kubernetes cluster, you need:

* [OpenShift service-serving-cert-signer](https://github.com/openshift/service-serving-cert-signer).  
The demo manifests uses the `openshift/service-serving-cert-signer` to generate and handle certificates renewal and rotation for the sample-apiserver.
You can install `openshift/service-serving-cert-signer` by deploying the following two manifests:
```bash
# Deploy needed RBAC roles.
kubectl auth reconcile -f https://raw.githubusercontent.com/openshift/service-serving-cert-signer/master/install/serving-cert-signer/install-rbac.yaml

# Deploy the service-serving-cert-signer controller and prerequisites.
kubectl create -f https://raw.githubusercontent.com/openshift/service-serving-cert-signer/master/install/serving-cert-signer/install.yaml
```

* etcd cluster with the appropriate client TLS certificates.  
You can use any etcd v3.2+ cluster for the EtcdProxyController. If you need to deploy the new cluster, you can use the manifest
from the `etcd` subdirectory to deploy single-node etcd v3.2.18 cluster. The manifest also includes self-generated, static, TLS certificates.
```bash
# Deploy etcd cluster.
kubectl create -f etcd/01-etcd-deployment.yaml
```
To use the etcd with the EtcdProxyController, you need to deploy the client certificate for etcd in the controller namespace once the EtcdProxyController is deployed.
The client certificates can be found in the `etcd/02-etcd-client-certs.yaml` manifest. 

## Deploying EtcdProxyController

```bash
# Create namespace and ServiceAccounts.
kubectl create -f 01-etcdproxy-namespace.yaml

# Create RBAC roles, RoleBindings and ClusterRoleBindings.
kubectl auth reconcile -f 02-etcdproxy-rbac.yaml

# Deploy the EtcdProxyController.
kubectl create -f 03-etcdproxy-deployment.yaml
```

At this point, you have controller deployed, but before you can use it, you need to deploy the client TLS certificates for etcd.

### Deploying the client TLS certificates for etcd

```bash
# Deploy the etcd client TLS certiicates.
kubectl create -f etcd/02-etcd-client-certs.yaml
```

## Deploying the API server.

To demonstrate how EtcdProxyController works, we'll deploy the `sample-apiserver`.

```bash
# Create API server Namespace, and ConfigMap and Secret for storing etcd certificates.
kubectl create -f 04-apiserver-etcd-credentials.yaml

# Create RBAC roles.
kubectl auth reconcile -f 05-apiserver-rbac.yaml

# Deploy the sample-apiserver.
kubectl create -f 06-apiserver-deployment.yaml
```

### Testing the API server

To test is API server working as expected, we can deploy a Flunder resource.

When creating the Flunder resource, the API server communicates with the etcd server, so we're sure everything works as expected.

```bash
kubectl create -f 08-apiserver-flunder.yaml
```