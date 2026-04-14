package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v4/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v4/pkg/logger"
	"github.com/giantswarm/clustertest/v4/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v6/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v6/internal/state"
	"github.com/giantswarm/cluster-test-suites/v6/internal/timeout"
)

func RunApps(cfg *TestConfig) {
	Context("default apps and helm releases", func() {
		It("all HelmReleases are deployed without issues", func() {
			timeout := state.GetTestTimeout(timeout.DeployApps, 15*time.Minute)
			logger.Log("Waiting for all HelmReleases to be deployed. Timeout: %s", timeout.String())

			// Get all HelmReleases in the cluster organization namespace
			helmReleaseList := newUnstructuredHelmReleaseList()
			err := state.GetFramework().MC().List(state.GetContext(), helmReleaseList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()))
			Expect(err).NotTo(HaveOccurred())

			if len(helmReleaseList.Items) == 0 {
				logger.Log("No HelmReleases found in namespace %s", state.GetCluster().Organization.GetNamespace())
				return
			}

			helmReleaseNamespacedNames := []types.NamespacedName{}
			for _, hr := range helmReleaseList.Items {
				helmReleaseNamespacedNames = append(helmReleaseNamespacedNames, types.NamespacedName{Name: hr.GetName(), Namespace: hr.GetNamespace()})
			}

			Eventually(wait.Consistent(areAllHelmReleasesReady(state.GetContext(), state.GetFramework().MC(), helmReleaseNamespacedNames), 5, 10*time.Second)).
				WithTimeout(timeout).
				WithPolling(10*time.Second).
				Should(
					Succeed(),
					failurehandler.Bundle(
						failurehandler.HelmReleasesNotReady(state.GetFramework(), state.GetCluster()),
						failurehandler.PodsNotReady(state.GetFramework(), state.GetCluster()),
						reportHelmReleaseOwningTeams(),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate HelmReleases not ready"),
					),
				)
		})

		It("all default apps are deployed without issues", func() {
			if UsesHelmReleaseBasedDefaultApps() {
				Skip("Release deploys default apps as HelmReleases; the HelmRelease assertion covers this release.")
			}

			timeout := state.GetTestTimeout(timeout.DeployApps, 15*time.Minute)
			logger.Log("Waiting for all apps to be deployed. Timeout: %s", timeout.String())
			logger.Log("Checking default apps deployed from the unified %s app.", state.GetCluster().ClusterApp.AppName)

			// Wait for all default-apps apps to be deployed
			appList := &v1alpha1.AppList{}
			err := state.GetFramework().MC().List(state.GetContext(), appList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()), getDefaultAppsSelector())
			Expect(err).NotTo(HaveOccurred())

			appNamespacedNames := []types.NamespacedName{}
			for _, app := range appList.Items {
				appNamespacedNames = append(appNamespacedNames, types.NamespacedName{Name: app.Name, Namespace: app.Namespace})
			}

			Eventually(wait.IsAllAppDeployed(state.GetContext(), state.GetFramework().MC(), appNamespacedNames)).
				WithTimeout(timeout).
				WithPolling(10*time.Second).
				Should(
					BeTrue(),
					failurehandler.Bundle(
						failurehandler.AppIssues(state.GetFramework(), state.GetCluster()),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate Apps not ready"),
						reportOwningTeams(),
					),
				)
		})
	})
	Context("observability-bundle apps", func() {
		It("all observability-bundle apps are deployed without issues", func() {
			if !cfg.ObservabilityBundleInstalled {
				Skip("observability-bundle is not installed")
			}
			if UsesHelmReleaseBasedDefaultApps() {
				Skip("Release deploys default apps as HelmReleases; the HelmRelease observability-bundle assertion covers this release.")
			}

			helper.SetResponsibleTeam(helper.TeamAtlas)

			// We need to wait for the observability-bundle app to be deployed before we can check the apps it deploys.
			observabilityAppsAppName := fmt.Sprintf("%s-%s", state.GetCluster().Name, "observability-bundle")

			bundleTimeout := state.GetTestTimeout(timeout.BundleApps, 90*time.Second)
			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), observabilityAppsAppName, state.GetCluster().GetNamespace())).
				WithTimeout(bundleTimeout).
				WithPolling(5 * time.Second).
				Should(BeTrue())

			// Wait for all observability-bundle apps to be deployed
			appList := &v1alpha1.AppList{}
			err := state.GetFramework().MC().List(state.GetContext(), appList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()), ctrl.MatchingLabels{"giantswarm.io/managed-by": observabilityAppsAppName})
			Expect(err).NotTo(HaveOccurred())

			appNamespacedNames := []types.NamespacedName{}
			for _, app := range appList.Items {
				appNamespacedNames = append(appNamespacedNames, types.NamespacedName{Name: app.Name, Namespace: app.Namespace})
			}

			Eventually(wait.IsAllAppDeployed(state.GetContext(), state.GetFramework().MC(), appNamespacedNames)).
				WithTimeout(8*time.Minute).
				WithPolling(10*time.Second).
				Should(
					BeTrue(),
					failurehandler.AppIssues(state.GetFramework(), state.GetCluster()),
				)
		})

		It("all observability-bundle HelmReleases are deployed without issues", func() {
			if !cfg.ObservabilityBundleInstalled {
				Skip("observability-bundle is not installed")
			}
			if !UsesHelmReleaseBasedDefaultApps() {
				Skip("Release deploys default apps as App CRs; the App-CR observability-bundle assertion covers this release.")
			}

			helper.SetResponsibleTeam(helper.TeamAtlas)

			parent := fmt.Sprintf("%s-%s", state.GetCluster().Name, "observability-bundle")
			waitForBundleHelmReleases(parent, 8*time.Minute)
		})
	})
	Context("security-bundle apps", func() {
		It("all security-bundle apps are deployed without issues", func() {
			if !cfg.SecurityBundleInstalled {
				Skip("security-bundle is not installed")
			}
			if UsesHelmReleaseBasedDefaultApps() {
				Skip("Release deploys default apps as HelmReleases; the HelmRelease security-bundle assertion covers this release.")
			}

			helper.SetResponsibleTeam(helper.TeamShield)

			// We need to wait for the security-bundle app to be deployed before we can check the apps it deploys.
			securityAppsAppName := fmt.Sprintf("%s-%s", state.GetCluster().Name, "security-bundle")

			bundleTimeout := state.GetTestTimeout(timeout.BundleApps, 90*time.Second)
			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), securityAppsAppName, state.GetCluster().GetNamespace())).
				WithTimeout(bundleTimeout).
				WithPolling(5 * time.Second).
				Should(BeTrue())

			// Wait for all security-bundle apps to be deployed
			appList := &v1alpha1.AppList{}
			err := state.GetFramework().MC().List(state.GetContext(), appList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()), ctrl.MatchingLabels{"giantswarm.io/managed-by": securityAppsAppName})
			Expect(err).NotTo(HaveOccurred())

			appNamespacedNames := []types.NamespacedName{}
			for _, app := range appList.Items {
				appNamespacedNames = append(appNamespacedNames, types.NamespacedName{Name: app.Name, Namespace: app.Namespace})
			}

			Eventually(wait.IsAllAppDeployed(state.GetContext(), state.GetFramework().MC(), appNamespacedNames)).
				WithTimeout(10*time.Minute).
				WithPolling(10*time.Second).
				Should(
					BeTrue(),
					failurehandler.AppIssues(state.GetFramework(), state.GetCluster()),
				)
		})

		It("all security-bundle HelmReleases are deployed without issues", func() {
			if !cfg.SecurityBundleInstalled {
				Skip("security-bundle is not installed")
			}
			if !UsesHelmReleaseBasedDefaultApps() {
				Skip("Release deploys default apps as App CRs; the App-CR security-bundle assertion covers this release.")
			}

			helper.SetResponsibleTeam(helper.TeamShield)

			parent := fmt.Sprintf("%s-%s", state.GetCluster().Name, "security-bundle")
			waitForBundleHelmReleases(parent, 10*time.Minute)
		})
	})
}

