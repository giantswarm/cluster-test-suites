package common

import (
	"context"
	"fmt"
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
				"ReplicaCount": "2",
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

				return wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), helloApp.InstallName, helloApp.GetNamespace())()
			}).
				WithTimeout(5 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		It("scales node by creating anti-affiniy pods", func() {
			if !autoScalingSupported {
				Skip("autoscaling is not supported")
			}

			ctx := context.Background()

			expectedReplicas := helloAppValues["ReplicaCount"]
			Eventually(func() (string, error) {
				helloDeployment := &v1.Deployment{}

				err := wcClient.Get(ctx,
					cr.ObjectKey{
						Name:      "scale-hello-world",
						Namespace: helloApp.InstallNamespace,
					},
					helloDeployment,
				)
				if err != nil {
					return "", err
				}

				replicas := fmt.Sprint(helloDeployment.Status.ReadyReplicas)
				logger.Log("Checking for increased replicas. Expected: %s, Actual: %s", expectedReplicas, replicas)
				return replicas, nil
			}, "15m", "10s").Should(Equal(expectedReplicas))
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
