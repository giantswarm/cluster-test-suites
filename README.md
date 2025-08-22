[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/cluster-test-suites/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/cluster-test-suites/tree/main)

# cluster-test-suites

## ☑️ Requirements

* A valid Kubeconfig, pointing at a `stable-testing` MC, with the required context defined. (See [cluster-standup-teardown](https://github.com/giantswarm/cluster-standup-teardown) for more details.)
* Install [ginkgo](https://onsi.github.io/ginkgo/) on your machine: `go install github.com/onsi/ginkgo/v2/ginkgo`.\
  Run this command from inside the repository to get the correct version and do not just install `latest` instead. The following error is an indicator of not being in the correct directory:
  ```
  go: 'go install' requires a version when current directory is not in a module
  	Try 'go install github.com/onsi/ginkgo/v2/ginkgo@latest' to install the latest version
  ```
* The `E2E_KUBECONFIG` environment variable set to point to the path of the above kubeconfig.

Optional:

* When `E2E_WC_NAME` and `E2E_WC_NAMESPACE` environment variables are set, the tests will run against the specified WC on the targeted MC. If one or both of the variables isn't set, the tests will create their own WC.
* When `TELEPORT_IDENTITY_FILE` environment variable is set to point to the path of a valid teleport credential, the test will check if E2E WC is registered in Teleport cluster (`teleport.giantswarm.io`). If it isn't set, the test will be skipped.

## 🏃 Running Tests

> [!NOTE]
> If you need the current kubeconfig its best to pull it from the `cluster-test-suites-mc-kubeconfig` Secret on the Tekton cluster.
>
> If you need the current teleport identity file its best to pull it from the `teleport-identity-output` Secret on the Tekton cluster.

> [!IMPORTANT]
> The test suites are designed to be run against `stable-testing` MCs and possibly require some config or resources that already exists on those MCs.
>
> If you require running the tests against a different MC please reach out to [#Team-Tenet](https://gigantic.slack.com/archives/C07KSM2E51A) to discuss any pre-requisites that might be needed.

Assuming the above requirements are fulfilled:

* Running all the test suites:

  ```sh
  E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo --timeout 4h -v -r .
  ```

* Running a single provider (e.g. `capa`):

  ```sh
  E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo --timeout 4h -v -r ./providers/capa
  ```

* Running a single test suite (e.g. the `capa` `standard` test suite)

  ```sh
  E2E_KUBECONFIG=/path/to/kubeconfig.yaml ginkgo --timeout 4h -v -r ./providers/capa/standard
  ```

* Running a single test suite with teleport test enabled (e.g. the `capa` `standard` test suite):

  ```sh
  kubectl get secrets teleport-identity-output -n tekton-pipelines --template='{{.data.identity}}' | base64 -D > teleport-identity-file.pem

  E2E_KUBECONFIG=/path/to/kubeconfig.yaml TELEPORT_IDENTITY_FILE=/path/to/teleport-identity-file.pem -v -r ./providers/capa/standard
  ```

* Running with Docker:

  ```sh
  docker run --rm -it -v /path/to/kubeconfig.yaml:/kubeconfig.yaml -e E2E_KUBECONFIG=/kubeconfig.yaml quay.io/giantswarm/cluster-test-suites ./
  ```

### Testing with an in-progress Release CR

To be able to create a workload cluster based on a not yet merged Release CR you can use the following two environment variables:

* `E2E_RELEASE_VERSION` - The base Release version to use when creating the Workload Cluster.<br/>Must be used with `E2E_RELEASE_COMMIT`
* `E2E_RELEASE_COMMIT` - The git commit from the `releases` repo that contains the Release version to use when creating the Workload Cluster.<br/>Must be used with `E2E_RELEASE_VERSION`

The Release CR must at least be committed and pushed to a branch (e.g. a WiP PR) in the [Releases](https://github.com/giantswarm/releases) repo.

### Testing with an existing Workload Cluster

It's possible to re-use an existing workload cluster to speed up development and to debug things after the tests have run. The Workload Cluster needs to be created manually in the relevant Management Cluster first and once ready you can set the following environment variables when running the tests to make use of your own WC:

* `E2E_WC_NAME` - The name of your Workload Cluster on the MC
* `E2E_WC_NAMESPACE` - The namespace your Workload Cluster is in on the MC

Example:

```sh
E2E_KUBECONFIG=/path/to/kubeconfig.yaml E2E_WC_NAME=mn-test E2E_WC_NAMESPACE=org-giantswarm ginkgo -v -r ./providers/capa/standard
```

If you'd like to create a workload cluster test using the same configuration as the test suites you can make use of the `standup` & `teardown` CLIs available in [cluster-standup-teardown](https://github.com/giantswarm/cluster-standup-teardown).

### Testing changes to `clustertest`

To test out changes to [clustertest](https://github.com/giantswarm/clustertest) without needing to create a new release you can add a `replace` directive to your `go.mod` to point to your local copy of `clustertest`.

```
module github.com/giantswarm/cluster-test-suites

go 1.20

replace github.com/giantswarm/clustertest => /path/to/clustertest

```

### Testing changes to `cluster-standup-teardown`

To test out changes to [cluster-standup-teardown](https://github.com/giantswarm/cluster-standup-teardown) without needing to create a new release you can add a `replace` directive to your `go.mod` to point to your local copy of `cluster-standup-teardown`.

```
module github.com/giantswarm/cluster-test-suites

go 1.20

replace github.com/giantswarm/cluster-standup-teardown => /path/to/cluster-standup-teardown

```

### ⚙️ Running Tests in CI

These tests are configures to run in our Tekton pipelines with our [cluster-test-suites pipeline](https://github.com/giantswarm/tekton-resources/blob/main/tekton-resources/tekton-pipelines/pipelines/cluster-test-suites.yaml). This pipeline can be triggered on appropriate repos by using the `/run cluster-test-suites` comment trigger.

Note: Test suites are configured to run in parallel using a [Matrix](https://tekton.dev/docs/pipelines/matrix/) in the Tekton Pipeline. Once all suites are complete the results of each will be collected and presented in a final Pipeline Task to show the results of all suites.

When the pipeline runs against PRs to the `cluster-test-suites` repo the following is of note:
* The pipeline will use a container image built from the changes in the PR. The image building and publishing is still handled by CircleCI so the pipeline has a step at the start to wait for this image to be available.
* The `upgrade` tests won't run as there's no information as to what versions to upgrade from/to.
* All providers will be tested. These test suites will run in parallel and the pipeline will wait for all of them to complete before finishing.

When the pipeline runs against one of the provider-specific repos (e.g. cluster or default-apps repos) the following is of note:
* The pipeline will use the latest tagged release of the cluster-test-suites container image.
* Only the tests associated with that provider will be run, including the `upgrade` tests.

### Running a specific subset of test suites

It's possible to specify the test suites you'd like to run in CI by providing the `TARGET_SUITES` parameter with your comment trigger. This is useful when a test suite has failed due to what seems to be a flakey test as you can re-run just the failing without wasting resources on the other test suites.

E.g.

```
/run cluster-test-suites TARGET_SUITES=./providers/capa/standard
```

This will only run the CAPA Standard test suite.

If you need to target multiple test suites you can do so with a comma separated list, e.g.

```
/run cluster-test-suites TARGET_SUITES=./providers/capa/standard,./providers/capa/china
```

### Running against a specific Release version

If you need to run the tests against a specific Release version that is not the latest you can do so by providing the `RELEASE_VERSION` parameter with your comment trigger.

E.g.

```
/run cluster-test-suites RELEASE_VERSION=v25.0.0
```

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
* [CI Tekton Pipeline](https://github.com/giantswarm/tekton-resources/blob/main/tekton-resources/tekton-pipelines/pipelines/cluster-test-suites.yaml)
* [Ginkgo docs](https://onsi.github.io/ginkgo/)
