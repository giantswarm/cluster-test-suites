package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
)

var _ = XDescribe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	// No autoscaling on-prem
	cfg.AutoScalingSupported = false
	common.Run(cfg)
})
