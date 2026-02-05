package common

import (
	"context"
	"fmt"
	"time"

	helm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v3/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v3/pkg/logger"
	"github.com/giantswarm/clustertest/v3/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v3/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v3/internal/state"
	"github.com/giantswarm/cluster-test-suites/v3/internal/timeout"
)

func RunApps(cfg *TestConfig) {
	Context("default apps and helm releases", func() {
		It("all HelmReleases are deployed without issues", func() {
			timeout := state.GetTestTimeout(timeout.DeployApps, 15*time.Minute)
			logger.Log("Waiting for all HelmReleases to be deployed. Timeout: %s", timeout.String())

			// Get all HelmReleases in the cluster organization namespace
			helmReleaseList := &helm.HelmReleaseList{}
			err := state.GetFramework().MC().List(state.GetContext(), helmReleaseList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()))
			Expect(err).NotTo(HaveOccurred())

			if len(helmReleaseList.Items) == 0 {
				logger.Log("No HelmReleases found in namespace %s", state.GetCluster().Organization.GetNamespace())
				return
			}

			helmReleaseNamespacedNames := []types.NamespacedName{}
			for _, hr := range helmReleaseList.Items {
				helmReleaseNamespacedNames = append(helmReleaseNamespacedNames, types.NamespacedName{Name: hr.Name, Namespace: hr.Namespace})
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

			helper.SetResponsibleTeam(helper.TeamAtlas)

			// We need to wait for the observability-bundle app to be deployed before we can check the apps it deploys.
			observabilityAppsAppName := fmt.Sprintf("%s-%s", state.GetCluster().Name, "observability-bundle")

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), observabilityAppsAppName, state.GetCluster().GetNamespace())).
				WithTimeout(30 * time.Second).
				WithPolling(50 * time.Millisecond).
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
	})
	Context("security-bundle apps", func() {
		It("all security-bundle apps are deployed without issues", func() {
			if !cfg.SecurityBundleInstalled {
				Skip("security-bundle is not installed")
			}

			helper.SetResponsibleTeam(helper.TeamShield)

			// We need to wait for the security-bundle app to be deployed before we can check the apps it deploys.
			securityAppsAppName := fmt.Sprintf("%s-%s", state.GetCluster().Name, "security-bundle")

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), securityAppsAppName, state.GetCluster().GetNamespace())).
				WithTimeout(30 * time.Second).
				WithPolling(50 * time.Millisecond).
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
	})
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
			helmRelease := &helm.HelmRelease{}
			err := client.Get(ctx, hr, helmRelease)
			if err != nil {
				logger.Log("HelmRelease status for '%s' failed to retrieve: %v", hr.Name, err)
				allReady = false
				continue
			}

			ready := false
			readyCondition := ""
			readyReason := ""
			readyMessage := ""

			for _, condition := range helmRelease.Status.Conditions {
				if condition.Type == "Ready" {
					if condition.Status == metav1.ConditionTrue {
						ready = true
					}
					readyCondition = string(condition.Status)
					readyReason = condition.Reason
					readyMessage = condition.Message
					break
				}
			}

			if ready {
				logger.Log("HelmRelease status for '%s' is as expected: expectedStatus='Ready' actualStatus='Ready'", hr.Name)
			} else {
				if readyCondition == "" {
					logger.Log("HelmRelease status for '%s' is not yet as expected: expectedStatus='Ready' actualStatus='Unknown' (reason: 'No Ready condition found')", hr.Name)
				} else {
					logger.Log("HelmRelease status for '%s' is not yet as expected: expectedStatus='Ready' actualStatus='%s' (reason: '%s - %s')", hr.Name, readyCondition, readyReason, readyMessage)
				}
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

		helmReleaseList := &helm.HelmReleaseList{}
		err := state.GetFramework().MC().List(ctx, helmReleaseList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()))
		if err != nil {
			logger.Log("Failed to get HelmReleases - %v", err)
			return
		}

		for _, hr := range helmReleaseList.Items {
			ready := false
			for _, condition := range hr.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
					ready = true
					break
				}
			}

			if !ready {
				// Get team label from labels
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
