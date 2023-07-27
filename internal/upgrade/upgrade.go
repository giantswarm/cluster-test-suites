package upgrade

import (
	"context"
	"time"

	"github.com/giantswarm/cluster-test-suites/common"
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	Framework *clustertest.Framework
	Cluster   *application.Cluster
)

func Run() {
	Context("upgrade", func() {
		ctx := context.Background()

		It("has all the control-plane nodes running", func() {
			values := &application.ClusterValues{}
			err := Framework.MC().GetHelmValues(Cluster.Name, Cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := Framework.WC(Cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, values.ControlPlane), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := Framework.MC().GetHelmValues(Cluster.Name, Cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := Framework.WC(Cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("should upgrade successfully", func() {
			// Set app versions to `""` so that it makes use of the overrides set in the `E2E_OVERRIDE_VERSIONS` environment var
			Cluster = Cluster.WithAppVersions("", "")
			applyCtx, cancelApplyCtx := context.WithTimeout(ctx, 20*time.Minute)
			defer cancelApplyCtx()

			_, err := Framework.ApplyCluster(applyCtx, Cluster)
			Expect(err).NotTo(HaveOccurred())

			clusterApp, _, defaultAppsApp, _, _ := Cluster.Build()

			Eventually(
				wait.IsAppVersion(ctx, Framework.MC(), defaultAppsApp.Name, defaultAppsApp.Namespace, defaultAppsApp.Spec.Version),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppStatus(ctx, Framework.MC(), defaultAppsApp.Name, defaultAppsApp.Namespace, "deployed"),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppVersion(ctx, Framework.MC(), clusterApp.Name, clusterApp.Namespace, clusterApp.Spec.Version),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())

			Eventually(
				wait.IsAppStatus(ctx, Framework.MC(), clusterApp.Name, clusterApp.Namespace, "deployed"),
				10*time.Minute, 5*time.Second,
			).Should(BeTrue())
		})

		It("has all the control-plane nodes running", func() {
			values := &application.ClusterValues{}
			err := Framework.MC().GetHelmValues(Cluster.Name, Cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := Framework.WC(Cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, values.ControlPlane), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all the worker nodes running", func() {
			values := &application.ClusterValues{}
			err := Framework.MC().GetHelmValues(Cluster.Name, Cluster.Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			wcClient, err := Framework.WC(Cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
}
