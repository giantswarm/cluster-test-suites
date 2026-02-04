package upgrade

import (
	"context"
	"os"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadm "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v3/pkg/application"
	"github.com/giantswarm/clustertest/v3/pkg/client"
	"github.com/giantswarm/clustertest/v3/pkg/logger"
	"github.com/giantswarm/clustertest/v3/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
	"github.com/giantswarm/cluster-test-suites/v3/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v3/internal/state"
	"github.com/giantswarm/cluster-test-suites/v3/internal/timeout"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

// nodeInfo tracks node identity for reliable roll detection
type nodeInfo struct {
	Name      string
	UID       types.UID
	CreatedAt time.Time
}

type TestConfig struct {
	ControlPlaneNodesTimeout time.Duration
	WorkerNodesTimeout       time.Duration
	MinimalCluster           bool
}

func NewTestConfigWithDefaults() *TestConfig {
	return &TestConfig{
		ControlPlaneNodesTimeout: 15 * time.Minute,
		WorkerNodesTimeout:       15 * time.Minute,
		MinimalCluster:           false,
	}
}

func Run(cfg *TestConfig) {
	Context("upgrade", func() {
		var cluster *application.Cluster
		var wcClient *client.Client
		var preUpgradeControlPlaneResourceGeneration int64
		var initialNodes map[string]nodeInfo
		var initialNodeCount int

		BeforeAll(func() {
			var err error
			cluster = state.GetCluster()
			preUpgradeControlPlane, err := state.GetFramework().GetKubeadmControlPlane(state.GetContext(), cluster.Name, cluster.GetNamespace())
			Expect(err).NotTo(HaveOccurred())
			if preUpgradeControlPlane == nil {
				preUpgradeControlPlaneResourceGeneration = 0
			} else {
				preUpgradeControlPlaneResourceGeneration = preUpgradeControlPlane.GetGeneration()
			}
			wcClient, err = state.GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			nodes := &corev1.NodeList{}
			err = wcClient.List(state.GetContext(), nodes)
			Expect(err).NotTo(HaveOccurred())
			initialNodes = make(map[string]nodeInfo, len(nodes.Items))
			for _, node := range nodes.Items {
				initialNodes[node.Name] = nodeInfo{
					Name:      node.Name,
					UID:       node.UID,
					CreatedAt: node.CreationTimestamp.Time,
				}
			}
			initialNodeCount = len(nodes.Items)
			logger.Log("Node roll detection - Captured %d initial nodes before upgrade", initialNodeCount)
		})

		BeforeEach(func() {
			var err error
			cluster = state.GetCluster()
			wcClient, err = state.GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())
		})

		It("has all the control-plane nodes running", func() {
			replicas, err := state.GetFramework().GetExpectedControlPlaneReplicas(state.GetContext(), state.GetCluster().Name, state.GetCluster().GetNamespace())
			Expect(err).NotTo(HaveOccurred())

			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreNumNodesReady(state.GetContext(), wcClient, int(replicas), &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
					12,
					5*time.Second,
				)).
				WithTimeout(cfg.ControlPlaneNodesTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := state.GetFramework().MC().GetHelmValues(cluster.Name, cluster.GetNamespace(), values)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(state.GetContext(), wcClient, values), 12, 5*time.Second)).
				WithTimeout(cfg.WorkerNodesTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has Cluster Ready condition with Status='True'", func() {
			// Overriding the default timeout, when clusterReadyTimeout is set
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

			Eventually(wait.Consistent(common.CheckMachinePoolsReadyAndRunning(state.GetContext(), mcClient, cluster.Name, cluster.GetNamespace()), 5, 5*time.Second)).
				WithTimeout(30 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		common.RunApps(cfg.MinimalCluster)

		It("has all its Deployments Ready (means all replicas are running)", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllDeploymentsReady(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its StatefulSets Ready (means all replicas are running)", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllStatefulSetsReady(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its DaemonSets Ready (means all daemon pods are running)", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllDaemonSetsReady(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all of its Pods in the Running state", func() {
			Eventually(
				wait.ConsistentWaitCondition(
					wait.AreAllPodsInSuccessfulPhase(state.GetContext(), wcClient),
					10,
					time.Second,
				)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("should apply new version successfully", func() {
			cluster = cluster.
				// Set app versions to `""` so that it makes use of the overrides set in the `E2E_OVERRIDE_VERSIONS` environment var
				WithAppVersions("").
				// Set release versions to `""` so that it makes use of the overrides set in the `E2E_RELEASE_VERSION` environment var
				WithRelease(application.ReleasePair{Version: "", Commit: ""})
			applyCtx, cancelApplyCtx := context.WithTimeout(state.GetContext(), 20*time.Minute)
			defer cancelApplyCtx()

			builtCluster, _ := cluster.Build()

			_, err := state.GetFramework().ApplyBuiltCluster(applyCtx, builtCluster)
			Expect(err).NotTo(HaveOccurred())

			Eventually(
				wait.IsAppVersion(state.GetContext(), state.GetFramework().MC(), builtCluster.Cluster.App.Name, builtCluster.Cluster.App.Namespace, builtCluster.Cluster.App.Spec.Version),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), builtCluster.Cluster.App.Name, builtCluster.Cluster.App.Namespace),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())
		})

		It("successfully finishes control plane nodes rolling update if it is needed", func() {
			// Check MachinesSpecUpToDate condition on KubeadmControlPlane. Repeat the check 5 times, with some waiting time,
			// so Cluster API controllers have time to react to upgrade (it is usually instantaneous).
			numberOfChecks := 18
			waitBetweenChecks := 10 * time.Second
			controlPlaneRollingUpdateStarted := false

			for i := 0; i < numberOfChecks; i++ {
				controlPlane, err := state.GetFramework().GetKubeadmControlPlane(state.GetContext(), cluster.Name, cluster.GetNamespace())
				Expect(err).NotTo(HaveOccurred())

				if controlPlane == nil {
					Skip("Control plane resource not found (assuming this is a managed cluster)")
				}

				if controlPlane.GetGeneration() == preUpgradeControlPlaneResourceGeneration {
					Skip("Control plane resource generation did not change, skipping rolling update test")
				}

				if capiconditions.IsFalse(controlPlane, kubeadm.MachinesSpecUpToDateCondition) &&
					capiconditions.GetReason(controlPlane, kubeadm.MachinesSpecUpToDateCondition) == kubeadm.RollingUpdateInProgressReason {
					controlPlaneRollingUpdateStarted = true
					break
				} else {
					machinesSpecUpToDateCondition := capiconditions.Get(controlPlane, kubeadm.MachinesSpecUpToDateCondition)
					if machinesSpecUpToDateCondition == nil {
						logger.Log("KubeadmControlPlane condition %s is still not set on the KubeadmControlPlane resource, expected condition with Status='False' and Reason='%s'", kubeadm.MachinesSpecUpToDateCondition, kubeadm.RollingUpdateInProgressReason)
					} else {
						logger.Log("KubeadmControlPlane condition %s has Status='%s' and Reason='%s', expected condition with Status='False' and Reason='%s'", kubeadm.MachinesSpecUpToDateCondition, machinesSpecUpToDateCondition.Status, machinesSpecUpToDateCondition.Reason, kubeadm.RollingUpdateInProgressReason)
					}
				}

				time.Sleep(waitBetweenChecks)
			}

			if !controlPlaneRollingUpdateStarted {
				Skip("Control plane nodes rolling update is not happening")
			}

			mcClient := state.GetFramework().MC()
			Eventually(
				wait.IsKubeadmControlPlaneConditionSet(state.GetContext(), mcClient, cluster.Name, cluster.GetNamespace(), kubeadm.MachinesSpecUpToDateCondition, corev1.ConditionTrue, ""),
				30*time.Minute,
				30*time.Second,
			).Should(BeTrue())
		})

		It("detects if nodes were rolled", func() {
			// Run node roll detection at the very end of the upgrade context, right before cluster deletion.
			// This ensures that old nodes have been fully removed before we check if nodes were rolled.
			// On providers with single CP nodes (like CAPV), old nodes might still be around when
			// KubeadmControlPlane reports as up-to-date, so we need to wait until the very end.
			if os.Getenv("SKIP_NODE_ROLL_DETECTION") == "true" {
				Skip("Node roll detection is disabled for this test suite.")
			}

			// Log initial nodes with UIDs for debugging
			initialNodeNames := make([]string, 0, len(initialNodes))
			for nodeName := range initialNodes {
				initialNodeNames = append(initialNodeNames, nodeName)
			}
			sort.Strings(initialNodeNames)
			logger.Log("Node roll detection - Initial nodes (%d): %v", len(initialNodes), initialNodeNames)
			for _, name := range initialNodeNames {
				info := initialNodes[name]
				logger.Log("  - %s (UID: %s, Created: %s)", name, info.UID, info.CreatedAt.Format(time.RFC3339))
			}

			rolled := false
			var rolledNodes []string
			var replacedNodes []string
			timeout := 5 * time.Minute
			startTime := time.Now()

			// Poll for node rolling without failing the test if it doesn't happen (e.g. scale-up)
			for {
				nodes := &corev1.NodeList{}
				if err := wcClient.List(state.GetContext(), nodes); err != nil {
					logger.Log("Failed to list nodes for roll detection: %v", err)
				} else {
					// Build current node map with UIDs
					currentNodes := make(map[string]nodeInfo, len(nodes.Items))
					currentNodeNames := make([]string, 0, len(nodes.Items))
					for _, node := range nodes.Items {
						currentNodes[node.Name] = nodeInfo{
							Name:      node.Name,
							UID:       node.UID,
							CreatedAt: node.CreationTimestamp.Time,
						}
						currentNodeNames = append(currentNodeNames, node.Name)
					}

					rolledNodes = nil
					replacedNodes = nil

					// Check each initial node
					for nodeName, initialInfo := range initialNodes {
						if currentInfo, exists := currentNodes[nodeName]; exists {
							// Node with same name exists - check if it was replaced (different UID)
							if currentInfo.UID != initialInfo.UID {
								replacedNodes = append(replacedNodes, nodeName)
								rolled = true
								logger.Log("Node %s was replaced (UID changed: %s -> %s)", nodeName, initialInfo.UID, currentInfo.UID)
							}
						} else {
							// Node no longer exists - it was rolled away
							rolledNodes = append(rolledNodes, nodeName)
							rolled = true
							logger.Log("Node %s was rolled (not found in current nodes: %v)", nodeName, currentNodeNames)
						}
					}

					// Also check for new nodes that appeared (could indicate scale-up vs roll)
					// If node count increased and no nodes were removed, it's likely a scale-up, not a roll
					if len(nodes.Items) > initialNodeCount && len(rolledNodes) == 0 && len(replacedNodes) == 0 {
						logger.Log("Node count increased from %d to %d without removing existing nodes - likely a scale-up, not a roll",
							initialNodeCount, len(nodes.Items))
					}
				}

				if rolled || time.Since(startTime) >= timeout {
					break
				}
				time.Sleep(10 * time.Second)
			}

			// Final summary
			if rolled {
				logger.Log("Node roll detection result: rolled=true (removed: %v, replaced: %v)", rolledNodes, replacedNodes)
			} else {
				logger.Log("Node roll detection result: rolled=false - all %d initial nodes still present with same UIDs", len(initialNodes))
			}
			helper.RecordNodeRolling(rolled)
		})
	})
}
