# standup

Create a workload cluster using the same known-good configuration as the E2E test suites.

Once the workload cluster is ready two files will be produced:

* a `kubeconfig` to use to access the cluster
* a `results.json` that contains details about the cluster created and can be used by `teardown` to cleanup the cluster when done

## Install

```shell
go install github.com/giantswarm/cluster-test-suites/cmd/standup@latest
```

## Usage

```
$ standup --help

Standup create a test workload cluster in a standard, reproducible way.
A valid Management Cluster kubeconfig must be available and set to the `E2E_KUBECONFIG` environment variable.

Usage:
  standup [flags]

Examples:
standup --provider aws --context capa --cluster-values ./cluster_values.yaml --default-apps-values ./default-apps_values.yaml

Flags:
      --cluster-values string         The path to the cluster app values (required)
      --cluster-version string        The version of the cluster app to install (default "latest")
      --context string                The kubernetes context to use (required)
      --control-plane-nodes int       The number of control plane nodes to wait for being ready (default 1)
      --default-apps-values string    The path to the default-apps app values (required)
      --default-apps-version string   The version of the default-apps app to install (default "latest")
  -h, --help                          help for standup
      --output string                 The directory to store the results.json and kubeconfig in (default "./")
      --provider string               The provider (required)
      --worker-nodes int              The number of worker nodes to wait for being ready (default 1)
```

### Example

```
standup --provider eks --context eks \
  --cluster-values ./providers/eks/standard/test_data/cluster_values.yaml \
  --default-apps-values ./providers/eks/standard/test_data/default-apps_values.yaml
```
