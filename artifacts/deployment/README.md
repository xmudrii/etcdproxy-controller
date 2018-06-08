# `EtcdProxyController` manifest

This directory contains manifest for deploying the `EtcdProxyController` in the cluster. It creates the following resources:
* **namespace/kube-apiserver-storage** - namespace where controller and resources managed by controller are deployed.
* **serviceaccount/etcdproxy-controller-sa** - ServiceAccount used by Controller for managing EtcdStorage objects, ReplicaSets and Services.
* **serviceaccount/etcdproxy-sa** - ServiceAccount used by EtcdProxy Pods.
* **clusterrole/etcdproxy-crd-clusterrole** - ClusterRole allowing **etcdproxy-controller-sa** ServiceAccount to update EtcdStorage status.
* **role/etcdproxy-controller-role** - Role allowing **etcdproxy-controller-sa** ServiceAccount to manage ReplicaSets, Services, ConfigMap and Secrets in the **kube-apiserver-storage** namespace.
* **clusterrolebinding/etcdproxy-crd-clusterrolebinding** - Binds **clusterrole/etcdproxy-crd-clusterrole** to **serviceaccount/etcdproxy-controller-sa**.
* **rolebinding/etcdproxy-controller-rolebinding** - Binds **role/etcdproxy-controller-role** to **serviceaccount/etcdproxy-controller-sa**.
* **customresourcedefinition/etcdstorages.etcd.xmudrii.com** - EtcdStorage CRD used to manage etcd proxies.
* **deployment/etcdproxy-controller-deployment** - Deployment for the `EtcdProxyController`.

## RBAC Rules

The **clusterrole/etcdproxy-crd-clusterrole** allows controller to get the EtcdStorage instances and update to the Status.
The role is supposed to be bound to the Controller's ServiceAccount.

The **role/etcdproxy-controller-role** ensures Controller can manage resources used by the controller (ReplicaSets, Services, ConfigMaps, Secrets). The role is supposed to be bound to the Controller's Service Account.

## Discovering the core `etcd`

The deployment manifest assumes:
* The core `etcd` is available on the `https://etcd-svc-1.etcd.svc:2379` URL. To modify it, change the command in the Deployment object (`-etcd-core-url` flag).
* The CA certificate for the core etcd is stored in the ConfigMap called `etcd-coreserving-ca`, in the controller's namespace. This can be configured using the `-etcd-core-ca-configmap` flag which takes the name of the ConfigMap, in the controller's namespace.
* The client certificate and key for the core etcd are stored in the Secret called `etcd-coreserving-cert`, in the controller's namespace. This can be configured using the `-etcd-core-certs-secret` flag which takes the name of the Secret, in the controller's namespace.