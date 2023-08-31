package standard

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/clustertest/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

type ClusterValues struct {
	ControlPlane application.ControlPlane `yaml:"controlPlane"`
	NodePools    []application.NodePool   `yaml:"nodePools"`
}

var _ = Describe("Common tests", func() {
	common.Run(&common.TestConfig{
		BastionSupported:     true,
		ExternalDnsSupported: false,
	})

	It("has all the control-plane nodes running", func() {
		values := &ClusterValues{}
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
		values := &ClusterValues{}
		err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
		Expect(err).NotTo(HaveOccurred())

		wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.Consistent(CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})
})

func CheckWorkerNodesReady(wcClient *client.Client, values *ClusterValues) func() error {
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

	workersFunc := wait.AreNumNodesReadyWithinRange(context.Background(), wcClient, expectedNodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})

	return func() error {
		ok, err := workersFunc()
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return err
	}
}
