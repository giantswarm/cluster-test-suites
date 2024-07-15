# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- A framework for overriding default timeouts used by test cases. Introduces a new `timeout` internal package and new functions on the `state` that allows getting and setting a custom timeout per test suite.
- Re-enabled `capa-private` tests because EC2 user data limit issue should be solved now that we store Ignition bootstrap data in S3 ([issue](https://github.com/giantswarm/roadmap/issues/3442))

## [1.59.0] - 2024-07-09

### Changed

- Updated `cluster-standup-teardown` to the latest release.

## [1.58.0] - 2024-07-08

### Added

- Added failure handling for Deployment, StatefulSet and DaemonSet not having expected number of ready replicas
- Updated `clustertest` to include logs when debugging failing pods

## [1.57.2] - 2024-07-05

### Fixed

- Switched to using `ShouldSkipUpgrade` from `clustertest`

## [1.57.1] - 2024-07-05

### Fixed

- Allow upgrade tests to be run from Releases test pipeline

## [1.57.0] - 2024-07-05

### Changed

- Switched to using DNS resolver and HTTP client from `clustertest`

## [1.56.0] - 2024-07-02

### Changed

- Updated upgrade tests so they can test upgrades to Releases

## [1.55.0] - 2024-06-28

### Changed

- Made use of `GetWarningEventsForResource` from `clustertest` for Cert-Manager tests

### Added

- Use `failurehandler.AppIssues` to provide extra debugging when App-related tests fail (timeout)

## [1.54.1] - 2024-06-27

## [1.54.0] - 2024-06-27

### Changed

- Update `clustertest` to v1.7.0 to support the new environment variable for controlling what Release to use when creating clusters

## [1.53.0] - 2024-06-25

### Added

- Added a test case to check the status of all Jobs in the WC

### Changed

- Use actual types from Cert-Manager instead of using `unstructured` for `Certificate` and `ClusterIssuer`
- Refactored some functions to explicitly take in a context instance instead of using `context.Background()` so that timeouts set on the context can be respected

## [1.52.0] - 2024-06-25

### Changed

- Updated all dependencies to latest version

## [1.51.0] - 2024-06-24

### Changed

- Updated all depenedncies to latest version

## [1.50.0] - 2024-06-24

### Added

- Added a test case to ensure the hello-world ingress has a ready Certificate
- Included extra logging in ClusterIssuer test case to output the status and failing events of the post-install Helm Job.

## [1.49.0] - 2024-06-21

### Changed

- Updated `clustertest` to latest with additional logging for node checks
- Updated `cluster-standup-teardown` to latest with cluster-autoscaler config for `scaleDownUnneededTime`

## [1.48.0] - 2024-06-20

### Fixed

- Ensure timeout is always reset on the context in the AfterSuite to ensure enough time is given for cleanup

### Added

- Test if default apps are deployed before upgrading the cluster.
- Test workload cluster's Deployments, DaemonSets, StatefulSets and Pods before upgrading the cluster.
- Test if cluster is ready before upgrading the cluster (check Cluster resource Ready condition).
- Test if machine pools are ready and running before upgrading the cluster.
- Test if machine pools are ready and running in common tests.

### Changed

- Update clustertest to v1.3.0 to support releases with cluster Apps for Azure.
- Updated teleport api module to latest available.
- Increase node pool min size from 2 to 3 for CAPA upgrade test.
- Disable spot instances for CAPA upgrade suite, as we suspect that using spot instances is causing Upgrade suite failures lately.

### Removed

- Remove node checks from CAPA upgrade suite because we already check nodes in common tests after the upgrade.

### Added

- Additional logging added to the deployment scale test to display conditions of the pods and taints of the worker nodes

## [1.47.0] - 2024-06-14

### Added

- Add `china` test suite for `capa` provider.
- Add wildcard ingress DNS check test to the `hello` test.

## [1.46.0] - 2024-06-13

### Changed

- Update `clustertest` and `cluster-standup-teardown` to latest releases
- Switch to using new `ApplyBuiltCluster` in upgrade tests to avoid building the cluster twice

## [1.45.0] - 2024-06-10

### Changed

- Update `clustertest` to v1.0.0 to support Releases with cluster Apps
- Update `cluster-standup-teardown` to v1.5.0 to support Releases with cluster Apps

## [1.44.3] - 2024-06-10

### Changed

- Update `cluster-standup-teardown` to v1.5.0

## [1.44.2] - 2024-06-06

### Changed

- Update `cluster-standup-teardown` to v1.4.0

## [1.44.1] - 2024-05-17

### Changed

- Update `cluster-standup-teardown` to v1.3.0

## [1.44.0] - 2024-05-16

### Added

- Add test for CAPA cluster in Cilium ENI mode

## [1.43.0] - 2024-05-16

### Added

- Wait and test for `security-bundle` apps to be deployed.

### Changed

- Increase `kubeadmControlPlane` rollout timeout from 20min to 30min.
- Increase node roll check time to 180 seconds from 25 seconds in the upgrade test.
- Reduce the timeout for default apps checks

### Fixed

- Skip checking default-apps-aws version when unified cluster-aws app is deployed.

## [1.42.0] - 2024-05-14

### Added

- Add support for unified cluster-aws app. With cluster-aws v0.76.0 and newer, default apps are deployed with cluster-aws and default-apps-aws app is not deployed anymore.

## [1.41.0] - 2024-05-13

### Added

- CAPV on CAPZ tests.

### Changed

- Get base domain from Cluster values instead of default-apps values

## [1.40.0] - 2024-05-10

### Changed

- Make error messages actionable for Prometheus metrics query test failures
- Disabled `capa-private` until we have fixed the userdata issue with MachinePools on `goat`

## [1.39.1] - 2024-04-29

### Changed

- Updated cluster-standup-teardown to v1.0.2
- Made use of new `clusterbuilder.KubeContext()` function
- Updated readme requirements

### Removed

- Removed KubeContext const from each test suite

## [1.39.0] - 2024-04-26

### Changed

- Switched to using `cluster-standup-teardown` to handle cluster creation and deletion logic
- Removed cluster and default-apps values in favour of defaults from `cluster-standup-teardown`

## [1.38.0] - 2024-04-25

### Changed

- CAPZ: Change Helm values structure where root-level properties are moved to `.Values.global`.

## [1.37.0] - 2024-04-19

### Added

- Add common basic test that checks if Cluster Ready condition has Status=True.
- Add upgrade test that checks if control plane rolling update has finished (if it has started).

## [1.36.0] - 2024-04-18

### Changed

- Revert #169 by increasing CAPVCD CP nodes back to 3.
- Use default values of `cluster-cloud-director chart:v0.50.0`.
- Update README example kubeconfig for CAPVCD to use `gerbil` as `guppy` is dead.

## [1.35.0] - 2024-04-08

### Added

- Add support for mimir in the `metrics` test.

## [1.34.0] - 2024-04-04

### Added

- Include the MCs api endpoint when logging out the "checking connection to MC" message

### Changed

- Switch CAPVCD to Flatcar (1.25.16).

## [1.33.0] - 2024-03-28

### Changed

- Increase cluster creation timeout to 30 minutes.
- Switch CAPVCD values to 1.25.13.

## [1.32.0] - 2024-03-20

### Changed

- Increase cert-manager test timeout to 2 minutes.
- Enabled EKS tests.
- Avoid checking control plane metrics in EKS.

## [1.31.0] - 2024-03-14

### Changed

- Disabled EKS tests due to environment being broken

## [1.30.1] - 2024-03-14

### Fixed

- Set app versions when using `standup` CLI
- Create the `results.json` file as soon as possible when running `standup` in case the cluster creation fails to allows the `teardown` command to still use it.

## [1.30.0] - 2024-03-12

### Added

- Add a new test that checks if key workload cluster metrics exist in prometheus.

## [1.29.0] - 2024-02-29

### Added

- Add teleport connectivity test

## [1.28.0] - 2024-02-29

### Changed

- Bump Go version `v1.21` for teleport connectivity test
- Update CAPV values and make more use of chart's default values

### Added

- Try and use ingress nginx app's config if present.
- Enable ingress test for private CAPA test.

### Fixed

- In environments where an egress proxy exists, don't use 8.8.8.8 as a DNS resolver for the ingress test as that would prevent the proxy address from being resolved.

## [1.27.0] - 2024-02-14

### Changed

- Enable scale test for private CAPA clusters.
- Increase max node pool size to 5 for CAPA test suites.

## [1.26.2] - 2024-02-13

### Fixed

- Ensure upgrade tests install latest released version first

## [1.26.1] - 2024-02-12

### Changed

- Use `Standard_D4s_v5` and 2 replicas for workers in CAPZ.

## [1.26.0] - 2024-02-08

### Added

- Add tests on DeploymentSets, StatefulSets and DaemonSets to ensure all desired replicas/pods are running.

## [1.25.4] - 2024-02-08

### Changed

- Re-enable CAPVCD tests on gerbil.

## [1.25.3] - 2024-02-05

### Changed

- Disable Bastion Support for CAPZ

## [1.25.2] - 2024-01-31

### Changed

- Make PVC pod run unprivelleged.

## [1.25.1] - 2024-01-31

### Changed

- Add security context to PVC pod.

## [1.25.0] - 2024-01-30

### Changed

- Refactored `hello-world` tests to split out into individual (ordered) test cases to better understand where things fail
- Increased the timout of the `hello-world` ingress check from 10min to 15min
- Introduced a check for the ingress resources status being updated with the load balancer hostname

## [1.24.0] - 2024-01-26

### Changed

- Set min worker nodes to 2 for CAPA clusters
- Make the scale test dynamic based on the current number of worker nodes

## [1.23.2] - 2024-01-26

### Changed

- Use custom dialer when calling hello-world application to avoid DNS caching the not found result

## [1.23.1] - 2024-01-19

### Changed

- Disabled CAPVCD tests while we recover the broken MC

## [1.23.0] - 2024-01-19

### Added

- Add a retrying healthcheck call to the MC at the start of the test suite to ensure connection is usable

### Changed

- Re-enabled CAPA private test suite

## [1.22.0] - 2024-01-18

### Changed

- Disable capa private test suite until MC stability issues are solved

## [1.21.3] - 2024-01-17

### Changed

- Remove CPI config from CAPVCD tests so that we use the chart defaults instead.

## [1.21.2] - 2024-01-15

### Changed

- Consolidated test suite setup (`BeforeSuite` and `AfterSuite`) into a single, reusable module to reduce duplication.

### Fixed

- Added `AfterSuite` to perform test cluster cleanup. This should now cleanup clusters even if they timeout during creation in the `BeforeSuite`.

### Added

- Added EKS cluster support to the `standup` CLI

## [1.21.1] - 2024-01-11

### Fixed

- Fix ClusterIssuer test to actually keep checking for issues being ready until timeout
- Fixed typo in CAPA private suite name

## [1.21.0] - 2024-01-11

### Added

- CAPA private cluster tests
- EKS autoscale tests

### Changed

- Allow basic tests to be flakey and retry in case of network issues

## [1.20.4] - 2024-01-08

### Changed

- Reduce the number of CP nodes from 3 to 1 in CAPVCD tests in order to avoid timeouts.

## [1.20.3] - 2023-12-19

### Added

- Add test config for upgrades to be able to customize timeouts per provider.

## [1.20.2] - 2023-12-13

- Disable Bastion tests for `capa` provider.

## [1.20.1] - 2023-12-13

### Changed

- Update to new schema for `cluster-eks` values.

## [1.20.0] - 2023-12-11

### Changed

- Bump `clustertest` to v0.14.0 that increased the char count of cluster names to 20 chars
- CAPV: WCs have the default deny-all network policies

## [1.19.3] - 2023-12-05

### Fixed

- Increased timeout for observability-bundle apps.

## [1.19.2] - 2023-12-04

### Fixed

- Removed the old schema from the cluster-aws values

## [1.19.1] - 2023-12-04

### Fixed

- include both schema versions in CAPA cluster values file while migration is underway

## [1.19.0] - 2023-12-01

### Added

- Add initial support for new cluster app Helm values structure where root-level properties are moved to `.Values.global`.

## [1.18.0] - 2023-11-28

### Added

- Add `observability-bundle` test in the common/apps.go file.

## [1.17.1] - 2023-11-26

### Changed

- Re-enabled CAPVCD tests
- Small cleanup tasks - added some relevant comments, explicitly set false bools, added logging to scale test

## [1.17.0] - 2023-11-15

### Added

- Add new test provider EKS.
- Add test for autoscaling MachinePool nodes.

## [1.16.2] - 2023-11-13

### Changed

- Fix CAPV tests by switching to Flatcar image.

## [1.16.1] - 2023-11-03

### Changed

- Updated `clustertest` to v0.12.1` to fix upgrade tests.

## [1.16.0] - 2023-11-02

### Changed

- Updated `clustertest` to v0.12.0` that now creates a ServiceAccount in the workload cluster to use for authentication when communicating with the WC in tests and updates `GetExpectedControlPlaneReplicas` to handle managed clusters such as EKS (introduced in `v0.11.0`)

## [1.15.1] - 2023-11-02

## [1.15.0] - 2023-10-06

### Changed

- Vsphere now uses IPAM also for WC's Service LB (kubevip)
- Moved `helpers` into `internal` directory
- Moved `common` into `internal` directory
- Removed unused function
- Cleaned up Readme
- Updated `clustertest` to v0.10.0
- Change control plane node test to use new `GetExpectedControlPlaneReplicas` helper function

## [1.14.0] - 2023-10-05

### Changed

- Use Spot instances for CAPA tests.
- Use smaller volumes sizes for CAPA tests.
- Updated `clustertest` to v0.9.0.
- Make timeouts more relevant to the context of the test

## [1.13.0] - 2023-09-15

### Changed

- Updated `clustertest` to v0.7.0.
- Refactored app deployment to use `clustertest` helper functions.

## [1.12.1] - 2023-09-12

### Changed

- Refactored the various app status checks to make use of the helper functions in `clustertest`
- Updated `clustertest` to v0.4.0

## [1.12.0] - 2023-09-12

### Added

- Add test that deploys hello-world application.
- Add a cert-manager ClusterIssuer readiness test.

## [1.11.1] - 2023-08-31

### Changed

- Shrink container image by excluding build artifacts and language files.
- Rely on `cluster-aws` default values when installing the chart.

### Docs

- Updated readme info

### Changed

- Bumped `clustertest` to v0.3.1 with fix for `ClusterValues` unmarshalling if different `NodePool` types
- Refactored node tests to reuse code now the `NodePool` type fix is in `clustertest`

## [1.11.0] - 2023-08-29

### Added

- Generate test result reports

## [1.10.0] - 2023-08-29

### Changed

- Bumped clustertest to `v0.3.0` to be able to set the app `Organization`.
- Bumped clustertest to `v0.2.0` to use custom app values.
- Refactored default app tests so they can log out current state during iterations
- Log out the non-running pods in test spec to show progress
- Bump Ginkgo to v2.12.0

## [1.9.0] - 2023-08-21

### Added

- Add a PVC test.

### Fixed

- Bastion tests are disabled for CAPV.

### Changed

- Refactored state management to use singleton

## [1.8.0] - 2023-08-14

### Added

- Add common test to make sure all default apps are marked as `deployed`.
- Upgrade cluster tests

### Changed

- Bumped clustertest to `v0.1.1` to fix `GITHUB_TOKEN` issue.
- Bumped clustertest to `v0.2.0` to use custom app values.

### Added

- Upgrade cluster tests

## [1.7.0] - 2023-07-20

### Changed

- Abstract away the cluster app creation to a provider-specific function to handle provider specific requirements (such as VSphere credentials as extra config)
- Bump `clustertest` to v0.0.18

## [1.6.0] - 2023-07-20

### Added

- `standup` now supports specifying cluster app and default-apps app versions as arguments, defaulting to latest

## [1.5.0] - 2023-07-17

### Added

- CLI tool `standup` for creating test clusters outside of the test suite
- CLI tool `teardown` for deleting test clusters previously created

## [1.4.0] - 2023-07-14

### Added

- Support marking test suites as skipped / ignored by prefixing the directory with `X`
- Tests verifying control plane nodes and worker nodes are consistently Running
- Tests that verify cross-provider DNS records

### Changed

- Marked CAPV standard test suite as active (removed the static IP for control plane -> relying on IPAM).
- Marked CAPV standard test suite as skipped until it can reliably run the tests.
- Changed parameters for CAPVCD tests by applying latest schema changes.
- Changed parameters for CAPVCD tests for `0.12.0` release.

## [1.3.0] - 2023-05-30

## Added

- CAPV standard test suite

## [1.2.0] - 2023-05-12

### Changed

- Wait for Org to finish deleting when deleting cluster
- Increased Ginkgo suite timeout to 4 hour
- Continue running other remaining test suites when a suite fails

## [1.1.0] - 2023-04-19

### Docs

- Updated readme with instructions on adding tests

## Added

- Ability to run tests against an existing WC. To do this export E2E_WC_NAME and E2E_WC_NAMESPACE

## [1.0.2] - 2023-03-31

### Added

- Prebuild the test suites in the Docker image to speed up starting the tests

## [1.0.1] - 2023-03-31

### Changed

- Bumped custertest to v0.0.7

## [1.0.0] - 2023-03-30

### Added

- CAPA standard test suite
- CAPZ standard test suite
- CAPVCD standard test suite
- Example common tests
- Dockerfile for running tests in CI

[Unreleased]: https://github.com/giantswarm/cluster-test-suites/compare/v1.59.0...HEAD
[1.59.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.58.0...v1.59.0
[1.58.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.57.2...v1.58.0
[1.57.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.57.1...v1.57.2
[1.57.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.57.0...v1.57.1
[1.57.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.56.0...v1.57.0
[1.56.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.55.0...v1.56.0
[1.55.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.54.1...v1.55.0
[1.54.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.54.0...v1.54.1
[1.54.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.53.0...v1.54.0
[1.53.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.52.0...v1.53.0
[1.52.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.51.0...v1.52.0
[1.51.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.50.0...v1.51.0
[1.50.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.49.0...v1.50.0
[1.49.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.48.0...v1.49.0
[1.48.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.47.0...v1.48.0
[1.47.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.46.0...v1.47.0
[1.46.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.45.0...v1.46.0
[1.45.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.44.3...v1.45.0
[1.44.3]: https://github.com/giantswarm/cluster-test-suites/compare/v1.44.2...v1.44.3
[1.44.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.44.1...v1.44.2
[1.44.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.44.0...v1.44.1
[1.44.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.43.0...v1.44.0
[1.43.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.42.0...v1.43.0
[1.42.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.41.0...v1.42.0
[1.41.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.40.0...v1.41.0
[1.40.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.39.1...v1.40.0
[1.39.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.39.0...v1.39.1
[1.39.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.38.0...v1.39.0
[1.38.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.37.0...v1.38.0
[1.37.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.36.0...v1.37.0
[1.36.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.35.0...v1.36.0
[1.35.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.34.0...v1.35.0
[1.34.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.33.0...v1.34.0
[1.33.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.32.0...v1.33.0
[1.32.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.31.0...v1.32.0
[1.31.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.30.1...v1.31.0
[1.30.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.30.0...v1.30.1
[1.30.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.29.0...v1.30.0
[1.29.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.28.0...v1.29.0
[1.28.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.27.0...v1.28.0
[1.27.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.26.2...v1.27.0
[1.26.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.26.1...v1.26.2
[1.26.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.26.0...v1.26.1
[1.26.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.25.4...v1.26.0
[1.25.4]: https://github.com/giantswarm/cluster-test-suites/compare/v1.25.3...v1.25.4
[1.25.3]: https://github.com/giantswarm/cluster-test-suites/compare/v1.25.2...v1.25.3
[1.25.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.25.1...v1.25.2
[1.25.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.25.0...v1.25.1
[1.25.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.24.0...v1.25.0
[1.24.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.23.2...v1.24.0
[1.23.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.23.1...v1.23.2
[1.23.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.23.0...v1.23.1
[1.23.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.22.0...v1.23.0
[1.22.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.21.3...v1.22.0
[1.21.3]: https://github.com/giantswarm/cluster-test-suites/compare/v1.21.2...v1.21.3
[1.21.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.21.1...v1.21.2
[1.21.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.21.0...v1.21.1
[1.21.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.20.4...v1.21.0
[1.20.4]: https://github.com/giantswarm/cluster-test-suites/compare/v1.20.3...v1.20.4
[1.20.3]: https://github.com/giantswarm/cluster-test-suites/compare/v1.20.2...v1.20.3
[1.20.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.20.1...v1.20.2
[1.20.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.20.0...v1.20.1
[1.20.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.19.3...v1.20.0
[1.19.3]: https://github.com/giantswarm/cluster-test-suites/compare/v1.19.2...v1.19.3
[1.19.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.19.1...v1.19.2
[1.19.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.19.0...v1.19.1
[1.19.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.18.0...v1.19.0
[1.18.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.17.1...v1.18.0
[1.17.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.17.0...v1.17.1
[1.17.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.16.2...v1.17.0
[1.16.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.16.1...v1.16.2
[1.16.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.16.0...v1.16.1
[1.16.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.15.1...v1.16.0
[1.15.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.15.0...v1.15.1
[1.15.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.14.0...v1.15.0
[1.14.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.13.0...v1.14.0
[1.13.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.12.1...v1.13.0
[1.12.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.12.0...v1.12.1
[1.12.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.11.1...v1.12.0
[1.11.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.11.0...v1.11.1
[1.11.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.10.0...v1.11.0
[1.10.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.8.0...v1.9.0
[1.8.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.0.2...v1.1.0
[1.0.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/giantswarm/cluster-test-suites/releases/tag/v1.0.0
