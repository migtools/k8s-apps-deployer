#!/bin/sh

set -eu

kubeconfig=${KUBECONFIG:-${HOME}/.kube/config}
image=${K8SDEPLOY_IMAGE:-quay.io/migqe/k8s-apps-deployer:multiarch}

exec podman run --rm --pull=missing \
  --volume "${kubeconfig}:/kube/config:ro" \
  --env KUBECONFIG=/kube/config \
  "${image}" \
  "$@"
