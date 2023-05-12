# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/giantswarm/cluster-test-suites/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/giantswarm/cluster-test-suites/compare/v1.0.2...v1.1.0
[1.0.2]: https://github.com/giantswarm/cluster-test-suites/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/giantswarm/cluster-test-suites/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/giantswarm/cluster-test-suites/releases/tag/v1.0.0
