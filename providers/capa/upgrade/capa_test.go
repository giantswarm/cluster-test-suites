package upgrade

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/ecr"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
	"github.com/giantswarm/cluster-test-suites/v7/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	BeforeEach(func() {
		// Set higher timeout for deploying apps because Karpenter workers take longer to come up
		state.SetTestTimeout(timeout.DeployApps, time.Minute*30)
	})

	upgrade.Run(upgrade.NewTestConfigWithDefaults())

	// Finally run the common tests after upgrade is completed
	common.Run(common.NewTestConfigWithDefaults())

	// ECR Credential Provider specific tests
	ecr.Run()
})
