#!/usr/bin/env bash

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..

IMAGE=xmudrii/etcdproxy-controller
KIND_CLUSTER_NAME="kind-1"
KIND_CONTAINER_NAME="${KIND_CLUSTER_NAME}-control-plane"

build_image() {
    # Switch to the root directory.
    cd $SCRIPT_ROOT
    
    # Create a temporary directory to store generated Docker image.
    TMP_DIR=$(mktemp -d)
    IMAGE_FILE=${TMP_DIR}/etcdproxy-controller.tar.gz

    # Build Docker image.
    docker build -t "${IMAGE}":latest .
    # Export generated Docker image to an archive.
    docker save "${IMAGE}" -o "${IMAGE_FILE}"
    # Copy saved archive into kind's Docker container.
    docker cp "${IMAGE_FILE}" "${KIND_CONTAINER_NAME}":/etcdproxy-controller.tar.gz
    # Import image into kind's Docker daemon to make it accessible to Kubernetes.
    docker exec "${KIND_CONTAINER_NAME}" docker load -i /etcdproxy-controller.tar.gz

    # Cleanup the temporary directory.
    rm -rf "${TMP_DIR}"
}

build_image
