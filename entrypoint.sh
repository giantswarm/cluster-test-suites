#!/usr/bin/env bash

if [ "${E2E_KUBECONFIG}" == "" ]; then
  echo "The env var 'E2E_KUBECONFIG' must be provided"
  exit 1
fi

ginkgo -v $@
