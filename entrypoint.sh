#!/usr/bin/env bash

if [ "$1" == "" ]; then
  echo "Test suite needs to be provided"
  exit 1
fi

ginkgo -v $1
