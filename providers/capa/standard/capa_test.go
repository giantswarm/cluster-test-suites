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
	// Tie the net-exporter / cert-exporter pod-check exclusions to the same release-version
	// gate that decides whether the arm64 node pool is applied (see capa_suite_test.go).
	// Older releases don't get the arm pool, and shouldn't apply the exclusions either.
	// TODO(arm64): drop this gate once v35.0.0 is the minimum release across CI.
	// https://github.com/giantswarm/roadmap/issues/4302
	cfg.ARMNodePoolEnabled = armSupported()
	common.Run(cfg)

	// ECR Credential Provider specific tests
	ecr.Run()

})
