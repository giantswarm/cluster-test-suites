package common

import (
	"context"
	"fmt"
	"time"

	helm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/giantswarm/clustertest/v4/pkg/client"
	"github.com/giantswarm/clustertest/v4/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v4/pkg/logger"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/v6/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v6/internal/state"
)

func runScale(autoScalingSupported bool) {
	Context("scale", func() {
		var (
			helmRelease  *helm.HelmRelease
			wcClient     *client.Client
			replicaCount int
		)

		BeforeEach(func() {
			if !autoScalingSupported {
				Skip("autoscaling is not supported")
			}

			helper.SetResponsibleTeam(helper.TeamTenet)

			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}

			ctx := context.Background()
			org := state.GetCluster().Organization
			clusterName := state.GetCluster().Name
			namespace := org.GetNamespace()

			// Get the current number of worker nodes and set the replicas to one more to force scale up
			nodes := corev1.NodeList{}
			err = wcClient.List(ctx, &nodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})
			Expect(err).To(BeNil())

			replicaCount = len(nodes.Items) + 1

			values := map[string]interface{}{
				"autoscaling": map[string]interface{}{
					"enabled": false,
				},
				"replicaCount": replicaCount,
				"affinity": map[string]interface{}{
					"podAntiAffinity": map[string]interface{}{
						"requiredDuringSchedulingIgnoredDuringExecution": []interface{}{
							map[string]interface{}{
								"labelSelector": map[string]interface{}{
									"matchExpressions": []interface{}{
										map[string]interface{}{
											"key":      "app.kubernetes.io/instance",
											"operator": "In",
											"values":   []interface{}{"scale-hello-world"},
										},
									},
								},
								"topologyKey": "kubernetes.io/hostname",
							},
						},
					},
				},
			}

			err = ensureTestHelmRepository(ctx, state.GetFramework().MC(), namespace)
			Expect(err).To(BeNil())

			helmRelease, err = newTestHelmRelease(
				fmt.Sprintf("%s-scale-hello-world", clusterName),
				namespace,
				"hello-world",
				"scale-hello-world",
				"giantswarm",
				clusterName,
				values,
			)
			Expect(err).To(BeNil())

			err = state.GetFramework().MC().Create(ctx, helmRelease)
			Expect(err).To(BeNil())

			Eventually(isHelmReleaseReady(ctx, state.GetFramework().MC(), types.NamespacedName{
				Name:      helmRelease.Name,
				Namespace: helmRelease.Namespace,
			})).
				WithTimeout(5 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		It("scales node by creating anti-affinity pods", func() {
			if !autoScalingSupported {
				Skip("autoscaling is not supported")
			}

			ctx := context.Background()

			expectedReplicas := fmt.Sprintf("%d", replicaCount)
			Eventually(func() (bool, error) {
				deploymentName := "scale-hello-world"
				helloDeployment := &v1.Deployment{}

				err := wcClient.Get(ctx,
					cr.ObjectKey{
						Name:      deploymentName,
						Namespace: "giantswarm",
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
			}, "15m", "10s").Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate 'hello-world' deployment has not scaled up properly"))
		})

		AfterEach(func() {
			if !autoScalingSupported {
				Skip("autoscaling is not supported")
			}

			ctx := state.GetContext()
			err := state.GetFramework().MC().Delete(ctx, helmRelease)
			Expect(err).ShouldNot(HaveOccurred())

			err = deleteTestHelmRepository(ctx, state.GetFramework().MC(), helmRelease.Namespace)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
}
