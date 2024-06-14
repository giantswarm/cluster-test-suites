package china

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	"time"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

var _ = Describe("Common tests", func() {
	BeforeEach(func() {
		// Set the timeout for deploying apps to 25 minutes for China test
		state.SetContext(context.WithValue(state.GetContext(), common.DeployAppsTimeoutContextKey, time.Minute*25))
	})

	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})
})
