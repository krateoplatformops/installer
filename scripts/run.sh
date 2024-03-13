#!/bin/bash

kind get kubeconfig >/dev/null 2>&1 || kind create cluster

kubectl apply -f crds/
kubectl apply -f testdata/ns.yaml
kubectl apply -f testdata/sample.yaml


go run cmd/main.go --debug --poll "2m"
