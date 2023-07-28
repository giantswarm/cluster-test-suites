package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func runBasic() {
	Context("basic", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = state.Get().GetFramework().WC(state.Get().GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("should be able to connect to MC cluster", func() {
			Expect(state.Get().GetFramework().MC().CheckConnection()).To(Succeed())
		})

		It("should be able to connect to WC cluster", func() {
			Expect(wcClient.CheckConnection()).To(Succeed())
		})

		It("has all of it's Pods in the Running state", func() {
			Eventually(wait.Consistent(checkAllPodsSuccessfulPhase(wcClient), 10, time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
}

func CheckControlPlaneNodesReady(wcClient *client.Client, values application.ControlPlane) func() error {
	expectedNodes := values.Replicas
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

func checkAllPodsSuccessfulPhase(wcClient *client.Client) func() error {
	return func() error {
		podList := &corev1.PodList{}
		err := wcClient.List(context.Background(), podList)
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				return fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		return nil
	}
}
