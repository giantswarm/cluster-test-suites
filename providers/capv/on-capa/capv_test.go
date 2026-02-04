package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
)

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	// No autoscaling on-prem
	cfg.AutoScalingSupported = false
	// Disabled until https://github.com/giantswarm/roadmap/issues/1037
	cfg.ExternalDnsSupported = false

	common.Run(cfg)
})