// waitForBundleHelmReleases waits for the named parent bundle HelmRelease to be
// Ready and then for all its child HelmReleases (selected by the
// giantswarm.io/managed-by=<parent> label, same convention as the App-CR
// variant) to be Ready too. childrenTimeout bounds the children's wait; the
// parent uses the shared BundleApps timeout (default 90s) to match the
// App-based sibling's behaviour.
func waitForBundleHelmReleases(parentName string, childrenTimeout time.Duration) {
	mc := state.GetFramework().MC()
	org := state.GetCluster().Organization.GetNamespace()

	parentTimeout := state.GetTestTimeout(timeout.BundleApps, 90*time.Second)
	Eventually(WaitHelmReleaseReady(state.GetContext(), mc, parentName, org)).
		WithTimeout(parentTimeout).
		WithPolling(5 * time.Second).
		Should(BeTrue())

	helmReleaseList := newUnstructuredHelmReleaseList()
	err := mc.List(state.GetContext(), helmReleaseList, ctrl.InNamespace(org), ctrl.MatchingLabels{"giantswarm.io/managed-by": parentName})
	Expect(err).NotTo(HaveOccurred())

	children := make([]types.NamespacedName, 0, len(helmReleaseList.Items))
	for _, hr := range helmReleaseList.Items {
		children = append(children, types.NamespacedName{Name: hr.GetName(), Namespace: hr.GetNamespace()})
	}

	Eventually(wait.Consistent(areAllHelmReleasesReady(state.GetContext(), mc, children), 5, 10*time.Second)).
		WithTimeout(childrenTimeout).
		WithPolling(10*time.Second).
		Should(
			Succeed(),
			failurehandler.Bundle(
				failurehandler.HelmReleasesNotReady(state.GetFramework(), state.GetCluster()),
				reportHelmReleaseOwningTeams(),
			),
		)
}

