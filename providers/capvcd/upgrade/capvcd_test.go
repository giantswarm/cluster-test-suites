package upgrade

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
	"github.com/giantswarm/cluster-test-suites/v3/internal/state"
	"github.com/giantswarm/cluster-test-suites/v3/internal/timeout"
	"github.com/giantswarm/cluster-test-suites/v3/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	BeforeEach(func() {
		// Set the timeout for `ClusterReady` check to 40 minutes when upgrading the cluster
		state.SetTestTimeout(timeout.ClusterReadyTimeout, time.Minute*40)
	})
	// it is better to get defaults at first and then customize
	// further changes in defaults will be effective here.
	cfg := upgrade.NewTestConfigWithDefaults()
	cfg.ControlPlaneNodesTimeout = 30 * time.Minute
	cfg.WorkerNodesTimeout = 30 * time.Minute

	upgrade.Run(cfg)

	// Finally run the common tests after upgrade is completed
	ccfg := common.NewTestConfigWithDefaults()
	// No autoscaling on-prem
	ccfg.AutoScalingSupported = false
	// Disabled until https://github.com/giantswarm/roadmap/issues/1037
	ccfg.ExternalDnsSupported = false
	common.Run(ccfg)
})
