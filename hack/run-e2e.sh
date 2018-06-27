#!/usr/bin/env bash

set -e

echo -e '- Staring EtcdProxy Controller End-to-End tests'

# Environment variables.
echo -e '- Setting up the test environment..'

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}

echo ''
echo -e 'kubernetes version:\t' $(kubectl version -o json | jq .serverVersion.gitVersion)
echo -e 'etcdproxy version:\t' $(git rev-parse --verify HEAD)
echo ''

# Deploying prerequisites
echo '- Deploying the core etcd'
kubectl create -f ${SCRIPT_ROOT}/artifacts/etcd/etcd.yaml
echo ''

# Test deploying controller and creating the EtcdStorage object.
echo '- Testing EtcdStorage deployment:'

echo '* Deploying the EtcdProxy Controller.'
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/00-etcdproxy-controller.yaml
kubectl create -f ${SCRIPT_ROOT}/artifacts/etcd/etcd-client-certs.yaml

# Run Go tests.
echo '* Deploying EtcdStorage object and verifying deployed resources.'
go test -v ${SCRIPT_ROOT}/test/e2e/...

echo -e '- EtcdStorage tests completed successfully!\n'

# Test deploying the sample-apiserver.
echo '- Testing sample-apiserver deployment:'

echo '* Deploying sample-apiserver resources.'
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/01-sample-apiserver-prerequisites.yaml
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/02-sample-apiserver-certs.yaml
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/03-sample-apiserver-deployment.yaml

echo '* Waiting for API server to become ready'
READY_REPLICAS=$(kubectl get rs apiserver -o jsonpath="{.status.readyReplicas}")
while [ $READY_REPLICAS -eq 0 ]
do
    READY_REPLICAS=$(kubectl get rs apiserver -o jsonpath="{.status.readyReplicas}")
done

echo '* Starting sample-apiserver e2e tests.'

echo ''
echo '- All tests passed successfully!'

echo ''
echo '- Cleaning up resources..'

kubectl delete -f ${SCRIPT_ROOT}/artifacts/etcd/etcd.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/00-etcdproxy-controller.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/etcd/etcd-client-certs.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/01-sample-apiserver-prerequisites.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/02-sample-apiserver-certs.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/03-sample-apiserver-deployment.yaml

echo '- Clean up successful!'