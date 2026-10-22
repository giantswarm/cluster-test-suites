package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
)

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	// EKS does not have metrics for k8s control plane components.
	cfg.ControlPlaneMetricsSupported = false
	// EKS doesn't have most of the Giant Swarm apps deployed, but the
	// security-bundle is part of EKS releases.
	cfg.ObservabilityBundleInstalled = false
	cfg.ExternalDnsSupported = false
	cfg.AutoScalingSupported = false
	cfg.CertManagerSupported = false
	cfg.GatewayAPISupported = false
	common.Run(cfg)
})
