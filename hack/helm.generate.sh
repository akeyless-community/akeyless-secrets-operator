#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BUNDLE_DIR="${1}"
HELM_DIR="${2}"

cd "${SCRIPT_DIR}"/../

# Split the generated bundle yaml file to inject control flags
yq e -Ns "\"${HELM_DIR}/templates/crds/\" + .spec.names.singular" ${BUNDLE_DIR}/bundle.yaml

python3 "${SCRIPT_DIR}/helm.generate_crds.py" "${HELM_DIR}/templates/crds"
