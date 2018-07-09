#!/usr/bin/env bash

set -e

echo -e '- Staring EtcdProxy Controller End-to-End tests'

# Environment variables.
echo -e '- Setting up the test environment..'

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
export KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}

echo ''
echo -e 'kubernetes version:\t' $(kubectl version -o json | jq .serverVersion.gitVersion)
echo -e 'etcdproxy version:\t' $(git rev-parse --verify HEAD)
echo ''

# Tag xmudrii/etcdproxy-controller:latest Docker image.
echo '- Tagging etcdproxy-controller Docker image.'
docker build -t xmudrii/etcdproxy-controller:latest .
echo ''

# Deploying prerequisites
echo '- Deploying the core etcd'
kubectl create -f ${SCRIPT_ROOT}/artifacts/etcd/etcd.yaml
echo ''

echo '- Deploying the EtcdProxy Controller.'
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/00-etcdproxy-controller.yaml
kubectl create -f ${SCRIPT_ROOT}/artifacts/etcd/etcd-client-certs.yaml

echo '- Deploying the sample-apiserver prerequisites.'
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/01-sample-apiserver-prerequisites.yaml
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/02-sample-apiserver-certs.yaml

# Test deploying controller and creating the EtcdStorage object.
echo '- Testing EtcdStorage deployment:'
# Run Go tests.
echo '* Deploying EtcdStorage object and verifying deployed resources.'
GOCACHE=off go test -v ${SCRIPT_ROOT}/test/e2e/...

echo -e '- EtcdStorage tests completed successfully!\n'

# Test deploying the sample-apiserver.
echo '- Testing sample-apiserver deployment:'
echo '* Deploying sample-apiserver.'
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/03-sample-apiserver-deployment.yaml

echo '* Waiting for API server to become ready'
READY_REPLICAS=$(kubectl get rs apiserver -n k8s-sample-apiserver -o jsonpath="{.status.readyReplicas}")
while [ "$READY_REPLICAS" == "0" ]
do
    READY_REPLICAS=$(kubectl get rs apiserver -n k8s-sample-apiserver -o jsonpath="{.status.readyReplicas}")
done
APISERVICE_STATUS=$(kubectl get apiservice v1alpha1.wardle.k8s.io -o jsonpath="{.status.conditions[0].status}")
while [ "$APISERVICE_STATUS" != "True" ]
do
    APISERVICE_STATUS=$(kubectl get apiservice v1alpha1.wardle.k8s.io -o jsonpath="{.status.conditions[0].status}")
done
until kubectl get --raw /apis/wardle.k8s.io/v1alpha1
do
    sleep 0.5
done

echo '* Creating a sample Flunder resource.'
kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/04-sample-apiserver-flunder.yaml

echo ''
echo '- All tests passed successfully!'

echo ''
echo '- Cleaning up resources..'

kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/04-sample-apiserver-flunder.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/03-sample-apiserver-deployment.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/02-sample-apiserver-certs.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/01-sample-apiserver-prerequisites.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/etcd/etcd-client-certs.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/deployment/00-etcdproxy-controller.yaml
kubectl delete -f ${SCRIPT_ROOT}/artifacts/etcd/etcd.yaml

echo '- Clean up successful!'
