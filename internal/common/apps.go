package common

import (
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/wait"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runApps() {
	Context("default apps", func() {
		It("all default apps are deployed without issues", func() {

			// We need to wait for default-apps to be deployed before we can check all apps.
			defaultAppsAppName := fmt.Sprintf("%s-%s", state.GetCluster().Name, "default-apps")

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), defaultAppsAppName, state.GetCluster().Organization.GetNamespace())).
				WithTimeout(30 * time.Second).
				WithPolling(50 * time.Millisecond).
				Should(BeTrue())

			// Wait for all default-apps apps to be deployed
			appList := &v1alpha1.AppList{}
			err := state.GetFramework().MC().List(state.GetContext(), appList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()), ctrl.MatchingLabels{"giantswarm.io/managed-by": defaultAppsAppName})
			Expect(err).NotTo(HaveOccurred())

			appNamespacedNames := []types.NamespacedName{}
			for _, app := range appList.Items {
				appNamespacedNames = append(appNamespacedNames, types.NamespacedName{Name: app.Name, Namespace: app.Namespace})
			}

			Eventually(wait.IsAllAppDeployed(state.GetContext(), state.GetFramework().MC(), appNamespacedNames)).
				WithTimeout(15 * time.Minute).
				WithPolling(10 * time.Second).
				Should(BeTrue())
		})
	})
	Context("observability-bundle apps", func() {
		It("all observability-bundle apps are deployed without issues", func() {

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
				WithTimeout(25 * time.Minute).
				WithPolling(10 * time.Second).
				Should(BeTrue())
		})
	})
}
