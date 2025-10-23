package common

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v2/pkg/application"
	"github.com/giantswarm/clustertest/v2/pkg/client"
	"github.com/giantswarm/clustertest/v2/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v2/pkg/logger"
	"github.com/giantswarm/clustertest/v2/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v2/internal/state"
	"github.com/giantswarm/cluster-test-suites/v2/internal/timeout"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

func runBasic() {
	Context("basic", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("should be able to connect to the management cluster", FlakeAttempts(3), func() {
			Expect(state.GetFramework().MC().CheckConnection()).To(Succeed())
		})

		It("should be able to connect to the workload cluster", FlakeAttempts(5), func() {
			Expect(wcClient.CheckConnection()).To(Succeed())
		})

		It("has all the control-plane nodes running", func() {
			replicas, err := state.GetFramework().GetExpectedControlPlaneReplicas(state.GetContext(), state.GetCluster().Name, state.GetCluster().GetNamespace())
			Expect(err).NotTo(HaveOccurred())

			// Skip this test is the cluster is a managed cluster (e.g. EKS)
			if replicas == 0 {
				Skip("ControlPlane is not supported.")
			}

			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreNumNodesReady(state.GetContext(), wcClient, int(replicas), &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
					5,
					5*time.Second,
				)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(CheckWorkerNodesReady(state.GetContext(), wcClient, values), 12, 5*time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its Deployments Ready (means all replicas are running)", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllDeploymentsReady(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.Bundle(
						failurehandler.DeploymentsNotReady(state.GetFramework(), state.GetCluster()),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate deployments not ready"),
					),
				)
		})

		It("has all its StatefulSets Ready (means all replicas are running)", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllStatefulSetsReady(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.Bundle(
						failurehandler.StatefulSetsNotReady(state.GetFramework(), state.GetCluster()),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate statefulsets not ready"),
					),
				)
		})

		It("has all its DaemonSets Ready (means all daemon pods are running)", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllDaemonSetsReady(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.Bundle(
						failurehandler.DaemonSetsNotReady(state.GetFramework(), state.GetCluster()),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate daemonsets not ready"),
					),
				)
		})

		It("has all its Jobs completed successfully", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllJobsSucceeded(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.Bundle(
						failurehandler.JobsUnsuccessful(state.GetFramework(), state.GetCluster()),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate kubernetes Jobs that have not finished successfully"),
					),
				)
		})

		It("has all of its Pods in the Running state", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllPodsInSuccessfulPhase(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.Bundle(
						failurehandler.PodsNotReady(state.GetFramework(), state.GetCluster()),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate pods that are not in Running state"),
					),
				)
		})

		It("doesn't have restarting pods", func() {
			filterLabels := []string{
				// Excluding cluster-autoscaler as we have a specific test case for ensuring it is functioning
				"app.kubernetes.io/name!=cluster-autoscaler-app",
			}

			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreNoPodsCrashLoopingWithFilter(state.GetContext(), wcClient, 0, filterLabels),
					10,
					5*time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.Bundle(
						failurehandler.PodsNotReady(state.GetFramework(), state.GetCluster()),
						failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate pods that are restarting"),
					),
				)
		})

		It("has Cluster Ready condition with Status='True'", func() {
			// Overriding the default timeout, when ClusterReadyTimeout is set
			timeout := state.GetTestTimeout(timeout.ClusterReadyTimeout, 15*time.Minute)

			mcClient := state.GetFramework().MC()
			cluster := state.GetCluster()
			Eventually(wait.IsClusterConditionSet(state.GetContext(), mcClient, cluster.Name, cluster.GetNamespace(), capi.ReadyCondition, corev1.ConditionTrue, "")).
				WithTimeout(timeout).
				WithPolling(wait.DefaultInterval).
				Should(BeTrue())
		})

		It("has all machine pools ready and running", func() {
			mcClient := state.GetFramework().MC()
			cluster := state.GetCluster()

			machinePools, err := state.GetFramework().GetMachinePools(state.GetContext(), cluster.Name, cluster.GetNamespace())
			Expect(err).NotTo(HaveOccurred())
			if len(machinePools) == 0 {
				Skip("Machine pools are not found")
			}

			Eventually(wait.Consistent(CheckMachinePoolsReadyAndRunning(state.GetContext(), mcClient, cluster.Name, cluster.GetNamespace()), 5, 5*time.Second)).
				WithTimeout(30 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
}

func CheckWorkerNodesReady(ctx context.Context, wcClient *client.Client, values *application.ClusterValues) func() error {
	minNodes := 0
	maxNodes := 0
	for _, pool := range values.NodePools {
		// MachineDeployment node pool
		if pool.Replicas > 0 {
			minNodes += pool.Replicas
			maxNodes += pool.Replicas
			continue
		}

		if pool.MinSize == 0 && pool.MaxSize == 0 {
			// It's a Karpenter node pool, and we don't care about the number of workers.
			// We set the min to 2 as we have some affinity rules that would require at least that.
			minNodes += 2
			maxNodes += 99
		}

		// MachinePool node pool
		minNodes += pool.MinSize
		maxNodes += pool.MaxSize
	}
	expectedNodes := wait.Range{
		Min: minNodes,
		Max: maxNodes,
	}

	workersFunc := wait.AreNumNodesReadyWithinRange(ctx, wcClient, expectedNodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})

	return func() error {
		ok, err := workersFunc()
		if err != nil {
			logger.Log("failed to get nodes: %s", err)
			return err
		}
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return nil
	}
}

// CheckMachinePoolsReadyAndRunning checks if all MachinePool resources have Ready condition with Status True and are in
// Running phase.
func CheckMachinePoolsReadyAndRunning(ctx context.Context, mcClient *client.Client, clusterName string, clusterNamespace string) func() error {
	return func() error {
		machinePools := &capiexp.MachinePoolList{}
		machinePoolListOptions := []cr.ListOption{
			cr.InNamespace(clusterNamespace),
			cr.MatchingLabels{
				"cluster.x-k8s.io/cluster-name": clusterName,
			},
		}
		err := mcClient.List(ctx, machinePools, machinePoolListOptions...)
		if err != nil {
			return err
		}

		if len(machinePools.Items) == 0 {
			logger.Log("MachinePools not found.")
			return nil
		}

		allMachinePoolsAreReadyAndRunning := true
		for _, mp := range machinePools.Items {
			machinePool := mp
			var machinePoolIsReady bool
			machinePoolIsReady, err = wait.IsClusterApiObjectConditionSet(&mp, capi.ReadyCondition, corev1.ConditionTrue, "")
			if err != nil {
				return err
			}
			allMachinePoolsAreReadyAndRunning = allMachinePoolsAreReadyAndRunning && machinePoolIsReady

			currentMachinePoolPhase := capiexp.MachinePoolPhase(machinePool.Status.Phase)
			machinePoolIsRunning := currentMachinePoolPhase == capiexp.MachinePoolPhaseRunning
			allMachinePoolsAreReadyAndRunning = allMachinePoolsAreReadyAndRunning && machinePoolIsRunning
			logger.Log(
				"MachinePool '%s/%s' expected to be in Running phase, found MachinePool is in '%s' phase.",
				machinePool.Namespace,
				machinePool.Name,
				machinePool.Status.Phase)
		}

		if !allMachinePoolsAreReadyAndRunning {
			return fmt.Errorf("not all MachinePools are ready and running")
		}

		return nil
	}
}
