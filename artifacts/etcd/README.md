# Core etcd

The `etcd.yaml` manifest can be used to deploy the core etcd in the `etcd` namespace (created by the manifest), running on the port `:2379`, and exposed on `https://etcd-svc-1.etcd.svc:2379`. The manifest deploys the CA certificate in the `etcd-coreserving-ca` ConfigMap, and the server certificate and key in the `etcd-coreserver-cert` Secret.

For the improved security, it's recommended to create a new pair of certificates and to replace existing ones in the `etcd.yaml` manifest. You generate certificates using [`cfssl`](https://github.com/cloudflare/cfssl). You can learn how to generate certificates using `cfssl` in the [Generate self-signed certificates](https://coreos.com/os/docs/latest/generate-self-signed-certificates.html) portion of the CoreOS documentation.

## Client certificates

The client certificates for accessing the core `etcd` can be deployed using the `etcd-client-certs.yaml` manifest.

The client certificates and key, and CA certificate, are needed by the EtcdProxy Controller, for the `etcd` proxy pods to access the core etcd.

The manifest deploys the `etcd-coreserving-ca` ConfigMap with the CA certificate, and the `etcd-coreserving-cert` Secret with the client certificate and key, both in the controller namespace (`kube-apiserver-storage`).

For improved security, it's recommended to generate new pair of certificates, for example by using the `cfssl` tool.

## Deploying the core etcd

This directory contains manifests for deploying and exposing the core etcd.

The `etcd.yaml` manifest file deploys a single-node etcd cluster and the `etcd-svc-1` Service. It can be deployed using `kubectl` such as:
```
kubectl create -f etcd.yaml
```

The `etcd-client-certs.yaml` manifest deploys the client certificates needed by clients (e.g. EtcdProxyController) to access the core etcd.

The ConfigMap called `etcd-coreserving-ca` contains the CA certificate, and the Secret called `etcd-coreserving-cert`, contains the client certificate and key.
By default, those resources are deployed in the namespace called `kube-apiserver-storage`.

Once `etcd` is deployed, you can use it to access `etcd` over the URL such as `https://etcd-svc-1.etcd.svc:2379`, where `etcd-svc-1` is the name
of the service, and `etcd` is the name of the namespace where the core `etcd` is deployed.