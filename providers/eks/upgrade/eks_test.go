package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v4/internal/common"
	"github.com/giantswarm/cluster-test-suites/v4/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	cfg := upgrade.NewTestConfigWithDefaults()
	cfg.ControlPlaneType = upgrade.ControlPlaneTypeAWSManaged
	// EKS doesn't have any of the Giant Swarm apps deployed
	cfg.ObservabilityBundleInstalled = false
	cfg.SecurityBundleInstalled = false
	upgrade.Run(cfg)

	// Finally run the common tests after upgrade is completed
	ccfg := common.NewTestConfigWithDefaults()
	// EKS does not have metrics for k8s control plane components.
	ccfg.ControlPlaneMetricsSupported = false
	// EKS doesn't have any of the Giant Swarm apps deployed
	ccfg.ObservabilityBundleInstalled = false
	ccfg.SecurityBundleInstalled = false
	ccfg.ExternalDnsSupported = false
	ccfg.AutoScalingSupported = false
	ccfg.CertManagerSupported = false
	common.Run(ccfg)
})
