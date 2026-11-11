package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/capz"
	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests after upgrade is completed
	cfg := common.NewTestConfigWithDefaults()
	cfg.AutoScalingSupported = capz.AutoScalingSupported()
	// Disabled until wildcard ingress support is added
	cfg.ExternalDnsSupported = false
	cfg.GatewayAPISupported = false
	common.Run(cfg)
})
