package common

import (
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

const deployedStatus = "deployed"

func runApps() {
	Context("default apps", func() {
		It("all default apps are deployed without issues", func() {

			// We need to wait for default-apps to be deployed before we can check all apps.
			defaultAppsAppName := fmt.Sprintf("%s-%s", state.GetCluster().Name, "default-apps")
			Eventually(func() error {
				app, err := state.GetFramework().GetApp(state.GetContext(), defaultAppsAppName, state.GetCluster().Organization.GetNamespace())
				if err != nil {
					return err
				}
				return checkAppStatus(app)
			}).
				WithTimeout(30 * time.Second).
				WithPolling(50 * time.Millisecond).
				Should(Succeed())

			// Wait for all default-apps apps to be deployed
			Eventually(func() error {
				appList := &v1alpha1.AppList{}
				err := state.GetFramework().MC().List(state.GetContext(), appList, ctrl.InNamespace(state.GetCluster().Organization.GetNamespace()), ctrl.MatchingLabels{"giantswarm.io/managed-by": defaultAppsAppName})
				if err != nil {
					return err
				}

				logger.Log("Checking status of %d apps", len(appList.Items))
				errs := []error{}
				for _, app := range appList.Items {
					if err := checkAppStatus(&app); err != nil {
						errs = append(errs, err)
					}
				}
				return errors.NewAggregate(errs)
			}).
				WithTimeout(15 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())

		})
	})
}

func checkAppStatus(app *v1alpha1.App) error {
	if app.Status.Release.Status != deployedStatus {
		logger.Log("App %s status is currently '%s'", app.Name, app.Status.Release.Status)
		return fmt.Errorf("app %s status is '%s'", app.Name, app.Status.Release.Status)
	}
	return nil
}
