package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v4/internal/common"
	"github.com/giantswarm/cluster-test-suites/v4/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests after upgrade is completed
	cfg := common.NewTestConfigWithDefaults()
	// No autoscaling on-prem
	cfg.AutoScalingSupported = false
	// Disabled until https://github.com/giantswarm/roadmap/issues/1037
	cfg.ExternalDnsSupported = false
	common.Run(cfg)
})
