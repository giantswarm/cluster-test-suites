package cilium_eni_mode

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

var _ = Describe("Cilium ENI mode tests", Ordered, func() {
	common.Run(&common.TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
	})

	runSecondaryPodIPs()
})

func runSecondaryPodIPs() {
	It("assigns IP addresses from secondary VPC CIDR to pods", func() {
		wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
		if err != nil {
			Fail(err.Error())
		}

		podList := &corev1.PodList{}
		err = wcClient.List(context.Background(), podList)
		if err != nil {
			Fail(err.Error())
		}

		numPodsWithSecondaryCIDRPodIP := 0
		foundPodIPs := []string{}
		for _, pod := range podList.Items {
			if strings.HasPrefix(pod.Status.PodIP, "10.1.") {
				numPodsWithSecondaryCIDRPodIP++
			}

			foundPodIPs = append(foundPodIPs, pod.Status.PodIP)
		}

		Expect(numPodsWithSecondaryCIDRPodIP).Should(BeNumerically(">=", 5), fmt.Sprintf("Many pods (except those on host network, for example) should have an IP assigned from the secondary CIDR block 10.1.0.0/16. Found these pod IPs: %s", strings.Join(foundPodIPs, ", ")))
	})
}
