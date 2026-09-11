#!/bin/bash

set -euo pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
RELEASE_DIR="${REPO_ROOT}/out"
FLAVORS_DIR="${REPO_ROOT}/templates/flavors"
# Passed in by the Makefile (tracks KUSTOMIZE_VERSION); fallback for manual runs.
KUSTOMIZE="${KUSTOMIZE:-${REPO_ROOT}/bin/kustomize-v5.7.1}"

SUPPORTED_FLAVORS=(
    "default"
    "cilium"
    "vultr-ccm"
    "vultr-csi"
    "full"
)

mkdir -p "${RELEASE_DIR}"

echo "==> Copying ClusterClass..."
cp "${REPO_ROOT}/templates/base/clusterclass.yaml" "${RELEASE_DIR}/clusterclass-vultr.yaml"

echo "==> Copying standalone cluster template..."
cp "${REPO_ROOT}/templates/cluster-template.yaml" "${RELEASE_DIR}/cluster-template.yaml"

echo "==> Building infrastructure components..."
"${KUSTOMIZE}" build "${REPO_ROOT}/config/default" > "${RELEASE_DIR}/infrastructure-components.yaml"

for flavor in "${SUPPORTED_FLAVORS[@]}"; do
    echo "==> Generating ${flavor} flavor..."
    "${KUSTOMIZE}" build "${FLAVORS_DIR}/${flavor}" > "${RELEASE_DIR}/cluster-template-${flavor}.yaml"
done

mv "${RELEASE_DIR}/cluster-template-default.yaml" "${RELEASE_DIR}/cluster-template-clusterclass-kubeadm.yaml"

echo "==> Done. Release artifacts in ${RELEASE_DIR}/"
echo "    - infrastructure-components.yaml"
echo "    - clusterclass-vultr.yaml"
echo "    - cluster-template.yaml"
echo "    - cluster-template-clusterclass-kubeadm.yaml"
echo "    - cluster-template-full.yaml"
echo "    - cluster-template-cilium.yaml"
echo "    - cluster-template-vultr-ccm.yaml"
echo "    - cluster-template-vultr-csi.yaml"