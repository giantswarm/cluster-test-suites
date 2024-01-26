TESTING


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

> Note: If you need the current kubeconfig its best to pull it from the `cluster-test-suites-mc-kubeconfig` Secret on the Tekton cluster

Running the all test suites:

```sh
E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo -v -r .
```

Running a single provider (e.g. `capa`):

```sh
E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo -v -r ./providers/capa
```

Running a single test suite (e.g. the `capa` `standard` test suite)

```sh
E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo -v -r ./providers/capa/standard
```

Running with Docker:

```sh
docker run --rm -it -v /path/to/kubeconfig.yaml:/kubeconfig.yaml -e E2E_KUBECONFIG=/kubeconfig.yaml quay.io/giantswarm/cluster-test-suites ./
```

### Testing with an existing Workload Cluster

It's possible to re-use an existing workload cluster to speed up development and to debug things after the tests have run. The Workload Cluster needs to be created manually in the relevant Management Cluster first and once ready you can set the following environment variables when running the tests to make use of your own WC:

* `E2E_WC_NAME` - The name of your Workload Cluster on the MC
* `E2E_WC_NAMESPACE` - The namespace your Workload Cluster is in on the MC

Example:

```sh
E2E_KUBECONFIG=/path/to/kubeconfig.yaml E2E_WC_NAME=mn-test E2E_WC_NAMESPACE=org-giantswarm ginkgo -v -r ./providers/capa/standard
```

If you'd like to create a workload cluster test using the same configuration as the test suites you can make use of the [standup](./cmd/standup/) & [teardown](./cmd/teardown/) CLIs available in this repo.

### Testing changes to `clustertest`

To test out changes to [clustertest](https://github.com/giantswarm/clustertest) without needing to create a new release you can add a `replace` directive to your `go.mod` to point to your local copy of `clustertest`.

```
module github.com/giantswarm/cluster-test-suites

go 1.20

replace github.com/giantswarm/clustertest v0.10.0 => /path/to/clustertest

```

### ⚙️ Running Tests in CI

These tests are configures to run in our Tekton pipelines with our [cluster-test-suites pipeline](https://github.com/giantswarm/tekton-resources/blob/main/tekton-resources/pipelines/cluster-test-suites.yaml). This pipeline can be triggered on appropriate repos by using the `/run cluster-test-suites` comment trigger.

Note: Test suites are configured to run in parallel using a [Matrix](https://tekton.dev/docs/pipelines/matrix/) in the Tekton Pipeline. Once all suites are complete the results of each will be collected and presented in a final Pipeline Task to show the results of all suites.

When the pipeline runs against PRs to the `cluster-test-suites` repo the following is of note:
* The pipeline will use a container image built from the changes in the PR. The image building and publishing is still handled by CircleCI so the pipeline has a step at the start to wait for this image to be available.
* The `upgrade` tests won't run as there's no information as to what versions to upgrade from/to.
* All providers will be tested. These test suites will run in parallel and the pipeline will wait for all of them to complete before finishing.

When the pipeline runs against one of the provider-specific repos (e.g. cluster or default-apps repos) the following is of note:
* The pipeline will use the latest tagged release of the cluster-test-suites container image.
* Only the tests associated with that provider will be run, including the `upgrade` tests.

## ⬆️ Upgrade Tests

Each of the providers have a test suite called `upgrade` that is designed to first install a cluster using the latest released version of both the cluster App and the default-apps App. It then upgrades that cluster to whatever currently needs testing.

There are a few things to be aware about these tests:

* These test suites only run if a `E2E_OVERRIDE_VERSIONS` environment variable is set, indicating the versions to upgrade the Apps to. For example `E2E_OVERRIDE_VERSIONS="cluster-aws=0.38.0-5f4372ac697fce58d524830a985ede2082d7f461"`.
* The initial workload cluster created uses whatever the latest released version on GitHub is, this is not currently configurable.
* These test suites use [Ginkgo Ordered Containers](https://onsi.github.io/ginkgo/#ordered-containers) to ensure certain tests specs are run before and after the upgrade process as required.

## ➕ Adding Tests

> See the Ginkgo docs for specifics on how to write tests: https://onsi.github.io/ginkgo/#writing-specs

Where possible, new tests cases should be added that are compatible with all providers so that all benefit. The is obviously not always possible and some provider-specific tests will be required.

All tests make use of our [clustertest](https://github.com/giantswarm/clustertest) test framework. Please refer to the [documentation](https://pkg.go.dev/github.com/giantswarm/clustertest) for more information.

### Adding cross-provider tests

New cross-provider tests should be added to the [`./internal/common/common.go`](./internal/common/common.go) package as part of the `Run` function.

A new test case can be included by adding a new `It` block within the `Run` functions. E.g.

```go
It("should test something", func() {
  // Do things...

  Expect(something).To(BeTrue())

  // Cleanup if needed...
})
```

To add a new grouping of common tests you can create a new file with a function similar to `runMyNewGrouping()` and then add a call to this from the [`./internal/common/common.go`](./internal/common/common.go) `Run()` function.

### Adding provider-specific tests

Each CAPI provider has its own subdirectory under [`./providers/`](./providers/) that specific tests can be added to.

Each directory under the provider directory is a test suite and each consists to a specific workload cluster configuration variation. All providers should at least contain a `standard` test suite that runs tests for a "default" cluster configuration.

New tests can be added to these provider-specific suites using any Ginkgo context nodes that make sense. Please refer to the [Ginkgo docs](https://onsi.github.io/ginkgo/) for more details.

### Adding Test Suites (new cluster variations in an existing provider)

As mentioned above, test suites are scoped to a single workload cluster configuration variant. To test the different possible configuration options of clusters, for example private networking, we need to create multiple test suites.

A new test suite is added by creating a new module under the provider directory containing the following:

* a `test_data` directory of values files
* a `suite_test.go` file
* at least one `*_test.go` file

The `suite_test.go` should mostly be the same across test suites so it will likely be enough to copy the function over from the `standard` test suite and update the names used to represent the test suite being created. This file mostly relies on the [`suite`](./internal/suite/) module to handle the test suite setup and clean up logic.

The `test_data` directory should contain at least the values files for the cluster app and the default-apps app. These values are what indicate the variant used for this test suite. See [Creating values files](#Creating-values-files) below for more details.

### Creating values files

Values files can be stored as `*.yaml` files and loaded in using the test framework for use when creating apps in clusters.

The values files can use Go templating to replace some variables with their cluster-specific values (see the [clustertest docs](https://pkg.go.dev/github.com/giantswarm/clustertest/pkg/application#TemplateValues) for specific variables available). It is also possible to provide an `ExtraValues` map that contains key/value pairs made available to the templateing.

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

## Resources

* [`clustertest` documentation](https://pkg.go.dev/github.com/giantswarm/clustertest)
* [standup](./cmd/standup/) & [teardown](./cmd/teardown/) CLIs
* [CI Tekton Pipeline](https://github.com/giantswarm/tekton-resources/blob/main/tekton-resources/pipelines/cluster-test-suites.yaml)
* [Ginkgo docs](https://onsi.github.io/ginkgo/)
