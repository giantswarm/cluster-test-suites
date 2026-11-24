package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/ecr"
)

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	cfg.GatewayAPISupported = false
	cfg.IgnoredRestartedPods = common.LabelFilter{
		// Excluding karpenter because it's deployed using a HelmRelease and its pods run on the control plane. Because of this the pod is scheduled pretty early in the cluster creation process.
		// Meanwhile, IRSA resources are getting created, but it takes a while. karpenter uses IRSA and can't run until IRSA is ready. Eventually, IRSA is ready, and the pod works normally.
		"app.kubernetes.io/name": []string{"karpenter"},
	}
	common.Run(cfg)

	// ECR Credential Provider specific tests
	ecr.Run()
})
