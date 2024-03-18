#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

kind get kubeconfig >/dev/null 2>&1 || kind create cluster --wait 120s --config $SCRIPT_DIR/kind.yaml

