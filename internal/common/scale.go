package common

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runScale(autoScalingSupported bool) {
	Context("scale", func() {
		var (
			helloApp       *application.Application
			wcClient       *client.Client
			helloAppValues map[string]string
		)

		BeforeEach(func() {
			if !autoScalingSupported {
				Skip("autoscaling is not supported")
			}

			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}

			ctx := context.Background()
			org := state.GetCluster().Organization

			helloAppValues = map[string]string{
				"ReplicaCount": "3",
			}

			helloApp = application.New(fmt.Sprintf("%s-scale-hello-world", state.GetCluster().Name), "hello-world").
				WithCatalog("giantswarm").
				WithOrganization(*org).
				WithVersion("latest").
				WithClusterName(state.GetCluster().Name).
				WithInCluster(false).
				WithInstallNamespace("giantswarm").
				MustWithValuesFile("./test_data/scale_helloworld_values.yaml", &application.TemplateValues{
					ClusterName:  state.GetCluster().Name,
					Organization: state.GetCluster().Organization.Name,
					ExtraValues:  helloAppValues,
				})

			err = state.GetFramework().MC().DeployApp(ctx, *helloApp)
			Expect(err).To(BeNil())

			Eventually(func() (bool, error) {
				managementClusterKubeClient := state.GetFramework().MC()

				helloApplication := &v1alpha1.App{}
				err := managementClusterKubeClient.Get(ctx, types.NamespacedName{Name: helloApp.InstallName, Namespace: helloApp.GetNamespace()}, helloApplication)
				if err != nil {
					return false, err
				}

				now := time.Now()
				patchedApp := helloApplication.DeepCopy()
				labels := patchedApp.GetLabels()
				labels["update"] = fmt.Sprintf("%d", now.Unix())
				patchedApp.SetLabels(labels)

				err = managementClusterKubeClient.Patch(ctx, patchedApp, ctrl.MergeFrom(helloApplication))
				if err != nil {
					return false, err
				}

				return wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), helloApp.InstallName, helloApp.GetNamespace())()
			}).
				WithTimeout(15 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		It("scales node by creating anti-affiniy pods", func() {
			if !autoScalingSupported {
				Skip("external-dns is not supported")
			}

			ctx := context.Background()

			Eventually(func() (string, error) {
				logger.Log("Trying to get number of nodes from %s", state.GetCluster().Name)
				helloDeployment := &v1.Deployment{}

				err := wcClient.Get(ctx, cr.ObjectKey{
					Name:      "scale-hello-world",
					Namespace: helloApp.InstallNamespace}, helloDeployment)

				if err != nil {
					return "", err
				}

				return strconv.Itoa(int(helloDeployment.Status.ReadyReplicas)), nil
			}, "10m", "5s").Should(Equal(helloAppValues["ReplicaCount"]))
		})

		AfterEach(func() {
			if !autoScalingSupported {
				Skip("autoscaling is not supported")
			}

			err := state.GetFramework().MC().DeleteApp(state.GetContext(), *helloApp)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
}
