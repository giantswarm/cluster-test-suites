package suite

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cb "github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder"
	"github.com/giantswarm/cluster-standup-teardown/pkg/standup"
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"

	"github.com/giantswarm/cluster-test-suites/internal/helper"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

// Setup handles the creation of the BeforeSuite and AfterSuite handlers. This covers the creations and cleanup of the test cluster.
// `clusterReadyFns` can be provided if the cluster requires custom checks for cluster-ready status. If not provided the cluster will
// be checked for at least a single control plane node being marked as ready.
func Setup(isUpgrade bool, clusterBuilder cb.ClusterBuilder, clusterReadyFns ...func(client *client.Client)) {
	BeforeSuite(func() {
		if isUpgrade && helper.ShouldSkipUpgrade() {
			Skip("E2E_OVERRIDE_VERSIONS env var not set, skipping upgrade test")
			return
		}

		logger.LogWriter = GinkgoWriter

		state.SetContext(context.Background())

		framework, err := clustertest.New(clusterBuilder.KubeContext())
		Expect(err).NotTo(HaveOccurred())
		state.SetFramework(framework)

		cluster := cb.LoadOrBuildCluster(framework, clusterBuilder)
		state.SetCluster(cluster)

		cluster, err = standup.New(framework, isUpgrade, clusterReadyFns...).Standup(cluster)
		Expect(err).NotTo(HaveOccurred())
		state.SetCluster(cluster)
	})

	AfterSuite(func() {
		if isUpgrade && helper.ShouldSkipUpgrade() {
			return
		}

		// Ensure we reset the context timeout to make sure we allow plenty of time to clean up
		ctx := state.GetContext()
		ctx, _ = context.WithTimeout(ctx, 1*time.Hour)
		state.SetContext(ctx)

		Expect(state.GetFramework().DeleteCluster(state.GetContext(), state.GetCluster())).To(Succeed())
	})
}
