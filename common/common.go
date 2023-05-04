package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	Framework *clustertest.Framework
	Cluster   *application.Cluster
)

func Run() {
	var wcClient *client.Client

	BeforeEach(func() {
		var err error

		wcClient, err = Framework.WC(Cluster.Name)
		if err != nil {
			Fail(err.Error())
		}
	})

	It("should be able to connect to MC cluster", func() {
		Expect(Framework.MC().CheckConnection()).To(Succeed())
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

	It("has all the control-plane nodes running", func() {
		values := &application.ClusterValues{}
		err := Framework.MC().GetHelmValues(Cluster.Name, Cluster.Namespace, values)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.Consistent(checkControlPlaneNodesReady(wcClient, values), 12, 5*time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("has all the worker nodes running", func() {
		values := &application.ClusterValues{}
		err := Framework.MC().GetHelmValues(Cluster.Name, Cluster.Namespace, values)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.Consistent(checkWorkerNodesReady(wcClient, values), 12, 5*time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})
}

func checkControlPlaneNodesReady(wcClient *client.Client, values *application.ClusterValues) func() error {
	expectedNodes := values.ControlPlane.Replicas
	controlPlaneFunc := wait.IsNumNodesReady(context.Background(), wcClient, expectedNodes, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""})

	return func() error {
		ok, err := controlPlaneFunc()
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return err
	}
}

func checkWorkerNodesReady(wcClient *client.Client, values *application.ClusterValues) func() error {
	minNodes := 0
	maxNodes := 0
	for _, pool := range values.NodePools {
		minNodes += pool.MinSize
		maxNodes += pool.MaxSize
	}
	expectedNodes := wait.Range{
		Min: minNodes,
		Max: maxNodes,
	}

	workersFunc := wait.AreNodesReadyWithinRange(context.Background(), wcClient, expectedNodes, &cr.MatchingLabels{"node-role.kubernetes.io/worker": ""})

	return func() error {
		ok, err := workersFunc()
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return err
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
