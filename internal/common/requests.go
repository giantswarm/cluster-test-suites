package common

import (
	"context"
	"time"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	MaximumCPURequestUtilization    float64 = 0.8
	MaximumMemoryRequestUtilization float64 = 0.8
)

func runRequests() {
	Context("requests", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("has spare CPU request capacity on the control plane nodes", func() {
			Eventually(checkControlPlaneNodesHaveSpareRequestCapacity(wcClient, corev1.ResourceCPU, MaximumCPURequestUtilization)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(BeTrue())
		})

		It("has spare memory request capacity on the control plane nodes", func() {
			Eventually(checkControlPlaneNodesHaveSpareRequestCapacity(wcClient, corev1.ResourceMemory, MaximumMemoryRequestUtilization)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(BeTrue())
		})
	})
}

func checkControlPlaneNodesHaveSpareRequestCapacity(wcClient *client.Client, resourceName corev1.ResourceName, utilization float64) wait.WaitCondition {
	return func() (bool, error) {
		totalRequested := resource.NewQuantity(0, resource.DecimalSI)
		totalAllocatable := resource.NewQuantity(0, resource.DecimalSI)

		nodes := &corev1.NodeList{}
		err := wcClient.List(context.Background(), nodes, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""})
		if err != nil {
			return false, err
		}

		for _, node := range nodes.Items {
			pods := &corev1.PodList{}
			err := wcClient.List(context.Background(), pods, &cr.MatchingFields{"spec.nodeName": node.Name})
			if err != nil {
				return false, err
			}

			for _, pod := range pods.Items {
				for _, container := range pod.Spec.Containers {
					totalRequested.Add(container.Resources.Requests[resourceName])
				}
			}

			totalAllocatable.Add(node.Status.Allocatable[resourceName])
		}

		return totalRequested.AsApproximateFloat64() < totalAllocatable.AsApproximateFloat64()*utilization, nil
	}
}
