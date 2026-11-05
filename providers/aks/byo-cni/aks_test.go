package byo_cni

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

const podCIDR = "100.64.0.0/16"

var _ = Describe("BYO CNI tests", func() {
	BeforeEach(func() {
		// AKS clusters take longer to fully settle after being created.
		state.SetTestTimeout(timeout.ClusterReadyTimeout, timeout.AKSClusterReady)
	})

	cfg := common.NewTestConfigWithDefaults()
	// AKS has a managed control plane, so there are no metrics for the k8s control plane components.
	cfg.ControlPlaneMetricsSupported = false
	// The cluster-autoscaler is not deployed on AKS (autoscaling is handled by the managed node pools).
	cfg.AutoScalingSupported = false
	// Disabled until wildcard ingress support is added (same as CAPZ).
	cfg.ExternalDnsSupported = false
	cfg.GatewayAPISupported = false
	// AKS has a managed control plane with its own Kubernetes API endpoint, so
	// our DNS controllers don't set up an A record for it.
	cfg.APIServerDNSRecordSupported = false
	// AKS ships its own managed DNS add-on, which names the Service `kube-dns`
	// instead of `coredns`.
	cfg.DNSServiceName = "kube-dns"
	cfg.IgnoredRestartedPodsLabelFilters = common.LabelFilter{
		// AKS manages the Azure Workload Identity webhook, and we have no control over when it gets deployed.
		// It will crash loop for a while until our network policies get deployed.
		"azure-workload-identity.io/system": []string{"true"},
	}
	common.Run(cfg)

	runPodIPs()
})

func runPodIPs() {
	It("assigns pod IPs from the configured pod CIDR", func() {
		_, pods, err := net.ParseCIDR(podCIDR)
		Expect(err).NotTo(HaveOccurred())

		wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
		Expect(err).NotTo(HaveOccurred())

		podList := &corev1.PodList{}
		Expect(wcClient.List(state.GetContext(), podList)).To(Succeed())

		checked := 0
		for _, pod := range podList.Items {
			if pod.Spec.HostNetwork || pod.Status.PodIP == "" {
				continue
			}
			checked++
			Expect(pods.Contains(net.ParseIP(pod.Status.PodIP))).To(BeTrue(),
				fmt.Sprintf("pod %s/%s has IP %s outside of %s", pod.Namespace, pod.Name, pod.Status.PodIP, podCIDR))
		}
		Expect(checked).To(BeNumerically(">", 0), "no pods with a pod network IP found")
	})
}
