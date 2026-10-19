package standard

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

var _ = Describe("Common tests", func() {
	BeforeEach(func() {
		// Set higher timeout for deploying apps because Karpenter workers take longer to come up
		state.SetTestTimeout(timeout.DeployApps, time.Minute*30)
	})

	cfg := common.NewTestConfigWithDefaults()
	// EKS does not have metrics for k8s control plane components.
	cfg.ControlPlaneMetricsSupported = false
	// EKS doesn't have any of the Giant Swarm apps deployed
	cfg.ObservabilityBundleInstalled = false
	cfg.SecurityBundleInstalled = false
	cfg.ExternalDnsSupported = false
	cfg.CertManagerSupported = false
	cfg.GatewayAPISupported = false
	common.Run(cfg)
})
