package common

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/clustertest/v5/pkg/client"
	"github.com/giantswarm/clustertest/v5/pkg/logger"

	"github.com/giantswarm/cluster-test-suites/v7/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
)

// runClusterValues cross-checks the cluster-values ConfigMap that
// cluster-apps-operator emits on the MC against the live state of the workload
// cluster it describes.
//
// The `clusterDNSIP` key is the one that matters. chart-operator runs with
// `hostNetwork: true` and `dnsPolicy: None`, so that value becomes its only
// resolver -- if it does not point at the real coredns Service, chart-operator
// cannot resolve the OCI registry and every chart pull fails.
//
// The value is derived twice, independently, from the same upstream field
// (`Cluster.spec.clusterNetwork.services.cidrBlocks[0]`):
//
//   - the coredns Service ClusterIP comes from a Helm regex in
//     giantswarm/cluster (`cluster.internal.apps.coredns.dns`), passed to
//     coredns-app as `service.clusterIP`;
//   - `clusterDNSIP` comes from Go code in cluster-apps-operator
//     (`key.DNSIP`, using net.ParseCIDR).
//
// Nothing couples the two, which is exactly why comparing them is worth doing.
// In giantswarm/giantswarm#37031 the Go path silently diverged -- it was still
// reading a KubeadmControlPlane field that the CAPI v1beta2 migration had
// removed, so it fell back to a hardcoded installation default -- while the
// Helm path stayed correct. Unit tests could not catch it: their fixtures
// supplied the field by hand, so they never noticed the real producer had
// stopped emitting it. This comparison would have failed immediately, on every
// affected cluster.
func runClusterValues() {
	Context("cluster values", func() {
		var (
			wcClient     *client.Client
			clusterDNSIP string
		)

		BeforeEach(func() {
			helper.SetResponsibleTeam(helper.TeamHoneybadger)

			// Reading the ConfigMap hits the MC API and can transiently fail;
			// retry so a blip doesn't fail the spec.
			configMapName := fmt.Sprintf("%s-cluster-values", state.GetCluster().Name)
			Eventually(func() error {
				cm, err := state.GetFramework().GetConfigMap(
					state.GetContext(),
					configMapName,
					state.GetCluster().GetNamespace(),
				)
				if err != nil {
					return err
				}

				// Only the one key is modelled here on purpose.
				// application.ClusterValues describes the cluster app's own
				// values, not the operator-emitted ConfigMap, and does not
				// carry clusterDNSIP.
				var values struct {
					ClusterDNSIP string `json:"clusterDNSIP"`
				}
				err = yaml.Unmarshal([]byte(cm.Data["values"]), &values)
				if err != nil {
					return err
				}

				if values.ClusterDNSIP == "" {
					return fmt.Errorf("clusterDNSIP is not set in configmap %s", configMapName)
				}

				clusterDNSIP = values.ClusterDNSIP
				return nil
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())

			// Building the WC client can transiently fail. Retry so a blip
			// doesn't fail the whole spec.
			Eventually(func() error {
				var err error
				wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
				return err
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("emits a clusterDNSIP that matches the coredns Service", func() {
			var coreDNSIP string

			// Fetched by name, which is the stable contract: coredns-app names
			// the Service `coredns` in kube-system, verified on CAPA, CAPZ and
			// CAPVCD clusters. Deliberately no fallback to a second lookup --
			// a test that quietly falls back to another source of truth is how
			// this class of bug hides in the first place. If a provider ever
			// differs, this should fail loudly and be fixed here.
			Eventually(func() error {
				svc := &corev1.Service{}
				err := wcClient.Get(state.GetContext(), types.NamespacedName{
					Name:      "coredns",
					Namespace: "kube-system",
				}, svc)
				if err != nil {
					return err
				}

				if svc.Spec.ClusterIP == "" || svc.Spec.ClusterIP == corev1.ClusterIPNone {
					return fmt.Errorf("coredns Service in kube-system has no ClusterIP")
				}

				coreDNSIP = svc.Spec.ClusterIP
				return nil
			}).
				WithTimeout(2 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())

			logger.Log("cluster-values clusterDNSIP=%s, coredns Service ClusterIP=%s", clusterDNSIP, coreDNSIP)

			Expect(clusterDNSIP).To(Equal(coreDNSIP),
				"clusterDNSIP in the cluster-values ConfigMap must equal the coredns Service ClusterIP. "+
					"chart-operator uses it as its only resolver, so a mismatch breaks every chart pull "+
					"(see giantswarm/giantswarm#37031).")
		})

		It("propagates clusterDNSIP into the chart-operator resolver", func() {
			deployment := &appsv1.Deployment{}
			err := wcClient.Get(state.GetContext(), types.NamespacedName{
				Name:      "chart-operator",
				Namespace: "giantswarm",
			}, deployment)
			if apierrors.IsNotFound(err) {
				Skip("chart-operator is not deployed on this cluster")
			}
			Expect(err).NotTo(HaveOccurred())

			// chart-operator's chart omits the whole dnsConfig block when it
			// has no external resolver to fall back to (externalDNSIP: "",
			// which yields dnsPolicy: Default) or when it runs on a management
			// cluster (dnsPolicy: ClusterFirst). Observed in practice on CAPZ
			// and CAPVCD clusters, so this skip is load-bearing, not defensive
			// boilerplate. The assertion above still covers those clusters.
			dnsConfig := deployment.Spec.Template.Spec.DNSConfig
			if dnsConfig == nil || len(dnsConfig.Nameservers) == 0 {
				Skip("chart-operator has no dnsConfig.nameservers on this cluster")
			}

			logger.Log("chart-operator dnsConfig.nameservers=%v", dnsConfig.Nameservers)

			Expect(dnsConfig.Nameservers[0]).To(Equal(clusterDNSIP),
				"chart-operator's first nameserver must be the clusterDNSIP from the cluster-values ConfigMap. "+
					"A mismatch means the value did not propagate, or the Deployment was patched out of band.")
		})
	})
}
