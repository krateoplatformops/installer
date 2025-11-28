#!/bin/bash

PROJECT_DIR=$( pwd )
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

cd ${PROJECT_DIR}

# SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
echo "SCRIPT_DIR: ${SCRIPT_DIR}"

kubectl delete -f manifests/

${SCRIPT_DIR}/generate.sh
${SCRIPT_DIR}/build.sh

kubectl apply -f manifests/

kubectl apply -f crds