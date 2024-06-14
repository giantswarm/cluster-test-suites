package standard

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
)

var _ = Describe("Common tests", Ordered, func() {
	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})
})
