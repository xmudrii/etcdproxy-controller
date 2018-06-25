#!/usr/bin/env bash

# The script is supposed to be ran from the project root directory.
SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..

# This deploys the EtcdProxyController using the default kubeconfig file and current context.
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/00-etcdproxy-controller.yaml

# Run Go tests.
cd ${SCRIPT_ROOT}/test/e2e
go test -v ./...
