package upgrade

import (
	"context"
	"time"

	"github.com/giantswarm/cluster-test-suites/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Run() {
	Context("upgrade", func() {
		var cluster *application.Cluster

		BeforeEach(func() {
			cluster = state.Get().GetCluster()
		})

		It("has all the control-plane nodes running", func() {
			values := &application.ClusterValues{}
			err := state.Get().GetFramework().MC().GetHelmValues(cluster.Name, cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.Get().GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, values.ControlPlane), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := state.Get().GetFramework().MC().GetHelmValues(cluster.Name, cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.Get().GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("should upgrade successfully", func() {
			// Set app versions to `""` so that it makes use of the overrides set in the `E2E_OVERRIDE_VERSIONS` environment var
			cluster = cluster.WithAppVersions("", "")
			applyCtx, cancelApplyCtx := context.WithTimeout(state.Get().GetContext(), 20*time.Minute)
			defer cancelApplyCtx()

			_, err := state.Get().GetFramework().ApplyCluster(applyCtx, cluster)
			Expect(err).NotTo(HaveOccurred())

			clusterApp, _, defaultAppsApp, _, _ := cluster.Build()

			Eventually(
				wait.IsAppVersion(state.Get().GetContext(), state.Get().GetFramework().MC(), defaultAppsApp.Name, defaultAppsApp.Namespace, defaultAppsApp.Spec.Version),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppStatus(state.Get().GetContext(), state.Get().GetFramework().MC(), defaultAppsApp.Name, defaultAppsApp.Namespace, "deployed"),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppVersion(state.Get().GetContext(), state.Get().GetFramework().MC(), clusterApp.Name, clusterApp.Namespace, clusterApp.Spec.Version),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppStatus(state.Get().GetContext(), state.Get().GetFramework().MC(), clusterApp.Name, clusterApp.Namespace, "deployed"),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())
		})

		It("has all the control-plane nodes running", func() {
			values := &application.ClusterValues{}
			err := state.Get().GetFramework().MC().GetHelmValues(cluster.Name, cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.Get().GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, values.ControlPlane), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := state.Get().GetFramework().MC().GetHelmValues(cluster.Name, cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := state.Get().GetFramework().WC(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
}
