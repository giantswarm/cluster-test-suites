package upgrade

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	cfg := upgrade.NewTestConfigWithDefaults()
	cfg.ControlPlaneType = upgrade.ControlPlaneTypeAzureManaged
	upgrade.Run(cfg)

	// Finally run the common tests after upgrade is completed
	ccfg := common.NewTestConfigWithDefaults()
	// AKS has a managed control plane, so there are no metrics for the k8s control plane components.
	ccfg.ControlPlaneMetricsSupported = false
	// The cluster-autoscaler is not deployed on AKS (autoscaling is handled by the managed node pools).
	ccfg.AutoScalingSupported = false
	// Disabled until wildcard ingress support is added (same as CAPZ).
	ccfg.ExternalDnsSupported = false
	ccfg.GatewayAPISupported = false
	// AKS has a managed control plane with its own Kubernetes API endpoint, so
	// our DNS controllers don't set up an A record for it.
	ccfg.APIServerDNSRecordSupported = false
	common.Run(ccfg)
})
