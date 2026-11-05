package overlay_acns

import (
	"fmt"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	clustertestclient "github.com/giantswarm/clustertest/v5/pkg/client"

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

const (
	podCIDR      = "100.64.0.0/16"
	serviceCIDR  = "172.16.0.0/16"
	dnsServiceIP = "172.16.0.10"
)

var hubbleRelayLabels = cr.MatchingLabels{"k8s-app": "hubble-relay"}

var _ = Describe("Overlay ACNS tests", func() {
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
		// AKS may start the ACNS hubble-relay before our network policies allow its traffic,
		// so it can crash loop at first. The hubble-relay test below checks it separately.
		"k8s-app": []string{"hubble-relay"},
	}
	common.Run(cfg)

	runPodIPs()
	runServiceIPs()
	runHubbleRelay()
})

func wcClient() *clustertestclient.Client {
	c, err := state.GetFramework().WC(state.GetCluster().Name)
	Expect(err).NotTo(HaveOccurred())
	return c
}

func mustParseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	Expect(err).NotTo(HaveOccurred())
	return ipNet
}

func runPodIPs() {
	It("assigns pod IPs from the overlay pod CIDR", func() {
		pods := mustParseCIDR(podCIDR)

		podList := &corev1.PodList{}
		Expect(wcClient().List(state.GetContext(), podList)).To(Succeed())

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

func runServiceIPs() {
	It("assigns Service IPs from the service CIDR", func() {
		services := mustParseCIDR(serviceCIDR)
		c := wcClient()

		kubernetesSvc := &corev1.Service{}
		Expect(c.Get(state.GetContext(), cr.ObjectKey{Namespace: "default", Name: "kubernetes"}, kubernetesSvc)).To(Succeed())
		Expect(services.Contains(net.ParseIP(kubernetesSvc.Spec.ClusterIP))).To(BeTrue(),
			fmt.Sprintf("Service default/kubernetes has IP %s outside of %s", kubernetesSvc.Spec.ClusterIP, serviceCIDR))

		dnsSvc := &corev1.Service{}
		Expect(c.Get(state.GetContext(), cr.ObjectKey{Namespace: "kube-system", Name: "kube-dns"}, dnsSvc)).To(Succeed())
		Expect(dnsSvc.Spec.ClusterIP).To(Equal(dnsServiceIP))
	})
}

func runHubbleRelay() {
	It("runs the ACNS hubble-relay behind the network policies", func() {
		c := wcClient()

		By("waiting for the allow-hubble-relay CiliumNetworkPolicy")
		Eventually(func() error {
			policy := &unstructured.Unstructured{}
			policy.SetGroupVersionKind(schema.GroupVersionKind{Group: "cilium.io", Version: "v2", Kind: "CiliumNetworkPolicy"})
			return c.Get(state.GetContext(), cr.ObjectKey{Namespace: "kube-system", Name: "allow-hubble-relay"}, policy)
		}).WithTimeout(15 * time.Minute).WithPolling(15 * time.Second).Should(Succeed())

		// AKS may have started hubble-relay before the policy existed, so we recreate it
		// to check that it starts cleanly with the policy in place.
		By("recreating the hubble-relay pods")
		Expect(c.DeleteAllOf(state.GetContext(), &corev1.Pod{}, cr.InNamespace("kube-system"), hubbleRelayLabels)).To(Succeed())

		By("waiting for hubble-relay to be Ready")
		Eventually(hubbleRelayReady(c)).WithTimeout(10 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

		By("checking that hubble-relay stays Ready without restarts")
		Consistently(hubbleRelayReady(c)).WithTimeout(2 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
	})
}

// hubbleRelayReady succeeds when all hubble-relay pods are running, Ready and have not restarted.
func hubbleRelayReady(c *clustertestclient.Client) func() error {
	return func() error {
		podList := &corev1.PodList{}
		if err := c.List(state.GetContext(), podList, cr.InNamespace("kube-system"), hubbleRelayLabels); err != nil {
			return err
		}
		if len(podList.Items) == 0 {
			return fmt.Errorf("no hubble-relay pods found")
		}
		for _, pod := range podList.Items {
			if pod.DeletionTimestamp != nil {
				return fmt.Errorf("pod %s is terminating", pod.Name)
			}
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ready = true
				}
			}
			if !ready {
				return fmt.Errorf("pod %s is not Ready", pod.Name)
			}
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.RestartCount > 0 {
					return fmt.Errorf("container %s of pod %s restarted %d times", cs.Name, pod.Name, cs.RestartCount)
				}
			}
		}
		return nil
	}
}
