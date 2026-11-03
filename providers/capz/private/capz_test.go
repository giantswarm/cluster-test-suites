package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/capz"
	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
)

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	cfg.AutoScalingSupported = capz.AutoScalingSupported()
	// Disabled until wildcard ingress support is added
	cfg.ExternalDnsSupported = false
	cfg.GatewayAPISupported = false
	common.Run(cfg)
})