func getDefaultAppsSelector() ctrl.MatchingLabels {
	// All providers now use unified cluster apps that deploy default apps directly via Helm
	return ctrl.MatchingLabels{
		"giantswarm.io/cluster":        state.GetCluster().Name,
		"app.kubernetes.io/managed-by": "Helm",
	}
}

func reportOwningTeams() failurehandler.FailureHandler {
	return failurehandler.Wrap(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		logger.Log("Attempting to get responsible teams for any failing Apps")

		appList := &v1alpha1.AppList{}
		err := state.GetFramework().MC().List(ctx, appList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()), getDefaultAppsSelector())
		if err != nil {
			logger.Log("Failed to get Apps - %v", err)
			return
		}

		for _, app := range appList.Items {
			if app.Status.Release.Status != "deployed" {
				teamLabel, ok := app.Annotations[annotation.AppTeam]
				if ok && !helper.SetResponsibleTeamFromLabel(teamLabel) {
					logger.Log("Unknown owner team - App='%s', TeamLabel='%s'", app.Name, teamLabel)
				}
			}
		}

	})
}

// areAllHelmReleasesReady checks if all HelmReleases in the list are ready
func areAllHelmReleasesReady(ctx context.Context, client ctrl.Client, helmReleases []types.NamespacedName) func() error {
	return func() error {
		allReady := true
		for _, hr := range helmReleases {
			helmRelease := newUnstructuredHelmRelease()
			err := client.Get(ctx, hr, helmRelease)
			if err != nil {
				logger.Log("HelmRelease status for '%s' failed to retrieve: %v", hr.Name, err)
				allReady = false
				continue
			}

			ready, reason, message := getHelmReleaseReadyCondition(helmRelease)
			if ready {
				logger.Log("HelmRelease status for '%s' is as expected: expectedStatus='Ready' actualStatus='Ready'", hr.Name)
			} else if reason != "" {
				logger.Log("HelmRelease status for '%s' is not yet as expected: expectedStatus='Ready' actualStatus='%s' (reason: '%s')", hr.Name, reason, message)
				allReady = false
			} else {
				logger.Log("HelmRelease status for '%s' is not yet as expected: expectedStatus='Ready' actualStatus='Unknown' (reason: 'No Ready condition found')", hr.Name)
				allReady = false
			}
		}

		if !allReady {
			return fmt.Errorf("not all HelmReleases are ready")
		}
		return nil
	}
}

// reportHelmReleaseOwningTeams reports the teams responsible for failing HelmReleases
func reportHelmReleaseOwningTeams() failurehandler.FailureHandler {
	return failurehandler.Wrap(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		logger.Log("Attempting to get responsible teams for any failing HelmReleases")

		helmReleaseList := newUnstructuredHelmReleaseList()
		err := state.GetFramework().MC().List(ctx, helmReleaseList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()))
		if err != nil {
			logger.Log("Failed to get HelmReleases - %v", err)
			return
		}

		for _, hr := range helmReleaseList.Items {
			ready, _, _ := getHelmReleaseReadyCondition(&hr)
			if !ready {
				labels := hr.GetLabels()
				if labels != nil {
					if teamLabel, ok := labels["application.giantswarm.io/team"]; ok && !helper.SetResponsibleTeamFromLabel(teamLabel) {
						logger.Log("Unknown owner team - HelmRelease='%s', TeamLabel='%s'", hr.GetName(), teamLabel)
					}
				}
			}
		}
	})
}
