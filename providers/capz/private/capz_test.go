package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
)

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	// Disabled until https://github.com/giantswarm/roadmap/issues/2693
	cfg.AutoScalingSupported = false
	common.Run(cfg)
})
