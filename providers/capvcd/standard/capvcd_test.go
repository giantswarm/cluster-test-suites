package standard

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/common"
)

var _ = PDescribe("Common tests", func() {
	common.Run()

	It("has all the control-plane nodes running", func() {
		values := &application.ClusterValues{}
		err := framework.MC().GetHelmValues(cluster.Name, cluster.Namespace, values)
		Expect(err).NotTo(HaveOccurred())

		wcClient, err := framework.WC(cluster.Name)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, values.ControlPlane), 12, 5*time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("has all the worker nodes running", func() {
		values := &application.ClusterValues{}
		err := framework.MC().GetHelmValues(cluster.Name, cluster.Namespace, values)
		Expect(err).NotTo(HaveOccurred())

		wcClient, err := framework.WC(cluster.Name)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})
})
