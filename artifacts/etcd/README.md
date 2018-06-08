# Experimental

This directory contains manifest for deploying `etcd`, etcd-gRPC proxy, and for creating Service to expose the etcd-gRPC
proxy. The manifests located in this directory have been used in the experimental phase to experiment and test the proposed setup.

Useful commands:
* `kubectl run my-shell --rm -i --tty --image quay.io/coreos/etcd:v3.2.18 -- ash` - run shell with etcdctl.
