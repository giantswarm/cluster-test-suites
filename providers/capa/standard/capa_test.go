package standard

import (
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/ecr"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

var _ = Describe("Common tests", func() {
	BeforeEach(func() {
		// Set higher timeout for deploying apps because Karpenter workers take longer to come up
		state.SetTestTimeout(timeout.DeployApps, time.Minute*30)
	})

	cfg := common.NewTestConfigWithDefaults()
	// Tie the net-exporter / cert-exporter pod-check exclusions to the same release-version
	// gate that decides whether the arm64 node pool is applied (see capa_suite_test.go).
	// Older releases don't get the arm pool, and shouldn't apply the exclusions either.
	// TODO(arm64): drop this gate once v35.0.0 is the minimum release across CI.
	// https://github.com/giantswarm/roadmap/issues/4302
	cfg.ARMNodePoolEnabled = armSupported()
	common.Run(cfg)

	// ECR Credential Provider specific tests
	ecr.Run()

	// TODO(temp): remove after validating crust-gather failure-gating — intentional failure
	It("TEMP: failing pod on WC for crust-gather snapshot validation", func() {
		wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
		Expect(err).NotTo(HaveOccurred())

		ctx := state.GetContext()
		dep := &appsv1.Deployment{
			ObjectMeta: v1.ObjectMeta{Name: "crust-gather-test", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": "crust-gather-test"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: v1.ObjectMeta{Labels: map[string]string{"app": "crust-gather-test"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "test", Image: "invalid.registry.does.not.exist/test:invalid"},
						},
					},
				},
			},
		}
		Expect(wcClient.Create(ctx, dep)).To(Succeed())
		// Eventually times out — pod is stuck in ImagePullBackOff on the WC.
		// The Deployment and its failing pod remain and are captured by crust-gather.
		Eventually(func() bool { return false }).
			WithTimeout(2 * time.Minute).
			Should(BeTrue(), "TEMP: pod is in ImagePullBackOff on WC — check crust-gather WC snapshot")
	})
})
