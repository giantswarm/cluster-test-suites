package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/upgrade"
)

var _ = XDescribe("Basic upgrade test", Ordered, func() {
	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests after upgrade is completed
	common.Run(&common.TestConfig{
		AutoScalingSupported: true,
		BastionSupported:     false,
		ExternalDnsSupported: true,
		// EKS does not have metrics for k8s control plane components.
		ControlPlaneMetricsSupported: false,
	})
})
