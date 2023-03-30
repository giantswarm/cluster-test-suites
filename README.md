[![CircleCI](https://circleci.com/gh/giantswarm/cluster-test-suites.svg?style=shield)](https://circleci.com/gh/giantswarm/cluster-test-suites)

# cluster-test-suites

## Requirements

* A valid Kubeconfig with the following context available:
  * `capa` pointing to a valid CAPA MC
  * `capz` pointing to a valid CAPZ MC
  * `capvcd` pointing to a valid CAPVCD MC
* The `E2E_KUBECONFIG` environment variable set to point to the path of the above kubeconfig.
