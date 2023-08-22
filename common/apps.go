package common

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runApps() {
	Context("default apps", func() {
		var customFormatterKey format.CustomFormatterKey
		BeforeEach(func() {
			customFormatterKey = format.RegisterCustomFormatter(func(value interface{}) (string, bool) {
				app, ok := value.(v1alpha1.App)
				if ok {
					return fmt.Sprintf("App: %s/%s, Status: %s, Reason: %s", app.Namespace, app.Name, app.Status.Release.Status, app.Status.Release.Reason), true
				}

				return "", false
			})
		})

		It("all default apps are deployed without issues", func() {
			ctx := context.Background()

			// We need to wait for default-apps to be deployed before we can check all apps.
			defaultAppsAppName := fmt.Sprintf("%s-%s", state.GetCluster().Name, "default-apps")
			Eventually(state.GetFramework().GetApp, "30s").WithContext(ctx).WithArguments(defaultAppsAppName, state.GetCluster().Organization.GetNamespace()).Should(HaveAppStatus("deployed"))

			managementClusterKubeClient := state.GetFramework().MC()
			appList := &v1alpha1.AppList{}
			err := managementClusterKubeClient.List(ctx, appList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()), ctrl.MatchingLabels{"giantswarm.io/managed-by": defaultAppsAppName})
			Expect(err).ShouldNot(HaveOccurred())

			for _, app := range appList.Items {
				Eventually(state.GetFramework().GetApp, "15m", "10s").WithContext(ctx).WithArguments(app.Name, app.Namespace).Should(HaveAppStatus("deployed"))
			}
		})

		AfterEach(func() {
			format.UnregisterCustomFormatter(customFormatterKey)
		})
	})
}

func HaveAppStatus(expected string) types.GomegaMatcher {
	return &haveAppStatus{expected: expected}
}

type haveAppStatus struct {
	expected string
}

func (m *haveAppStatus) Match(actual interface{}) (bool, error) {
	if actual == nil {
		return false, nil
	}

	actualApp, isApp := actual.(*v1alpha1.App)
	if !isApp {
		return false, fmt.Errorf("%#v is not an App", actual)
	}

	return Equal(actualApp.Status.Release.Status).Match(m.expected)
}

func (m *haveAppStatus) FailureMessage(actual interface{}) (message string) {
	return format.Message(
		actual,
		fmt.Sprintf("to be an App with release status: %s", m.expected),
	)
}

func (m *haveAppStatus) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(
		actual,
		fmt.Sprintf("not to be an App with release status: %s", m.expected),
	)
}
