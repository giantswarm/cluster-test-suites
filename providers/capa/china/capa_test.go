package china

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
	"github.com/giantswarm/cluster-test-suites/v3/internal/state"
	"github.com/giantswarm/cluster-test-suites/v3/internal/timeout"
)

var _ = Describe("Common tests", func() {
	BeforeEach(func() {
		// Set the timeout for deploying apps to 25 minutes for China test
		state.SetTestTimeout(timeout.DeployApps, time.Minute*25)
	})

	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})
})
