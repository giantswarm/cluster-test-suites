#!/bin/bash

set -e

export E2E_KUBECONFIG=~/.kube/clusters/e2e.yaml

# CAPA
for i in {1..5}; do
  WORKING_DIR=`mktemp -d`
	cd ${WORKING_DIR}

  echo "Creating new cluster"
  OUTPUT=$(/Users/marcus/Code/GiantSwarm/cluster-test-suites/standup \
              --provider aws --context capa \
              --cluster-values /Users/marcus/Code/GiantSwarm/cluster-test-suites/providers/capa/standard/test_data/cluster_values.yaml \
              --default-apps-values /Users/marcus/Code/GiantSwarm/cluster-test-suites/providers/capa/standard/test_data/default-apps_values.yaml \
              --cluster-version latest
          )
  echo "${OUTPUT}"
  RESULT=$(echo "${OUTPUT}" | tail -n 1)
  echo ${RESULT}
  printf "${RESULT}" >> "/Users/marcus/Downloads/CAPI Timings/capa.csv"
  echo "Tearing down cluster"
  OUTPUT=$(/Users/marcus/Code/GiantSwarm/cluster-test-suites/teardown --provider aws --context capa --standup-directory ${WORKING_DIR})
  RESULT=$(echo "${OUTPUT}" | tail -n 1)
  echo ${RESULT}
  printf ",${RESULT}" >> "/Users/marcus/Downloads/CAPI Timings/capa.csv"
  echo "" >> "/Users/marcus/Downloads/CAPI Timings/capa.csv"
  echo ""
done

# CAPZ
for i in {1..5}; do
  WORKING_DIR=`mktemp -d`
	cd ${WORKING_DIR}

  echo "Creating new cluster"
  OUTPUT=$(/Users/marcus/Code/GiantSwarm/cluster-test-suites/standup \
              --provider azure --context capz \
              --cluster-values /Users/marcus/Code/GiantSwarm/cluster-test-suites/providers/capz/standard/test_data/cluster_values.yaml \
              --default-apps-values /Users/marcus/Code/GiantSwarm/cluster-test-suites/providers/capz/standard/test_data/default-apps_values.yaml \
              --cluster-version latest
          )
  echo "${OUTPUT}"
  RESULT=$(echo "${OUTPUT}" | tail -n 1)
  echo ${RESULT}
  printf "${RESULT}" >> "/Users/marcus/Downloads/CAPI Timings/capz.csv"
  echo "Tearing down cluster"
  OUTPUT=$(/Users/marcus/Code/GiantSwarm/cluster-test-suites/teardown --provider azure --context capz --standup-directory ${WORKING_DIR})
  RESULT=$(echo "${OUTPUT}" | tail -n 1)
  echo ${RESULT}
  printf ",${RESULT}" >> "/Users/marcus/Downloads/CAPI Timings/capz.csv"
  echo "" >> "/Users/marcus/Downloads/CAPI Timings/capz.csv"
  echo ""
done

# CAPV
for i in {1..5}; do
  WORKING_DIR=`mktemp -d`
	cd ${WORKING_DIR}

  echo "Creating new cluster"
  OUTPUT=$(/Users/marcus/Code/GiantSwarm/cluster-test-suites/standup \
              --provider vsphere --context capv \
              --cluster-values /Users/marcus/Code/GiantSwarm/cluster-test-suites/providers/capv/standard/test_data/cluster_values.yaml \
              --default-apps-values /Users/marcus/Code/GiantSwarm/cluster-test-suites/providers/capv/standard/test_data/default-apps_values.yaml \
              --cluster-version latest
          )
  echo "${OUTPUT}"
  RESULT=$(echo "${OUTPUT}" | tail -n 1)
  echo ${RESULT}
  printf "${RESULT}" >> "/Users/marcus/Downloads/CAPI Timings/capv.csv"
  echo "Tearing down cluster"
  OUTPUT=$(/Users/marcus/Code/GiantSwarm/cluster-test-suites/teardown --provider vsphere --context capv --standup-directory ${WORKING_DIR})
  RESULT=$(echo "${OUTPUT}" | tail -n 1)
  echo ${RESULT}
  printf ",${RESULT}" >> "/Users/marcus/Downloads/CAPI Timings/capv.csv"
  echo "" >> "/Users/marcus/Downloads/CAPI Timings/capv.csv"
  echo ""
done

