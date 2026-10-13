#!/usr/bin/env bash

if [ "${E2E_KUBECONFIG}" == "" ]; then
  echo "The env var 'E2E_KUBECONFIG' must be provided"
  exit 1
fi

mapfile -t SUITES_TO_RUN < <(find "$1" -name '*.test')
# Without suites, `ginkgo -r` would compile and run every suite in the working directory
if [ "${#SUITES_TO_RUN[@]}" -eq 0 ]; then
  echo "No test suites found in '$1'"
  exit 1
fi
shift

REPORT_DIR=${REPORT_DIR:-/tmp/reports}
mkdir -p "${REPORT_DIR}"

ginkgo --output-dir="${REPORT_DIR}" --junit-report=test-results.xml --json-report=test-results.json --timeout 4h --keep-going -v -r "$@" "${SUITES_TO_RUN[@]}"
