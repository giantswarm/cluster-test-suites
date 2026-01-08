package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
)

var _ = Describe("Common tests", func() {
	common.Run(&common.TestConfig{
		AutoScalingSupported: true,
		BastionSupported:     false,
		ExternalDnsSupported: true,
		// EKS does not have metrics for k8s control plane components.
		ControlPlaneMetricsSupported: false,
	})
})
