package standard

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v6/internal/common"
	"github.com/giantswarm/cluster-test-suites/v6/internal/ecr"
	"github.com/giantswarm/cluster-test-suites/v6/internal/state"
	"github.com/giantswarm/cluster-test-suites/v6/internal/timeout"
)

var _ = Describe("Common tests", func() {
	BeforeEach(func() {
		// Set higher timeout for deploying apps because Karpenter workers take longer to come up
		state.SetTestTimeout(timeout.DeployApps, time.Minute*30)
	})

	common.Run(common.NewTestConfigWithDefaults())

	// ECR Credential Provider specific tests
	ecr.Run()
})
