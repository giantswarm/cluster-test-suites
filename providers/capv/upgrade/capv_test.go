package upgrade

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests after upgrade is completed
	common.Run(&common.TestConfig{
		// No autoscaling on-prem
		AutoScalingSupported: false,
		BastionSupported:     false,
		TeleportSupported:    true,
		// Disabled until https://github.com/giantswarm/roadmap/issues/1037
		ExternalDnsSupported: false,
	})
})
