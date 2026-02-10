package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
	"github.com/giantswarm/cluster-test-suites/v3/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests after upgrade is completed
	cfg := common.NewTestConfigWithDefaults()
	// Disabled until https://github.com/giantswarm/roadmap/issues/2693
	cfg.AutoScalingSupported = false
	// Disabled until wildcard ingress support is added
	cfg.ExternalDnsSupported = false
	common.Run(cfg)
})
