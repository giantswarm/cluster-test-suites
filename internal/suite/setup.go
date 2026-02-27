package suite

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/giantswarm/clustertest/v3/pkg/failurehandler"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cb "github.com/giantswarm/cluster-standup-teardown/v4/pkg/clusterbuilder"
	"github.com/giantswarm/cluster-standup-teardown/v4/pkg/standup"
	"github.com/giantswarm/clustertest/v3"
	"github.com/giantswarm/clustertest/v3/pkg/client"
	"github.com/giantswarm/clustertest/v3/pkg/env"
	"github.com/giantswarm/clustertest/v3/pkg/logger"
	"github.com/giantswarm/clustertest/v3/pkg/utils"
	"github.com/giantswarm/clustertest/v3/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/v4/internal/state"
)

const (
	OpenAIAPIKeySecretNamespace = "giantswarm"
	OpenAIAPIKeySecretName      = "openai-api-key"
)

// Setup handles the creation of the BeforeSuite and AfterSuite handlers. This covers the creations and cleanup of the test cluster.
// `clusterReadyFns` can be provided if the cluster requires custom checks for cluster-ready status. If not provided the cluster will
// be checked for at least a single control plane node being marked as ready.
func Setup(isUpgrade bool, clusterBuilder cb.ClusterBuilder, clusterReadyFns ...func(client *client.Client)) {
	BeforeSuite(func() {
		logger.LogWriter = GinkgoWriter
		state.SetContext(context.Background())

		if isUpgrade {
			overrideVersions := strings.TrimSpace(os.Getenv(env.OverrideVersions))
			if overrideVersions == "" {
				// We're not using override versions, so we must be using release versions.
				// We call GetUpgradeReleasesToTest to resolve the 'from' and 'to' versions.
				// This function also handles the "previous_major" magic value.
				provider, err := getProviderFromBuilder(clusterBuilder)
				if err != nil {
					Fail(fmt.Sprintf("failed to get provider from cluster builder: %s", err))
				}
				from, to, err := utils.GetUpgradeReleasesToTest(provider)
				if err != nil {
					Skip(fmt.Sprintf("failed to get upgrade releases to test: %s", err))
					return
				}

				if to == "" {
					// If there's no target release 'to', we can't run an upgrade test.
					// This is the expected case for PRs to this repo which don't have release context.
					Skip("Skipping upgrade test as no release version was provided")
					return
				}

				// Check if we were looking for a previous major release but got an empty 'from' version.
				// This happens when we're drafting the first release of a new major version.
				if from == "" {
					preUpgrade := os.Getenv(env.ReleasePreUpgradeVersion)
					if preUpgrade == "previous_major" || preUpgrade == "first_previous_major" {
						Skip("Skipping upgrade test as this is the first release of a new major version")
					} else {
						Skip("Skipping standard upgrade test as no previous release was found in this major version")
					}
					return
				}

				// Set the concrete, resolved versions back into the environment, so that they can be
				// picked up by the cluster-standup-teardown library.
				os.Setenv(env.ReleasePreUpgradeVersion, from)
				os.Setenv(env.ReleaseVersion, to)
			}
		}

		framework, err := clustertest.New(clusterBuilder.KubeContext())
		Expect(err).NotTo(HaveOccurred())
		state.SetFramework(framework)

		cluster := cb.LoadOrBuildCluster(framework, clusterBuilder)
		state.SetCluster(cluster)

		// We'll use this to track if the BeforeSuite failed and if we should do extra debug logging
		setupComplete := false
		defer (func() {
			if !setupComplete {
				// If we fail to standup the cluster, let's grab the status of the cluster App to see if there's an error
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

						// Trigger LLM investigation for cluster setup failure
						logger.Log("Triggering LLM investigation for cluster setup failure")
						handler := failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate cluster setup failure in BeforeSuite")
						if handlerFunc, ok := handler.(func() string); ok {
							logger.Log("LLM investigation failure message: %s", handlerFunc())
						} else {
							logger.Log("Failed to cast failure handler to expected type")
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
		// Only run cleanup if framework and cluster were actually initialized
		// This prevents panics when BeforeSuite skips for any reason (PRs, first major releases, etc.)
		if state.GetFramework() == nil || state.GetCluster() == nil {
			logger.Log("Skipping cleanup as cluster/framework were not initialized")
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

func getProviderFromBuilder(clusterBuilder cb.ClusterBuilder) (string, error) {
	t := reflect.TypeOf(clusterBuilder)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return getProviderFromBuilderLogic(t.PkgPath(), t.Name())
}

func getProviderFromBuilderLogic(pkgPath, structName string) (string, error) {
	// The full type name includes the package path, e.g., "github.com/giantswarm/cluster-test-suites/providers/capa.StandardBuilder"
	// We want to extract the provider part, which is the directory name after "providers/".
	parts := strings.Split(pkgPath, "/")
	if len(parts) > 2 && parts[len(parts)-2] == "providers" {
		provider := parts[len(parts)-1]

		// The EKS test suite uses the CAPA provider builders, but is considered the "eks" provider.
		// We can detect this by checking for the unique builder struct name and that it comes from the capa provider.
		if structName == "ManagedClusterBuilder" && provider == "capa" {
			return "eks", nil
		}

		// The CAPZ test suite has a different provider name.
		if provider == "capz" {
			return "azure", nil
		}
		// The CAPV test suite has a different provider name.
		if provider == "capv" {
			return "vsphere", nil
		}
		// The CAPVCD test suite has a different provider name.
		if provider == "capvcd" {
			return "cloud-director", nil
		}
		return provider, nil
	}

	return "", fmt.Errorf("could not determine provider from package path: %s", pkgPath)
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
