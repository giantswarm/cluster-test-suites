package standard

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/internal/timeout"
)

var _ = Describe("Common tests", func() {
	BeforeEach(func() {
		// Set higher timeout for deploying apps for karpenter test because workers take longer to come up
		state.SetTestTimeout(timeout.DeployApps, time.Minute*30)
	})

	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})
})
