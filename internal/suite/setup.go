package suite

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cb "github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder"
	"github.com/giantswarm/cluster-standup-teardown/pkg/standup"
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/env"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/utils"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

// Setup handles the creation of the BeforeSuite and AfterSuite handlers. This covers the creations and cleanup of the test cluster.
// `clusterReadyFns` can be provided if the cluster requires custom checks for cluster-ready status. If not provided the cluster will
// be checked for at least a single control plane node being marked as ready.
func Setup(isUpgrade bool, provider string, clusterBuilder cb.ClusterBuilder, clusterReadyFns ...func(client *client.Client)) {
	BeforeSuite(func() {
		if isUpgrade {
			if os.Getenv(env.OverrideVersions) == "" {
				// Try to automatically detect upgrade versions (with cross-major logic)
				from, to, err := utils.GetUpgradeReleasesToTest(provider)
				if err != nil {
					Skip(fmt.Sprintf("failed to get upgrade releases to test: %s", err))
					return
				}
				os.Setenv(env.ReleasePreUpgradeVersion, from)
				os.Setenv(env.ReleaseVersion, to)
			}
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
				} else {
					if len(events.Items) == 0 {
						logger.Log("No events found for Cluster App")
					}
					for _, event := range events.Items {
						logger.Log("Event: Reason='%s', Message='%s', Last Occurred='%v'", event.Reason, event.Message, event.LastTimestamp)
					}
				}

				if clusterApp.Status.Release.Status == "deployed" {
					logger.Log("Getting Cluster CR for the Cluster App")
					cl := &capi.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name:      cluster.Name,
							Namespace: cluster.GetNamespace(),
						},
					}

					err = framework.MC().Get(ctx, cr.ObjectKeyFromObject(cl), cl)
					if err != nil {
						logger.Log("Failed to get Cluster CR: %v", err)
					} else {
						logger.Log(
							"Cluster status: Phase='%s', InfrastructureReady='%t', ControlPlaneReady='%t', FailureReason='%s', FailureMessage='%s'",
							cl.Status.Phase,
							cl.Status.InfrastructureReady,
							cl.Status.ControlPlaneReady,
							ptr.Deref(cl.Status.FailureReason, ""),  //nolint:staticcheck // Ignore SA1019 this field is marked as deprecated.
							ptr.Deref(cl.Status.FailureMessage, ""), //nolint:staticcheck // Ignore SA1019 this field is marked as deprecated.
						)

						for _, condition := range cl.Status.Conditions {
							logger.Log("Cluster condition with type '%s' and status '%s' - Message='%s', Reason='%s', Last Occurred='%v'", condition.Type, condition.Status, condition.Message, condition.Reason, condition.LastTransitionTime)
						}
					}
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
		ctx, _ = context.WithTimeout(ctx, 1*time.Hour) //nolint:govet
		state.SetContext(ctx)

		err := cleanupPVs(ctx)
		if err != nil {
			logger.Log("Failed to cleanup PVs before delete - %v", err)
		}

		Expect(state.GetFramework().DeleteCluster(state.GetContext(), state.GetCluster())).To(Succeed())
	})
}

func cleanupPVs(ctx context.Context) error {
	logger.Log("Ensuring all PVs are cleaned up before deleting cluster")
	wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
	if err != nil {
		logger.Log("Failed to get WC client, skipping PV cleanup - %v", err)
		return err
	}

	pvs := &corev1.PersistentVolumeList{}
	err = wcClient.List(ctx, pvs, &cr.ListOptions{})
	if err != nil {
		logger.Log("Failed to list all PVs - %v", err)
		return err
	}
	logger.Log("Attempting to clean up %d PVs", len(pvs.Items))

	for _, pv := range pvs.Items {
		logger.Log("Deleting PV '%s'...", pv.Name)
		logger.Log("%v", pv)
		err := wcClient.Delete(state.GetContext(), &pv, &cr.DeleteOptions{})
		if err != nil && !apierror.IsNotFound(err) {
			logger.Log("Failed to delete PV '%s' - %v", pv.Name, err)
		}

		err = wait.For(
			wait.IsResourceDeleted(ctx, wcClient, &pv),
			wait.WithContext(ctx),
			wait.WithTimeout(5*time.Minute),
			wait.WithInterval(wait.DefaultInterval),
		)
		if err != nil {
			logger.Log("Failed to delete PV '%s' - %v", pv.Name, err)
		}
	}

	return nil
}
