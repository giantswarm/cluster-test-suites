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
		state.SetContext(context.WithValue(state.GetContext(), "deployedAppsTimeout", time.Minute*20))
	})

	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})
})
