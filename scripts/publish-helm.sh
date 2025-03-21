#!/usr/bin/env bash
# Publishes our helm chart via OCI.
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

required_tools=("yq" "helm")
for cmd in "${required_tools[@]}"; do
	if ! command -v "$cmd" @ >/dev/null; then
		echo "Missing '$cmd' (try 'mise install' or 'brew install'?)"
		exit 1
	fi
done

NEXT_VERSION=${1:-}
if [[ "$NEXT_VERSION" == "" ]]; then
	echo "Missing version"
	exit 1
fi

pushd "$REPO_ROOT/deploy/charts/ingress-anubis" >/dev/null
yq e -i '.version = "'"${NEXT_VERSION//v/}"'"' Chart.yaml
yq e -i '.appVersion = "'"${NEXT_VERSION//v/}"'"' Chart.yaml

rm -rf ./*.tgz
helm package .
helm push ./*.tgz oci://ghcr.io/jaredallard/helm-charts
popd >/dev/null
