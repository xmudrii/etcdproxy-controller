# etcdproxy-controller

[![Build Status](https://travis-ci.org/xmudrii/etcdproxy-controller.svg?branch=master)](https://travis-ci.org/xmudrii/etcdproxy-controller) [![GoDoc](https://godoc.org/github.com/xmudrii/etcdproxy-controller?status.svg)](https://godoc.org/github.com/xmudrii/etcdproxy-controller) [![Go Report Card](https://goreportcard.com/badge/github.com/xmudrii/etcdproxy-controller)](https://goreportcard.com/report/github.com/xmudrii/etcdproxy-controller) 

Implements https://groups.google.com/forum/#!msg/kubernetes-sig-api-machinery/rHEoQ8cgYwk/iglsNeBwCgAJ

## Purpose

This controller implements the `EtcdStorage` type, used to provide etcd storage for the aggregated API servers.

## Compatibility

HEAD of this repo matches versions `1.10` of `k8s.io/apiserver`, `k8s.io/apimachinery`, and `k8s.io/client-go`.

## Running controller

There are two ways to run `EtcdProxyController`:

* in-cluster, which is done by deploying [the deployment manifest, located in the `artifacts/deployment` directory](artifacts/deployment/00-etcdproxy-controller.yaml),
* out-of-cluster, which is done by building the controller binary and running it on the local machine.

Before running the controller, you need to deploy the etcd cluster. There's an example manifests along with the deploying instructions in the [`artifacts/etcd` directory](artifacts/etcd).

The deployment manifest assumes etcd is available on `https://etcd-svc-1.etcd.svc:2379`. This can be configured by updating the [`00-etcdproxy-controller.yaml` manifest](artifacts/deployment/00-etcdproxy-controller.yaml).

When running out-of-cluster, the etcd URL can be changed using the `etcd-core-url` flag.

### Providing the core etcd client certificates to the controller

When deploying the core etcd using the provided manifests, you can deploy the client certificates using the `etcd-client-certs.yaml` manifest. [The README of `artifacts/etcd`](artifacts/etcd) contains more details about deploying the client certificates.

If you're deploying etcd using another method, the `EtcdProxyController` requires you to provide it the CA certificate (`ca.pem`) as a ConfigMap,
and the client certificate (`client.pem`) and key (`client-key.pem`) as a Secret, in the controller namespace:

* The ConfigMap is by default called `etcd-coreserving-ca` (can be configured using the `--etcd-core-ca-configmap` flag),
* The Secret is by default called `etcd-coreserving-cert` (can be configured using the `--etcd-core-ca-secret` flag).

The ConfigMap and Secret can be created using the following `kubectl` commands:
```
kubectl create configmap etcd-coreserving-ca --from-file=ca.pem -n kube-apiserver-storage
kubectl create secret generic etcd-coreserving-cert --from-file=client.pem --from-file=client-key.pem -n kube-apiserver-storage
```

### Running out-of-cluster

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
Then, you can build the controller using the `compile` Make target, which compiles the controller and creates a binary in the `./bin` directory:
```
make compile
```

To run the controller, you need to provide it a path to `kubeconfig` and the URL of the core etcd:
```
./bin/etcdproxy-controller --kubeconfig ~/.kube/config --etcd-core-url https://etcd-svc-1.etcd.svc:2379
```

## Creating etcd instances for aggregated API servers

To create an etcd instance for your aggregated API server, you need to deploy an `EtcdStorage` manifest.
The sample manifest is located in the `artifacts/etcdstorage` directory, and you can deploy it with `kubectl`, such as:
```
kubectl create -f artifacts/etcdstorage/example-etcdstorage.yaml
```

Once the `EtcdStorage` is deployed, the controller creates the ReplicaSet for EtcdProxy pods, and a Service to expose the pods.

Then, you can use the etcd for your aggregated API server, over the URL such as `http://etcd-<name-of-etcdstorage-object>.kube-apiserver-storage.svc:2379`.
In case of the sample manifest, etcd is available on `http://etcd-etcd-name.kube-apiserver-storage.svc:2379`.

You can check what resources are created in the controller namespace with the following `kubectl` command:
```
kubectl get all -n kube-apiserver-storage
```