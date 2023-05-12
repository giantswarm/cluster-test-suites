#!/usr/bin/env bash

if [ "${E2E_KUBECONFIG}" == "" ]; then
  echo "The env var 'E2E_KUBECONFIG' must be provided"
  exit 1
fi

SUITES_TO_RUN=$(find $1 -name '*.test' | xargs)
shift

ginkgo --timeout 4h --keep-going -v -r $@ ${SUITES_TO_RUN}
