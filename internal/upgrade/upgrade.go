package upgrade

import (
	"context"
	"time"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Run() {
	Context("upgrade", func() {
		var cluster *application.Cluster

		BeforeEach(func() {
			cluster = state.GetCluster()
		})

		It("has all the control-plane nodes running", func() {
			replicas, err := state.GetFramework().GetExpectedControlPlaneReplicas(state.GetContext(), state.GetCluster().Name, state.GetCluster().GetNamespace())
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, int(replicas)), 12, 5*time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := state.GetFramework().MC().GetHelmValues(cluster.Name, cluster.GetNamespace(), values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("should upgrade successfully", func() {
			// Set app versions to `""` so that it makes use of the overrides set in the `E2E_OVERRIDE_VERSIONS` environment var
			cluster = cluster.WithAppVersions("", "")
			applyCtx, cancelApplyCtx := context.WithTimeout(state.GetContext(), 20*time.Minute)
			defer cancelApplyCtx()

			_, err := state.GetFramework().ApplyCluster(applyCtx, cluster)
			Expect(err).NotTo(HaveOccurred())

			clusterApp, _, defaultAppsApp, _, _ := cluster.Build()

			Eventually(
				wait.IsAppVersion(state.GetContext(), state.GetFramework().MC(), defaultAppsApp.Name, defaultAppsApp.Namespace, defaultAppsApp.Spec.Version),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), defaultAppsApp.Name, defaultAppsApp.Namespace),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppVersion(state.GetContext(), state.GetFramework().MC(), clusterApp.Name, clusterApp.Namespace, clusterApp.Spec.Version),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), clusterApp.Name, clusterApp.Namespace),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())
		})

		It("has all the control-plane nodes running", func() {
			replicas, err := state.GetFramework().GetExpectedControlPlaneReplicas(state.GetContext(), state.GetCluster().Name, state.GetCluster().GetNamespace())
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, int(replicas)), 12, 5*time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := state.GetFramework().MC().GetHelmValues(cluster.Name, cluster.GetNamespace(), values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
}
