package clusterwidedeny

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

var _ = Describe("Clusterwide deny", func() {
	BeforeEach(func() {
		// AKS clusters take longer to fully settle after being created.
		state.SetTestTimeout(timeout.ClusterReadyTimeout, timeout.AKSClusterReady)
	})

	// Same settings as the AKS standard suite.
	cfg := common.NewTestConfigWithDefaults()
	cfg.ControlPlaneMetricsSupported = false
	cfg.AutoScalingSupported = false
	cfg.GatewayAPISupported = false
	cfg.APIServerDNSRecordSupported = false
	cfg.DNSServiceName = "kube-dns"
	cfg.IgnoredRestartedPodsLabelFilters = common.LabelFilter{
		"azure-workload-identity.io/system": []string{"true"},
	}
	common.RunWithClusterwideDeny(cfg)
})
