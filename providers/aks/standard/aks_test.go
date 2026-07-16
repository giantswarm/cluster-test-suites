package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
)

var _ = Describe("Common tests", func() {
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
	common.Run(cfg)
})
