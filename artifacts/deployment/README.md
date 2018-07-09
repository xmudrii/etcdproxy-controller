# Deploying `EtcdProxy` Controller

This directory contains the `00-etcdproxy-controller.yaml` manifest, used for deploying the `EtcdProxy` Controller in the cluster. The manifest deploys the following resources:

* **namespace/kube-apiserver-storage** - namespace for the EtcdProxyController to lives in,

* **sa/etcdproxy-controller-sa** - ServiceAccount for managing EtcdStorage objects, ReplicaSets and Services, Secrets and ConfigMaps,
* **sa/etcdproxy-sa** - ServiceAccount used by EtcdProxy Pods. Does not have any permissions,

* **clusterrole/etcdproxy-crd-clusterrole** - ClusterRole for managing EtcdStorages.
* **clusterrolebinding/etcdproxy-crd-clusterrolebinding** - Binds **clusterrole/etcdproxy-crd-clusterrole** to **serviceaccount/etcdproxy-controller-sa**.

* **role/etcdproxy-controller-role** - Role for managing ReplicaSets, Services, ConfigMap and Secrets in the **kube-apiserver-storage** namespace.
* **rolebinding/etcdproxy-controller-rolebinding** - Binds **role/etcdproxy-controller-role** to **serviceaccount/etcdproxy-controller-sa**.

* **customresourcedefinition/etcdstorages.etcd.xmudrii.com** - CRD defining the EtcdStorage type for managing etcd proxies,
* **deployment/etcdproxy-controller-deployment** - Controller Deployment.

Before deploying the EtcdProxy controller on GKE, it's required to grant your user the ability to create and manage RBAC roles in the cluster,
which can be done by executing the following command:
```
kubectl create clusterrolebinding cluster-admin-binding \
--clusterrole cluster-admin --user <your-email-address>
```
More about the Role-Based Access Control in GKE can be found in [Google Cloud documentation](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control).

## Deploying the core `etcd`

The deployment manifest assumes you have the core `etcd` deployed and exposed on `https://etcd-svc-1.etcd.svc:2379`.
The URL can be changed by modifying the `--etcd-core-url` flag in the `00-etcdproxy-controller.yaml` file.

There is an example `etcd` deploymend manifest located in the [`artifacts/etcd`](../etcd) directory.

## Deploying certificates

Before creating the EtcdStorage instances, it's required to create the ConfigMap and Secret containing the core etcd CA certifiacte, and client certificate and key.

The CA certificate is deployed in the ConfigMap called `etcd-coreserving-ca`, in the controller namespace. The ConfigMap name can be changed by adding the `--etcd-core-ca-configmap` flag to controller command in the `00-etcdproxy-controller.yaml` file.

The client certificate and key are both deployed in the generic Secret called `etcd-coreserving-cert`. The Secret name can be changed by adding the `--etcd-core-certs-secret` flag to controller command in the `00-etcdproxy-controller.yaml` file.

Deploying an EtcdStorage resoruce without the requrired ConfigMap and Secret in place causes the EtcdProxy pod to hang in the `Creating` condition.

## Creating the proxied `etcd`

To create the etcd instance for the aggregated API server, deploy the `EtcdStorage` manifest. Once the manifest is deployed, the controller creates and exposes the etcd proxy running against the `etcd` namespace named same as the `EtcdStorage` instance.

The proxied `etcd` is exposed on `http://etcd-<etcdstorage-name>.<controller-namespace>.svc:2379`.

## Deploying the `sample-apiserver`

The [`sample-apiserver`](https://github.com/kubernetes/sample-apiserver) can be used to test the EtcdProxy Controller, by pointing the API server to use the proxied etcd.

The following manifests are used to deploy the `sample-apiserver`:
* `01-sample-apiserver-prerequisites.yaml` – creates a namespace, Service and Role/RoleBindings needed by the `sample-apiserver`.
* `02-sample-apiserver-certs.yaml` – creates ConfigMap, Secrets and RBAC roles for the proxied etcd certificates.
* `03-sample-apiserver-deployment.yaml` – creates an EtcdStorage resource, which provides the etcd storage for the API server, `sample-apiserver` ReplicaSet, and APIService to register the `sample-apiserver` with the Kubernetes API.

The manifest uses the [`xmudrii/kube-sample-apiserver`](https://hub.docker.com/r/xmudrii/kube-sample-apiserver/) image, which is based on Alpine 3.6 and is built on `sample-apiserver` commit [`4618274fbc9476e7f1a6a8771962f1eee6a83047`](https://github.com/kubernetes/sample-apiserver/commit/4618274fbc9476e7f1a6a8771962f1eee6a83047).