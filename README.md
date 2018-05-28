# etcdproxy-controller

[![Build Status](https://travis-ci.org/xmudrii/etcdproxy-controller.svg?branch=master)](https://travis-ci.org/xmudrii/etcdproxy-controller) [![GoDoc](https://godoc.org/github.com/xmudrii/etcdproxy-controller?status.svg)](https://godoc.org/github.com/xmudrii/etcdproxy-controller) [![Go Report Card](https://goreportcard.com/badge/github.com/xmudrii/etcdproxy-controller)](https://goreportcard.com/report/github.com/xmudrii/etcdproxy-controller) 

Implements https://groups.google.com/forum/#!msg/kubernetes-sig-api-machinery/rHEoQ8cgYwk/iglsNeBwCgAJ

## Purpose

This controller implements the `EtcdStorage` type, used to provide etcd storage for aggregated API servers.

## Compatibility

HEAD of this repo matches versions `1.10` of `k8s.io/apiserver`, `k8s.io/apimachinery`, and `k8s.io/client-go`.

## Running

Prerequisite: Since the sample-controller uses apps/v1 deployments, the Kubernetes cluster version should be greater than 1.9.

```
# assumes you have a working kubeconfig, not required if operating in-cluster
$ go run *.go -kubeconfig=$HOME/.kube/config

# create a CustomResourceDefinition
$ kubectl create -f artifacts/examples/etcdstorage-crd.yaml

# create a custom resource of type Foo
$ kubectl create -f artifacts/examples/etcdstorage-cr.yaml

# check deployments created through the custom resource
$ kubectl get deployments
```