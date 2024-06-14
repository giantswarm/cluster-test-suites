package upgrade

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	// Run common tests before the upgrade is started, so we ensure that the cluster is fully ready
	// before we do the upgrade.
	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})

	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests again after upgrade is completed
	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})
})
