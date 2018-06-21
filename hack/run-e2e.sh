#!/usr/bin/env bash

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..

kubectl create -f ${SCRIPT_ROOT}/artifacts/deployment/00-