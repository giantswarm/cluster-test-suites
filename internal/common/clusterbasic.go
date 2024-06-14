package common

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func runClusterBasic() {
	Context("cluster basic", func() {
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

			Eventually(wait.Consistent(CheckControlPlaneNodesReady(wcClient, int(replicas)), 12, 5*time.Second)).
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

			Eventually(wait.Consistent(CheckWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
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
	})
}

func CheckControlPlaneNodesReady(wcClient *client.Client, expectedNodes int) func() error {
	controlPlaneFunc := wait.AreNumNodesReady(context.Background(), wcClient, expectedNodes, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""})

	return func() error {
		ok, err := controlPlaneFunc()
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return err
	}
}

func CheckWorkerNodesReady(wcClient *client.Client, values *application.ClusterValues) func() error {
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
