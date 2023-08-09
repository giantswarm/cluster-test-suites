[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/cluster-test-suites/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/cluster-test-suites/tree/main)

# cluster-test-suites

## ☑️ Requirements

* A valid Kubeconfig with the following context available:
  * `capa` pointing to a valid CAPA MC
  * `capz` pointing to a valid CAPZ MC
  * `capv` pointing to a valid CAPV MC
  * `capvcd` pointing to a valid CAPVCD MC
* The `E2E_KUBECONFIG` environment variable set to point to the path of the above kubeconfig.
* When `E2E_WC_NAME` and `E2E_WC_NAMESPACE` environment variables are set, the tests will run against the specified WC on the targeted MC. If one or both of the variables isn't set, the tests will create their own WC.

Example kubeconfig:

```yaml
apiVersion: v1
kind: Config
contexts:
- context:
    cluster: glippy
    user: glippy-admin
  name: capz
- context:
    cluster: grizzly
    user: grizzly-admin
  name: capa
- context:
    cluster: gcapeverde
    user: gcapeverde-admin
  name: capv
- context:
    cluster: guppy
    user: guppy-admin
  name: capvcd
clusters:
- cluster:
    certificate-authority-data: [REDACTED]
    server: https://[REDACTED]:6443
  name: glippy
- cluster:
    certificate-authority-data: [REDACTED]
    server: https://[REDACTED]:6443
  name: grizzly
- cluster:
    certificate-authority-data: [REDACTED]
    server: https://[REDACTED]:6443
  name: gcapeverde
- cluster:
    certificate-authority-data: [REDACTED]
    server: https://[REDACTED]:6443
  name: guppy
current-context: grizzly
preferences: {}
users:
- name: glippy-admin
  user:
    client-certificate-data: [REDACTED]
    client-key-data: [REDACTED]
- name: grizzly-admin
  user:
    client-certificate-data: [REDACTED]
    client-key-data: [REDACTED]
- name: gcapeverde-admin
  user:
    client-certificate-data: [REDACTED]
    client-key-data: [REDACTED]
- name: guppy-admin
  user:
    client-certificate-data: [REDACTED]
    client-key-data: [REDACTED]

```

## 🏃 Running Tests

Assuming the above requirements are fulfilled:

Running the entire test suite:

```sh
E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo -v -r .
```

Running a single provider (e.g. `capa`):

```sh
E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo -v -r ./providers/capa
```

Running with Docker:

```sh
docker run --rm -it -v /path/to/kubeconfig.yaml:/kubeconfig.yaml -e E2E_KUBECONFIG=/kubeconfig.yaml quay.io/giantswarm/cluster-test-suites ./
```

## ➕ Adding Tests

> See the Ginkgo docs for specifics on how to write tests: https://onsi.github.io/ginkgo/#writing-specs

Where possible, new tests cases should be added that are compatible with all providers so that all benefit. The is obviously not always possible and some provider-specific tests will be required.

All tests make use of our [clustertest](https://github.com/giantswarm/clustertest) test framework.

### Adding cross-provider tests

New cross-provider tests should be added to the [`./common/common.go`](./common/common.go) package as part of the `Run` function.

A new test case can be included byu adding a new `It` block within the `Run` finction. E.g.

```go
It("should test something", func() {
  // Do things...

  Expect(something).To(BeTrue())

  // Cleanup if needed...
})
```

### Adding provider-specific tests

Each CAPI provider has its own subdirectory under [`./providers/`](./providers/) that specific tests can be added to.

Each directory under the provider directory is a test suite and each consists to a specific workload cluster configuration variation. All providers should at least contain a `standard` test suite that runs tests for a "default" cluster configuration.

New tests can be added to these provider-specific suites using any Ginkgo context nodes that make sense. Please refer to the [Ginkgo docs](https://onsi.github.io/ginkgo/) for more details.

### Adding Test Suites

As mentioned above, test suites are scoped to a single workload cluster configuration variant. To test the different possible configuration options of clusters, for example private networking, we need to create multiple test suites.

A new test suite is added by creating a new module under the provider directory containing the following:

* a `test_data` directory of values files
* a `suite_test.go` file
* at least one `*_test.go` file

The `suite_test.go` should mostly be the same across test suites so it will likely be enough to copy the function over from the `standard` test suite and update the names used to represent the test suite being created.

The `test_data` directory should contain at least the values files for the cluster app and the default-apps app. These values are what indicate the variant used for this test suite. See [Creating values files](#Creating-values-files) below for more details.

### Creating values files

Values files can be stored as `*.yaml` files and loaded in using the test framework for use when creating apps in clusters.

The values files can use Go templating to replace some specific values (see the [clustertest docs](https://pkg.go.dev/github.com/giantswarm/clustertest/pkg/application#ValuesTemplateVars) for specific variables available).

E.g. the following will have the `name` and `organization` values replaced with those set on the Cluster instance.

```yaml
metadata:
  name: "{{ .ClusterName }}"
  description: "E2E Test cluster"
  organization: "{{ .Organization }}"

controlPlane:
  replicas: 3
```

> Note: It is best to always wrap these template strings in quote where appropriate to ensure the data used doesn't accidentally break the yaml schema.
