package standard

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

var _ = PDescribe("Common tests", func() {
	common.Run(&common.TestConfig{
		BastionSupported:     true,
		ExternalDnsSupported: false,
	})

	It("has all the control-plane nodes running", func() {
		values := &application.ClusterValues{}
		err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
		Expect(err).NotTo(HaveOccurred())

		wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.Consistent(common.CheckControlPlaneNodesReady(wcClient, values.ControlPlane), 12, 5*time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("has all the worker nodes running", func() {
		values := &application.ClusterValues{}
		err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
		Expect(err).NotTo(HaveOccurred())

		wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.Consistent(common.CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})
})
