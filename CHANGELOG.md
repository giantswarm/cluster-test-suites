# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Add initial support for new cluster app Helm values structure where root-level properties are moved to `.Values.global`.

## [1.18.0] - 2023-11-28

### Added

- Add `observability-bundle` test in the common/apps.go file.

## [1.17.1] - 2023-11-26

### Changed

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

[Unreleased]: https://github.com/giantswarm/cluster-test-suites/compare/v1.18.0...HEAD
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
