package common

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

		It("should be able to connect to MC cluster", FlakeAttempts(3), func() {
			Expect(state.GetFramework().MC().CheckConnection()).To(Succeed())
		})

		It("should be able to connect to WC cluster", FlakeAttempts(3), func() {
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

			Eventually(wait.Consistent(CheckControlPlaneNodesReady(state.GetContext(), wcClient, int(replicas)), 12, 5*time.Second)).
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
			Eventually(wait.Consistent(CheckAllDeploymentsReady(state.GetContext(), wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its StatefulSets Ready (means all replicas are running)", func() {
			Eventually(wait.Consistent(CheckAllStatefulSetsReady(state.GetContext(), wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its DaemonSets Ready (means all daemon pods are running)", func() {
			Eventually(wait.Consistent(CheckAllDaemonSetsReady(state.GetContext(), wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its Jobs completed successfully", func() {
			Eventually(wait.Consistent(CheckAllJobsSucceeded(state.GetContext(), wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all of its Pods in the Running state", func() {
			Eventually(wait.Consistent(CheckAllPodsSuccessfulPhase(state.GetContext(), wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has Cluster Ready condition with Status='True'", func() {
			mcClient := state.GetFramework().MC()
			cluster := state.GetCluster()
			Eventually(wait.IsClusterConditionSet(state.GetContext(), mcClient, cluster.Name, cluster.GetNamespace(), capi.ReadyCondition, corev1.ConditionTrue, "")).
				WithTimeout(15 * time.Minute).
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

func CheckControlPlaneNodesReady(ctx context.Context, wcClient *client.Client, expectedNodes int) func() error {
	controlPlaneFunc := wait.AreNumNodesReady(ctx, wcClient, expectedNodes, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""})

	return func() error {
		ok, err := controlPlaneFunc()
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return err
	}
}

func CheckWorkerNodesReady(ctx context.Context, wcClient *client.Client, values *application.ClusterValues) func() error {
	minNodes := 0
	maxNodes := 0
	for _, pool := range values.NodePools {
		if pool.Replicas > 0 {
			minNodes += pool.Replicas
			maxNodes += pool.Replicas
			continue
		}

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

func CheckAllPodsSuccessfulPhase(ctx context.Context, wcClient *client.Client) func() error {
	return func() error {
		podList := &corev1.PodList{}
		err := wcClient.List(ctx, podList)
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				logger.Log("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
				return fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		logger.Log("All (%d) pods currently in a running or completed state", len(podList.Items))
		return nil
	}
}

func CheckAllDeploymentsReady(ctx context.Context, wcClient *client.Client) func() error {
	return func() error {
		deploymentList := &appsv1.DeploymentList{}
		err := wcClient.List(ctx, deploymentList)
		if err != nil {
			return err
		}

		for _, deployment := range deploymentList.Items {
			available := deployment.Status.AvailableReplicas
			desired := *deployment.Spec.Replicas
			if available != desired {
				logger.Log("deployment %s/%s has %d/%d replicas available", deployment.Namespace, deployment.Name, available, desired)
				return fmt.Errorf("deployment %s/%s has %d/%d replicas available", deployment.Namespace, deployment.Name, available, desired)
			}
		}

		logger.Log("All (%d) deployments have all replicas running", len(deploymentList.Items))
		return nil
	}
}

func CheckAllStatefulSetsReady(ctx context.Context, wcClient *client.Client) func() error {
	return func() error {
		statefulSetList := &appsv1.StatefulSetList{}
		err := wcClient.List(ctx, statefulSetList)
		if err != nil {
			return err
		}

		for _, statefulSet := range statefulSetList.Items {
			available := statefulSet.Status.AvailableReplicas
			desired := *statefulSet.Spec.Replicas
			if available != desired {
				logger.Log("statefulSet %s/%s has %d/%d replicas available", statefulSet.Namespace, statefulSet.Name, available, desired)
				return fmt.Errorf("statefulSet %s/%s has %d/%d replicas available", statefulSet.Namespace, statefulSet.Name, available, desired)
			}
		}

		logger.Log("All (%d) statefulSets have all replicas running", len(statefulSetList.Items))
		return nil
	}
}

func CheckAllJobsSucceeded(ctx context.Context, wcClient *client.Client) func() error {
	return func() error {
		jobList := &batchv1.JobList{}
		err := wcClient.List(ctx, jobList)
		if err != nil {
			return err
		}

		var loopErr error
		for _, job := range jobList.Items {
			if job.Status.Succeeded == 0 {
				logger.Log("Job %s/%s has not succeeded. (Failed: '%d')", job.ObjectMeta.Namespace, job.ObjectMeta.Name, job.Status.Failed)
				// We wrap the erros so that we can log out for all failures, not just the first found
				if loopErr != nil {
					loopErr = fmt.Errorf("%w, job %s/%s has not succeeded", loopErr, job.ObjectMeta.Namespace, job.ObjectMeta.Name)
				} else {
					loopErr = fmt.Errorf("job %s/%s has not succeeded", job.ObjectMeta.Namespace, job.ObjectMeta.Name)
				}
			}
		}
		if loopErr != nil {
			return loopErr
		}

		logger.Log("All (%d) Jobs have completed successfully", len(jobList.Items))
		return nil
	}
}

func CheckAllDaemonSetsReady(ctx context.Context, wcClient *client.Client) func() error {
	return func() error {
		daemonSetList := &appsv1.DaemonSetList{}
		err := wcClient.List(ctx, daemonSetList)
		if err != nil {
			return err
		}

		for _, daemonSet := range daemonSetList.Items {
			current := daemonSet.Status.CurrentNumberScheduled
			desired := daemonSet.Status.DesiredNumberScheduled
			if current != desired {
				logger.Log("daemonSet %s/%s has %d/%d daemon pods available", daemonSet.Namespace, daemonSet.Name, current, desired)
				return fmt.Errorf("daemonSet %s/%s has %d/%d daemon pods available", daemonSet.Namespace, daemonSet.Name, current, desired)
			}
		}

		logger.Log("All (%d) daemonSets have all daemon pods running", len(daemonSetList.Items))
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
