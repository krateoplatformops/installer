#!/bin/bash

kubectl apply -f crds/
kubectl apply -f testdata/ns.yaml
kubectl apply -f testdata/krateo.yaml


go run cmd/main.go --debug --poll "2m" -n "krateo-system"
