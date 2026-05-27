package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	capi "sigs.k8s.io/cluster-api/api/core/v1beta2"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v5/pkg/application"
	"github.com/giantswarm/clustertest/v5/pkg/client"
	"github.com/giantswarm/clustertest/v5/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v5/pkg/logger"
	"github.com/giantswarm/clustertest/v5/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

func runBasic(cfg *TestConfig) {
	Context("basic", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("should be able to connect to the management cluster", func() {
			Eventually(func() error {
				return state.GetFramework().MC().CheckConnection()
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("should be able to connect to the workload cluster", func() {
			Eventually(func() error {
				return wcClient.CheckConnection()
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
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
					failurehandler.DeploymentsNotReady(state.GetFramework(), state.GetCluster()),
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
					failurehandler.StatefulSetsNotReady(state.GetFramework(), state.GetCluster()),
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
					failurehandler.DaemonSetsNotReady(state.GetFramework(), state.GetCluster()),
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
					failurehandler.JobsUnsuccessful(state.GetFramework(), state.GetCluster()),
				)
		})

		It("has all of its Pods in the Running state", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					AreAllPodsInSuccessfulPhaseWithFilter(state.GetContext(), wcClient, armExcludedPodLabels(cfg)),
					10,
					time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.PodsNotReady(state.GetFramework(), state.GetCluster()),
				)
		})

		It("doesn't have restarting pods", func() {
			// Excluding cluster-autoscaler as we have a specific test case for ensuring it is functioning
			// Excluding karpenter because it's deployed using a HelmRelease and its pods run on the control plane. Because of this the pod is scheduled pretty early in the cluster creation process.
			// Meanwhile, IRSA resources are getting created, but it takes a while. karpenter uses IRSA and can't run until IRSA is ready. Eventually, IRSA is ready, and the pod works normally.
			excludedAppNames := []string{"cluster-autoscaler-app", "karpenter"}
			excludedAppNames = append(excludedAppNames, armExcludedAppNames(cfg)...)
			filterLabels := []string{
				fmt.Sprintf("app.kubernetes.io/name notin (%s)", strings.Join(excludedAppNames, ", ")),
			}

			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreNoPodsCrashLoopingWithFilter(state.GetContext(), wcClient, 2, filterLabels),
					10,
					5*time.Second,
				)).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.PodsNotReady(state.GetFramework(), state.GetCluster()),
				)
		})

		It("has Cluster Available condition with Status='True'", func() {
			// Overriding the default timeout, when ClusterReadyTimeout is set
			timeout := state.GetTestTimeout(timeout.ClusterReadyTimeout, 15*time.Minute)

			mcClient := state.GetFramework().MC()
			cluster := state.GetCluster()
			Eventually(wait.IsClusterConditionSet(state.GetContext(), mcClient, cluster.Name, cluster.GetNamespace(), capi.AvailableCondition, metav1.ConditionTrue, "")).
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

// armExcludedAppNames returns the `app.kubernetes.io/name` values to exclude from pod
// health checks when an arm64 node pool is present. The released net-exporter and
// cert-exporter app versions aren't multi-arch yet, so their DaemonSet pods crashloop on
// arm64 nodes. cluster-test-suites always pulls the latest release, which lags the fixed
// app versions, so exclude them until those versions ship.
//
// TODO(arm64): temporary. Remove this exclusion (and the ARMNodePoolEnabled plumbing)
// once release v35 ships the multi-arch net-exporter and cert-exporter versions.
// See: https://github.com/giantswarm/roadmap/issues/4302
func armExcludedAppNames(cfg *TestConfig) []string {
	if !cfg.ARMNodePoolEnabled {
		return nil
	}
	// net-exporter pods use `app.kubernetes.io/name: net-exporter`; cert-exporter's
	// DaemonSet pods use `app.kubernetes.io/name: cert-exporter-daemonset`.
	return []string{"net-exporter", "cert-exporter-daemonset"}
}

// armExcludedPodLabels returns label selectors filtering out the arm64-incompatible apps,
// or an empty slice when there's nothing to exclude (so checks behave as before).
func armExcludedPodLabels(cfg *TestConfig) []string {
	names := armExcludedAppNames(cfg)
	if len(names) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("app.kubernetes.io/name notin (%s)", strings.Join(names, ", "))}
}

// AreAllPodsInSuccessfulPhaseWithFilter checks that all Pods (minus those matched by the
// exclusion label selectors) are in a running or completed phase. It mirrors
// wait.AreAllPodsInSuccessfulPhase, which has no filtered variant upstream.
func AreAllPodsInSuccessfulPhaseWithFilter(ctx context.Context, wcClient *client.Client, filterLabels []string) wait.WaitCondition {
	return func() (bool, error) {
		podList := &corev1.PodList{}
		podListOptions := []cr.ListOption{}
		for _, filter := range filterLabels {
			parsedLabel, err := labels.Parse(filter)
			if err != nil {
				logger.Log("Failed to parse label '%s', skipping...", filter)
				continue
			}
			podListOptions = append(podListOptions, &cr.ListOptions{LabelSelector: parsedLabel})
		}
		if err := wcClient.List(ctx, podList, podListOptions...); err != nil {
			return false, err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				logger.Log("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
				return false, fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		logger.Log("All (%d) pods currently in a running or completed state", len(podList.Items))
		return true, nil
	}
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
			// It's a Karpenter node pool — nodes are provisioned on-demand when pods are pending.
			// We don't require Karpenter nodes in this basic check since they only appear when
			// workloads overflow the ASG pool. Karpenter is validated later by the scale test
			// which deploys enough replicas to force Karpenter to provision nodes.
			maxNodes += 99
			continue
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

// CheckMachinePoolsReadyAndRunning checks if all MachinePool resources are in Running phase with all replicas available.
func CheckMachinePoolsReadyAndRunning(ctx context.Context, mcClient *client.Client, clusterName string, clusterNamespace string) func() error {
	return func() error {
		machinePools := &capi.MachinePoolList{}
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

		allReady := true
		for _, mp := range machinePools.Items {
			phase := capi.MachinePoolPhase(mp.Status.Phase)
			isRunning := phase == capi.MachinePoolPhaseRunning

			var desired, available int32
			if mp.Spec.Replicas != nil {
				desired = *mp.Spec.Replicas
			}
			if mp.Status.AvailableReplicas != nil {
				available = *mp.Status.AvailableReplicas
			}
			replicasReady := available >= desired && desired > 0

			logger.Log(
				"MachinePool '%s/%s': phase=%s, replicas=%d/%d available",
				mp.Namespace, mp.Name, mp.Status.Phase, available, desired)

			if !isRunning || !replicasReady {
				allReady = false
			}
		}

		if !allReady {
			return fmt.Errorf("not all MachinePools are ready and running")
		}

		return nil
	}
}
