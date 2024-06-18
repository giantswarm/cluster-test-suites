package upgrade

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadm "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type TestConfig struct {
	ControlPlaneNodesTimeout time.Duration
	WorkerNodesTimeout       time.Duration
}

func NewTestConfigWithDefaults() *TestConfig {
	return &TestConfig{
		ControlPlaneNodesTimeout: 15 * time.Minute,
		WorkerNodesTimeout:       15 * time.Minute,
	}
}

func Run(cfg *TestConfig) {
	Context("upgrade", func() {
		var cluster *application.Cluster
		var wcClient *client.Client

		BeforeEach(func() {
			var err error
			cluster = state.GetCluster()
			wcClient, err = state.GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())
		})

		It("has all the control-plane nodes running", func() {
			replicas, err := state.GetFramework().GetExpectedControlPlaneReplicas(state.GetContext(), state.GetCluster().Name, state.GetCluster().GetNamespace())
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, int(replicas)), 12, 5*time.Second)).
				WithTimeout(cfg.ControlPlaneNodesTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := state.GetFramework().MC().GetHelmValues(cluster.Name, cluster.GetNamespace(), values)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
				WithTimeout(cfg.WorkerNodesTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its Deployments Ready (means all replicas are running)", func() {
			Eventually(wait.Consistent(common.CheckAllDeploymentsReady(wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its StatefulSets Ready (means all replicas are running)", func() {
			Eventually(wait.Consistent(common.CheckAllStatefulSetsReady(wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its DaemonSets Ready (means all daemon pods are running)", func() {
			Eventually(wait.Consistent(common.CheckAllDaemonSetsReady(wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all of its Pods in the Running state", func() {
			Eventually(wait.Consistent(common.CheckAllPodsSuccessfulPhase(wcClient), 10, time.Second)).
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

			machinePools, err := state.GetFramework().GetMachinePools(context.Background(), cluster.Name, cluster.GetNamespace())
			Expect(err).NotTo(HaveOccurred())
			if len(machinePools) == 0 {
				Skip("Machine pools are not found")
			}

			logger.Log("checking machine pools for cluster %s/%s", cluster.GetNamespace(), cluster.Name)
			Eventually(wait.Consistent(common.CheckMachinePoolsReadyAndRunning(mcClient, cluster.Name, cluster.GetNamespace()), 10, time.Second)).
				WithTimeout(30 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("should apply new version successfully", func() {
			// Set app versions to `""` so that it makes use of the overrides set in the `E2E_OVERRIDE_VERSIONS` environment var
			cluster = cluster.WithAppVersions("", "")
			applyCtx, cancelApplyCtx := context.WithTimeout(state.GetContext(), 20*time.Minute)
			defer cancelApplyCtx()

			builtCluster, _ := cluster.Build()

			_, err := state.GetFramework().ApplyBuiltCluster(applyCtx, builtCluster)
			Expect(err).NotTo(HaveOccurred())

			skipDefaultAppsApp, err := cluster.UsesUnifiedClusterApp()
			Expect(err).NotTo(HaveOccurred())

			if !skipDefaultAppsApp {
				Eventually(
					wait.IsAppVersion(state.GetContext(), state.GetFramework().MC(), builtCluster.DefaultApps.App.Name, builtCluster.DefaultApps.App.Namespace, builtCluster.DefaultApps.App.Spec.Version),
					10*time.Minute, 5*time.Second,
				).Should(BeTrue())

				Eventually(
					wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), builtCluster.DefaultApps.App.Name, builtCluster.DefaultApps.App.Namespace),
					10*time.Minute, 5*time.Second,
				).Should(BeTrue())
			}

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
	})
}
