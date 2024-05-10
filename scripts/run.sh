#!/bin/bash

kubectl apply -f crds/
kubectl apply -f testdata/ns.yaml
kubectl apply -f testdata/vcluster.yaml


go run cmd/main.go --poll "2m" -n "krateo-system"
