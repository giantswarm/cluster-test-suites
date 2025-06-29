package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests after upgrade is completed
	common.Run(&common.TestConfig{
		// Disabled until https://github.com/giantswarm/roadmap/issues/2693
		AutoScalingSupported: false,
		BastionSupported:     false,
		TeleportSupported:    true,
		// Disabled until wildcard ingress support is added
		ExternalDnsSupported:         false,
		ControlPlaneMetricsSupported: true,
	})
})
