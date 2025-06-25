package common

import (
	"context"
	"fmt"
	"time"

	helmv2beta2 "github.com/fluxcd/helm-controller/api/v2beta2"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/pkg/failurehandler"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/internal/helper"
	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/internal/timeout"
)

func RunHelmReleases() {
	Context("helm releases", func() {
		It("all HelmReleases are successful", func() {
			timeout := state.GetTestTimeout(timeout.DeployApps, 15*time.Minute)
			logger.Log("Waiting for all HelmReleases to be ready. Timeout: %s", timeout.String())

			// Get all HelmReleases in the cluster organization namespace
			helmReleaseList := &helmv2beta2.HelmReleaseList{}
			err := state.GetFramework().MC().List(state.GetContext(), helmReleaseList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()))
			Expect(err).NotTo(HaveOccurred())

			if len(helmReleaseList.Items) == 0 {
				logger.Log("No HelmReleases found in namespace %s", state.GetCluster().Organization.GetNamespace())
				return
			}

			logger.Log("Found %d HelmReleases to check", len(helmReleaseList.Items))

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
						helmReleaseIssues(),
						reportHelmReleaseOwningTeams(),
					),
				)
		})
	})
}

// areAllHelmReleasesReady checks if all HelmReleases in the list are ready
func areAllHelmReleasesReady(ctx context.Context, client ctrl.Client, helmReleases []types.NamespacedName) func() error {
	return func() error {
		for _, hr := range helmReleases {
			helmRelease := &helmv2beta2.HelmRelease{}
			err := client.Get(ctx, hr, helmRelease)
			if err != nil {
				return fmt.Errorf("failed to get HelmRelease %s/%s: %w", hr.Namespace, hr.Name, err)
			}

			ready := false
			for _, condition := range helmRelease.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
					ready = true
					break
				}
			}

			if !ready {
				// Log current condition details for debugging
				conditionDetails := ""
				for _, condition := range helmRelease.Status.Conditions {
					if condition.Type == "Ready" {
						conditionDetails = fmt.Sprintf("Ready condition: Status=%s, Reason=%s, Message=%s", condition.Status, condition.Reason, condition.Message)
						break
					}
				}
				if conditionDetails == "" {
					conditionDetails = "No Ready condition found"
				}

				return fmt.Errorf("HelmRelease %s/%s is not ready: %s", hr.Namespace, hr.Name, conditionDetails)
			}
		}
		return nil
	}
}

// helmReleaseIssues creates a failure handler for HelmRelease issues
func helmReleaseIssues() failurehandler.FailureHandler {
	return failurehandler.Wrap(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		logger.Log("Gathering HelmRelease status information for debugging")

		helmReleaseList := &helmv2beta2.HelmReleaseList{}
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
				logger.Log("HelmRelease '%s/%s' is not ready:", hr.Namespace, hr.Name)
				for _, condition := range hr.Status.Conditions {
					logger.Log("  Condition: Type=%s, Status=%s, Reason=%s, Message=%s",
						condition.Type, condition.Status, condition.Reason, condition.Message)
				}

				// Log recent events for this HelmRelease
				events := &corev1.EventList{}
				err := state.GetFramework().MC().List(ctx, events, ctrl.InNamespace(hr.Namespace),
					ctrl.MatchingFields{"involvedObject.name": hr.Name})
				if err == nil {
					logger.Log("  Recent events:")
					for _, event := range events.Items {
						if event.InvolvedObject.Kind == "HelmRelease" {
							logger.Log("    %s: %s", event.Reason, event.Message)
						}
					}
				}
			}
		}
	})
}

// reportHelmReleaseOwningTeams reports the teams responsible for failing HelmReleases
func reportHelmReleaseOwningTeams() failurehandler.FailureHandler {
	return failurehandler.Wrap(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		logger.Log("Attempting to get responsible teams for any failing HelmReleases")

		helmReleaseList := &helmv2beta2.HelmReleaseList{}
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
				teamLabel, ok := hr.Labels["application.giantswarm.io/team"]
				if ok && !helper.SetResponsibleTeamFromLabel(teamLabel) {
					logger.Log("Unknown owner team - HelmRelease='%s', TeamLabel='%s'", hr.Name, teamLabel)
				}
			}
		}
	})
}
