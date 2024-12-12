package standard

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
)

var _ = XDescribe("Common tests", func() {
	common.Run(&common.TestConfig{
		// No autoscaling on-prem
		AutoScalingSupported: false,
		BastionSupported:     false,
		TeleportSupported:    true,
		// Disabled until https://github.com/giantswarm/roadmap/issues/1037
		ExternalDnsSupported:         false,
		ControlPlaneMetricsSupported: true,
	})
})
