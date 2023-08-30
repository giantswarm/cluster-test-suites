#!/usr/bin/env bash

if [ "${E2E_KUBECONFIG}" == "" ]; then
  echo "The env var 'E2E_KUBECONFIG' must be provided"
  exit 1
fi

SUITES_TO_RUN=$(find $1 -name '*.test' | xargs)
shift

REPORT_DIR=${REPORT_DIR:-/tmp/reports}
mkdir -p ${REPORT_DIR}

ginkgo --output-dir=${REPORT_DIR} --junit-report=test-results.xml --timeout 4h --keep-going -v -r $@ ${SUITES_TO_RUN}
