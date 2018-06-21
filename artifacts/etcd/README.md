# Core etcd

The `etcd.yaml` manifest can be used to deploy the core etcd in the `etcd` namespace (created by the `etcd.yaml` manifest), running on the port `:2379`, and exposed on `https://etcd-svc-1.etcd.svc:2379`. The manifest deploys the CA certificate in the `etcd-coreserving-ca` ConfigMap, and the server certificate and key in the `etcd-coreserver-cert` Secret.

For the improved security, it's recommended to create a new pair of certificates and to replace existing ones in the `etcd.yaml` manifest. You generate certificates using [`cfssl`](https://github.com/cloudflare/cfssl). You can learn how to generate certificates using `cfssl` in the [Generate self-signed certificates](https://coreos.com/os/docs/latest/generate-self-signed-certificates.html) portion of the CoreOS documentation.

## Client certificates

The client certificates for accessing the core `etcd` can be deployed using the `etcd-client-certs.yaml` manifest.

The client certificates and key, and CA certificate, are needed by the EtcdProxy Controller, for the `etcd` proxy pods to access the core etcd.

The manifest deploys the `etcd-coreserving-ca` ConfigMap with the CA certificate, and the `etcd-coreserving-cert` Secret with the client certificate and key, both in the controller namespace (`kube-apiserver-storage`).