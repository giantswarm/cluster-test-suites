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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/helper"
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

			helper.SetResponsibleTeam(helper.TeamPhoenix)

			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}

			ctx := context.Background()
			org := state.GetCluster().Organization

			// Get the current number of worker nodes and set the replicas to one more to force scale up
			nodes := corev1.NodeList{}
			err = wcClient.List(ctx, &nodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})
			Expect(err).To(BeNil())

			helloAppValues = map[string]string{
				"ReplicaCount": fmt.Sprintf("%d", len(nodes.Items)+1),
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

		It("scales node by creating anti-affinity pods", func() {
			if !autoScalingSupported {
				Skip("autoscaling is not supported")
			}

			ctx := context.Background()

			expectedReplicas := helloAppValues["ReplicaCount"]
			Eventually(func() (bool, error) {
				deploymentName := "scale-hello-world"
				helloDeployment := &v1.Deployment{}

				err := wcClient.Get(ctx,
					cr.ObjectKey{
						Name:      deploymentName,
						Namespace: helloApp.InstallNamespace,
					},
					helloDeployment,
				)
				if err != nil {
					return false, err
				}

				replicas := fmt.Sprint(helloDeployment.Status.ReadyReplicas)
				logger.Log("Checking for increased replicas. Expected: %s, Actual: %s", expectedReplicas, replicas)
				if replicas == expectedReplicas {
					return true, nil
				}

				// Logging out information about pod conditions
				pods := corev1.PodList{}
				err = wcClient.List(ctx, &pods, cr.MatchingLabels{"app.kubernetes.io/instance": deploymentName})
				if err != nil {
					return false, err
				}
				podConditionMessages := []string{}
				for _, pod := range pods.Items {
					if pod.Status.Phase != corev1.PodRunning {
						for _, condition := range pod.Status.Conditions {
							if condition.Status != corev1.ConditionTrue && condition.Message != "" {
								podConditionMessages = append(podConditionMessages, fmt.Sprintf("%s='%s'", pod.ObjectMeta.Name, condition.Message))
							}
						}
					}
				}
				logger.Log("Condition messages from non-running deployment pods: %v", podConditionMessages)

				// Logging out information about node status
				nodes := corev1.NodeList{}
				err = wcClient.List(ctx, &nodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})
				if err != nil {
					return false, err
				}
				logger.Log("There are currently '%d' worker nodes", len(nodes.Items))
				for _, node := range nodes.Items {
					logger.Log("Worker node status: NodeName='%s', Taints='%s'", node.ObjectMeta.Name, node.Spec.Taints)
				}

				return false, nil
			}, "15m", "10s").Should(BeTrue())
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
