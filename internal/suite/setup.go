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
	"github.com/giantswarm/clustertest/pkg/utils"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

// Setup handles the creation of the BeforeSuite and AfterSuite handlers. This covers the creations and cleanup of the test cluster.
// `clusterReadyFns` can be provided if the cluster requires custom checks for cluster-ready status. If not provided the cluster will
// be checked for at least a single control plane node being marked as ready.
func Setup(isUpgrade bool, clusterBuilder cb.ClusterBuilder, clusterReadyFns ...func(client *client.Client)) {
	BeforeSuite(func() {
		if isUpgrade && utils.ShouldSkipUpgrade() {
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

		// We'll use this to track if the BeforeSuite failed and if we should do extra debug logging
		setupComplete := false
		defer (func() {
			if !setupComplete {
				// If we fail to standup the cluster, lets grab the status of the cluster App to see if there's an error
				ctx := context.Background()
				ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
				defer cancel()

				cluster := state.GetCluster()

				logger.Log("Attempting to get debug info for Cluster App")

				clusterApp, err := framework.GetApp(ctx, cluster.ClusterApp.InstallName, cluster.ClusterApp.GetNamespace())
				if err != nil {
					logger.Log("Failed to get Cluster App: %v", err)
					return
				}

				logger.Log(
					"Cluster App status: AppVersion='%s', Version='%s', ReleaseStatus='%s', ReleaseReason='%s', LastDeployed='%v'",
					clusterApp.Status.AppVersion,
					clusterApp.Status.Version,
					clusterApp.Status.Release.Status,
					clusterApp.Status.Release.Reason,
					clusterApp.Status.Release.LastDeployed,
				)

				logger.Log("Getting events for the Cluster App")
				events, err := framework.MC().GetEventsForResource(ctx, clusterApp)
				if err != nil {
					logger.Log("Failed to get events for App: %v", err)
					return
				}
				if len(events.Items) == 0 {
					logger.Log("No events found for Cluster App")
				}
				for _, event := range events.Items {
					logger.Log("Event: Reason='%s', Message='%s', Last Occurred='%v'", event.Reason, event.Message, event.LastTimestamp)
				}
			}
		})()

		cluster, err = standup.New(framework, isUpgrade, clusterReadyFns...).Standup(cluster)
		Expect(err).NotTo(HaveOccurred())
		state.SetCluster(cluster)

		// Make sure this comes last
		setupComplete = true
	})

	AfterSuite(func() {
		if isUpgrade && utils.ShouldSkipUpgrade() {
			return
		}

		// Ensure we reset the context timeout to make sure we allow plenty of time to clean up
		ctx := state.GetContext()
		ctx, _ = context.WithTimeout(ctx, 1*time.Hour)
		state.SetContext(ctx)

		Expect(state.GetFramework().DeleteCluster(state.GetContext(), state.GetCluster())).To(Succeed())
	})
}
