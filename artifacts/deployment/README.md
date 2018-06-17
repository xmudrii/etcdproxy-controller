# Deploying `EtcdProxy` Controller

This directory contains manifest for deploying the `EtcdProxy` Controller in the cluster. The manifest deploys the following resources:

* **namespace/kube-apiserver-storage** - namespace for the EtcdProxyController to lives in,

* **sa/etcdproxy-controller-sa** - ServiceAccount for managing EtcdStorage objects, ReplicaSets and Services, Secrets and ConfigMaps,
* **sa/etcdproxy-sa** - ServiceAccount used by EtcdProxy Pods. Does not have any permissions,

* **clusterrole/etcdproxy-crd-clusterrole** - ClusterRole for managing EtcdStorages.
* **clusterrolebinding/etcdproxy-crd-clusterrolebinding** - Binds **clusterrole/etcdproxy-crd-clusterrole** to **serviceaccount/etcdproxy-controller-sa**.

* **role/etcdproxy-controller-role** - Role for managing ReplicaSets, Services, ConfigMap and Secrets in the **kube-apiserver-storage** namespace.
* **rolebinding/etcdproxy-controller-rolebinding** - Binds **role/etcdproxy-controller-role** to **serviceaccount/etcdproxy-controller-sa**.

* **customresourcedefinition/etcdstorages.etcd.xmudrii.com** - CRD defining the EtcdStorage type for managing etcd proxies,
* **deployment/etcdproxy-controller-deployment** - Controller Deployment.

## Deploying the core `etcd`

The deployment manifest assumes you have the core `etcd` deployed and exposed on `https://etcd-svc-1.etcd.svc:2379`.
The URL can be changed by modifying the `--etcd-core-url` flag in the `00-etcdproxy-controller.yaml` file.

There is an example `etcd` deploymend manifest located in the [`artifacts/etcd`](https://github.com/xmudrii/etcdproxy-controller/tree/master/artifacts/etcd) directory.

## Deploying certificates

Before creating the EtcdStorage instances, it's required to create the ConfigMap and Secret containing the core etcd CA certifiacte, and client certificate and key.

The CA certificate is deployed in the ConfigMap called `etcd-coreserving-ca`, in the controller namespace. The ConfigMap name can be changed by adding the `--etcd-core-ca-configmap` flag to controller command in the `00-etcdproxy-controller.yaml` file.

The client certificate and key are both deployed in the generic Secret called `etcd-coreserving-cert`. The Secret name can be changed by adding the `--etcd-core-certs-secret` flag to controller command in the `00-etcdproxy-controller.yaml` file.

Deploying an EtcdStorage resoruce without the requrired ConfigMap and Secret in place causes the EtcdProxy pod to hang in the `Creating` condition.

## Creating the proxied `etcd`

To create the etcd instance for the aggregated API server, deploy the `EtcdStorage` manifest. Once the manifest is deployed, the controller creates and exposes the etcd proxy running against the `etcd` namespace named same as the `EtcdStorage` instance.

The proxied `etcd` is exposed on `http://etcd-<etcdstorage-name>.<controller-namespace>.svc:2379`.

## Deploying the `sample-apiserver`

The [`sample-apiserver`](https://github.com/kubernetes/sample-apiserver) can be used to test the EtcdProxy Controller, by pointing the API server to use the proxied etcd. The API server can be deployed using the `01-apiserver-prerequisites.yaml` and `03-apiserver-deployment.yaml` manifests.

The `01-apiserver-prerequisites.yaml` manifest deploys resources needed for running the API server (Namespace, APIService, RoleBindings, Service). It must be deployed before the `03-apiserver-deployment.yaml` manifest, otherwise the deployment will fail.

The `03-apiserver-deployment.yaml` manifest creates the `EtcdStorage` instance named `kube-sample-apiserver`, which represents the `etcd` instance for the API server. Once deployed, the `etcd` is available on `http://etcd-kube-sample-apiserver.kube-apiserver-storage.svc:2379`.

The manifest deploys the `sample-apiserver` from the [`xmudrii/kube-sample-apiserver` image](https://hub.docker.com/r/xmudrii/kube-sample-apiserver/), which is using the Alpine 3.6 image and is based on the `sample-apiserver` from the [commit `4618274fbc9476e7f1a6a8771962f1eee6a83047`](https://github.com/kubernetes/sample-apiserver/commit/4618274fbc9476e7f1a6a8771962f1eee6a83047).