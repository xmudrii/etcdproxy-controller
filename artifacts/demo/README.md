# EtcdProxyController demo

This directory contains the manifests for the EtcdProxyController and for the `sample-apiserver`.

Throughout the demo, we'll deploy:

* etcdproxy-controller, with all required resources,
* [sample-apiserver](https://github.com/kubernetes/sample-apiserver)

## Requirements

On your local machine, you need:

* [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

On your Kubernetes cluster, you need:

* [`openshift/service-serving-cert-signer`](https://github.com/openshift/service-serving-cert-signer). The demo manifests uses the `service-serving-cert-signer` to generate and handle certificates renewal for `sample-apiserver`.

You can install `openshift/service-serving-cert-signer` by deploying the following two manifests:

```bash
# Deploy needed RBAC roles.
kubectl auth reconcile -f https://raw.githubusercontent.com/openshift/service-serving-cert-signer/master/install/serving-cert-signer/install-rbac.yaml

# Deploy the service-serving-cert-signer controller and prerequisites.
kubectl create -f https://raw.githubusercontent.com/openshift/service-serving-cert-signer/master/install/serving-cert-signer/install.yaml
```

While `service-serving-cert-signer` is a project by the OpenShift team, it doesn't require OpenShift and works on bare Kubernetes cluster.

* etcd v3.2+ cluster with the appropriate client TLS certificates deployed in the EtcdProxyController namespace. 

If you need to deploy a new cluster, you can use the manifest from the `etcd` subdirectory to deploy single-node etcd v3.2.18 cluster. The manifest also includes self-generated, static, TLS certificates.
```bash
# Deploy etcd cluster.
kubectl create -f etcd/01-etcd-deployment.yaml
```

To use the etcd with the EtcdProxyController, you need to deploy the client certificate for etcd in the controller namespace once the EtcdProxyController is deployed. The client certificates can be found in the `etcd/02-etcd-client-certs.yaml` manifest. 

## Deploying EtcdProxyController

The EtcdProxyController can be deployed using the `etcdproxy` manifest from this directory.

```bash
kubectl create -f 01-etcdproxy-controller.yaml

```

At this point, you have controller deployed, but before you can use it, you need to deploy the client TLS certificates for etcd.

### Deploying the client TLS certificates for etcd

Once the controller is deployed, before deploying the EtcdStorage resources, you need to provide the trust CA and client certificates for the core etcd to the EtcdProxyController, so etcd-proxy pods can access the etcd cluster.

The trust CA is provided by putting it in a ConfigMap in the controller (by default kube-apiserver-storage) namespace. The ConfigMap is by default called etcd-coreserving-ca, but can be configured using the --etcd-core-ca-configmap flag.

The client certificate/key pair is provided to the controller as TLS Secret in the controller namespace, where tls.crt is a client certificate and tls.key is a client key. The Secret is by default called etcd-coreserving-cert, but can be configured using the --etcd-core-ca-secret flag.

When using the etcd cluster deployed using the sample manifest, you can use the following manifest to deploy the client certificates:

```bash
# Deploy the etcd client TLS certiicates.
kubectl create -f etcd/02-etcd-client-certs.yaml
```

## Deploying the API server.

To demonstrate how EtcdProxyController works, we'll deploy the `sample-apiserver`.

```bash
# Create API server Namespace, and ConfigMap and Secret for storing etcd certificates.
kubectl create -f 02-apiserver-etcd-credentials.yaml

# Create RBAC roles.
kubectl auth reconcile -f 03-apiserver-rbac.yaml

# Deploy the sample-apiserver.
kubectl create -f 04-apiserver-deployment.yaml
```

### Testing the API server

To test is API server working as expected, we can deploy a Flunder resource.

When creating the Flunder resource, the API server communicates with the etcd server, so we're sure everything works as expected.

```bash
kubectl create -f 05-apiserver-flunder.yaml
```