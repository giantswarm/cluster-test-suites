package standard

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
)

var _ = XDescribe("Common tests", func() {
	common.Run(&common.TestConfig{
		AutoScalingSupported: true,
		BastionSupported:     false,
		ExternalDnsSupported: true,
		// EKS does not have metrics for k8s control plane components.
		ControlPlaneMetricsSupported: false,
	})
})
