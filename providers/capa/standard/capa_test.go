package standard

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/ecr"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

var _ = Describe("Common tests", func() {
	BeforeEach(func() {
		// Set higher timeout for deploying apps because Karpenter workers take longer to come up
		state.SetTestTimeout(timeout.DeployApps, time.Minute*30)
	})

	cfg := common.NewTestConfigWithDefaults()
	// This suite runs an arm64 node pool. net-exporter and cert-exporter aren't multi-arch
	// in the currently-released app versions, so exclude their pods from the health checks.
	// TODO(arm64): remove once release v35 ships the multi-arch versions. https://github.com/giantswarm/roadmap/issues/4302
	cfg.ARMNodePoolEnabled = true
	common.Run(cfg)

	// ECR Credential Provider specific tests
	ecr.Run()
})
