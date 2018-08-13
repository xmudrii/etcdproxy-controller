# etcdproxy-controller

[![Build Status](https://travis-ci.org/xmudrii/etcdproxy-controller.svg?branch=master)](https://travis-ci.org/xmudrii/etcdproxy-controller) [![GoDoc](https://godoc.org/github.com/xmudrii/etcdproxy-controller?status.svg)](https://godoc.org/github.com/xmudrii/etcdproxy-controller) [![Go Report Card](https://goreportcard.com/badge/github.com/xmudrii/etcdproxy-controller)](https://goreportcard.com/report/github.com/xmudrii/etcdproxy-controller) 

Implements https://groups.google.com/forum/#!msg/kubernetes-sig-api-machinery/rHEoQ8cgYwk/iglsNeBwCgAJ

## Google Summer of Code

This project is result of my Google Summer of Code project: [**Storage API for Aggregated API Servers**](https://summerofcode.withgoogle.com/projects/#6400208972283904).

Project outline:

> Kubernetes offers two ways to extend the core API, by using the CustomResourceDefinitons or by setting up an aggregated API server. This ensures users don’t need to modify the core API in order to add the features needed for their workflow, which later ensures the more stable and secure core API.
> 
> One missing part is how to efficiently store data used by aggregated API servers. This project implements a Storage API, with a main goal to share the cluster’s main etcd server with the Aggregated API Servers, allowing it to use cluster’s main etcd just like it would use it’s own etcd server.

Student: Marko Mudrinić  
Mentors: [David Eads](https://github.com/deads2k), [Dr. Stefan Schimanski](https://github.com/sttts)  

More details about the project, including links to proposals and progress tracker, can be found in the [`xmudrii/gsoc-2018-meta-k8s`](https://github.com/xmudrii/gsoc-2018-meta-k8s) repository.

## Purpose

This controller implements the `EtcdStorage` type, used to provide etcd storage for the aggregated API servers.

## Compatibility

HEAD of this repo matches versions `1.10` of `k8s.io/apiserver`, `k8s.io/apimachinery`, and `k8s.io/client-go`.

## Running controller

There are two ways to run `EtcdProxyController`:

* in-cluster, which is done by deploying [the deployment manifests, located in the `artifacts/deployment` directory](artifacts/deployment),
* out-of-cluster, which is done by building the controller binary and running it on your local machine.

### Running the etcd cluster

Before running the controller, you need to expose an etcd cluster to the EtcdProxyController.

The EtcdProxyController assumes an etcd cluster is available on the `https://etcd-svc-1.etcd.svc:2379` endpoint. The endpoint URL can be configured by modifying value of the `--etcd-core-url` flag in the [`00-etcdproxy-controller.yaml` manifest](artifacts/deployment/00-etcdproxy-controller.yaml), or by providing the flag when running out-of-cluster.

There's an example manifest for deploying a single-node etcd cluster along with deploying instructions in the [`artifacts/etcd` directory](artifacts/etcd).

### Deploying the EtcdProxyController in-cluster

To run the controller in-cluster, deploy the deployment manifest from the `artifcats/deployment` directory, using `kubectl`:
```
kubectl create -f artifacts/deployment/00-etcdproxy-controller.yaml
```

This manifest creates namespace for the EtcdProxyController, ServiceAccounts for controller and etcd-proxy pods, RBAC roles for managing all resources used by the controller, EtcdStorage CRD and EtcdProxyController Deployment.

The controller is deployed from the latest Docker Hub image, which can be found in the [`xmudrii/etcdproxy-controller` Docker Hub repository](https://hub.docker.com/r/xmudrii/etcdproxy-controller/).

More details about the deployment manifest can be found in the [README file in the `artifacts/deployment` directory](artifacts/deployment/README.md).

### Running out-of-cluster

Running out-of-cluster is useful when developing the controller and you want to test the latest changes.

Before running the controller out-of-cluster you need to create the etcd proxy namespace and the EtcdStorage CRD.

The etcd proxy namespace is by default called `kube-apiserver-storage` (can be configured using `--namespace` flag),
and we can create it using `kubectl`:
```
kubectl create namespace kube-apiserver-storage
```

The EtcdStorage CRD can be deployed using the manifest located in [`artifacts/etcdstorage` directory](artifacts/etcdstorage):
```
kubectl create -f artifcats/etcdstorage/crd.yaml
```

To build the controller, you need the [Go toolchain installed and configured](https://golang.org/doc/install).

You can build the controller using the `compile` Make target, which compiles the controller and creates a binary in the `./bin` directory:
```
make compile
```

To run the controller, you need to provide it a path to `kubeconfig` and the URL of the core etcd:
```
./bin/etcdproxy-controller --kubeconfig ~/.kube/config --etcd-core-url https://etcd-svc-1.etcd.svc:2379
```

### Providing the core etcd client certificates to the controller

Once the controller is deployed, before deploying the EtcdStorage resources, you need to provide the trust CA and client certificates for the core etcd to the EtcdProxyController, so etcd-proxy pods can access the etcd cluster.

The trust CA is provided by putting it in a ConfigMap in the controller (by default `kube-apiserver-storage`) namespace. The ConfigMap is by default called `etcd-coreserving-ca`, but can be configured using the `--etcd-core-ca-configmap` flag.

The client certificate/key pair is provided to the controller as TLS Secret in the controller namespace, where `tls.crt` is a client certificate and `tls.key` is a client key. The Secret is by default called `etcd-coreserving-cert`, but can be configured using the `--etcd-core-ca-secret` flag.

The ConfigMap and Secret can be created using the following `kubectl` commands:
```
kubectl create configmap etcd-coreserving-ca --from-file=ca.crt -n kube-apiserver-storage
kubectl create secret tls etcd-coreserving-cert --from-file=tls.crt --from-file=tls.key -n kube-apiserver-storage
```

When deploying the core etcd using the example manifest, you can deploy the trust CA and client certificate/key pair using the `etcd-client-certs.yaml` manifest. [The README file in the `artifacts/etcd` directory](artifacts/etcd) contains more details about deploying the etcd and etcd client certificates.

## Creating etcd instances for aggregated API servers

To create an etcd instance for your aggregated API server, you need to deploy an `EtcdStorage` resource.

The sample manifest is located in the `artifacts/etcdstorage` directory, and you can deploy it with `kubectl`, such as:
```
kubectl create -f artifacts/etcdstorage/example-etcdstorage.yaml
```

Once the `EtcdStorage` is deployed, the controller creates a Deployment for EtcdProxy pods, and a Service to expose the pods.

Then, you can use the etcd for your aggregated API server, over the URL such as `http://etcd-<name-of-etcdstorage-object>.kube-apiserver-storage.svc:2379`.

In case of the sample manifest, etcd is available on `http://etcd-etcd-name.kube-apiserver-storage.svc:2379`.

You can check what resources are created in the controller namespace with the following `kubectl` command:
```
kubectl get all -n kube-apiserver-storage
```

## etcd-proxy certificates

The EtcdProxyController handles certificates generation, renewal and rotation for etcd-proxy.

When you create an EtcdStorage resource, the controller:

* Creates client CA certificate and server certificate/key pair. Both are stored in the controller namespace and used by etcd-proxy pods.
* Creates serving CA certificate and client certificate/key pair. Both are stored in the API server namespace and used by the API server. 

The serving CA certificate is stored in a ConfigMap and the client certificate/key pair is stored in a Secret, both in API server namespace. The API server operator must create the ConfigMap and Secret, give the EtcdProxyController ServiceAccount the `GET`, `UPDATE` and `PATCH` permissions on the ConfigMap and Secret, and provide names of the ConfigMap and Secret in the EtcdStorage Spec, such as:

```yaml
...
spec:
  caCertConfigMap:
  - name: etcd-serving-ca
    namespace: k8s-sample-apiserver
  clientCertSecret:
  - name: etcd-client-cert
    namespace: k8s-sample-apiserver
...
```

Beside providing destination ConfigMap and Secret, the API server operator have to provide the certificate validity for each certificate type: CA certificate, Serving certificate, and Client certificate.

This is done by setting appropriate keys in the EtcdStorage Spec:
```yaml
spec:
  ...
  signingCertificateValidity: 730h # defines for how long the signing certificate is valid.
  servingCertificateValidity: 730h # defines for how long the serving certificate/key pair is valid.
  clientCertificateValidity:  730h # defines for how long the client certificate/key pair is valid.
```

It's recommended for value to be longer than 10 minutes.