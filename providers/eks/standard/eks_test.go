package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
)

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	// EKS does not have metrics for k8s control plane components.
	cfg.ControlPlaneMetricsSupported = false
	// EKS doesn't have any of the Giant Swarm apps deployed
	cfg.MinimalCluster = true
	cfg.ExternalDnsSupported = false
	cfg.AutoScalingSupported = false
	common.Run(cfg)
})
