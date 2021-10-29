#!/bin/bash

bash vendor/k8s.io/code-generator/generate-groups.sh \
     all \
     github.com/giolekva/unique/operator/generated \
     github.com/giolekva/unique/operator/apis \
     "unique:v1" \
     --go-header-file hack/boilerplate.go.txt
