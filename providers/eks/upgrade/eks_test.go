package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	cfg := upgrade.NewTestConfigWithDefaults()
	cfg.ControlPlaneType = upgrade.ControlPlaneTypeAWSManaged
	upgrade.Run(cfg)

	// Finally run the common tests after upgrade is completed
	ccfg := common.NewTestConfigWithDefaults()
	// EKS does not have metrics for k8s control plane components.
	ccfg.ControlPlaneMetricsSupported = false
	ccfg.ExternalDnsSupported = false
	ccfg.AutoScalingSupported = false
	common.Run(ccfg)
})
