package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/clustertest/pkg/application"

	"github.com/giantswarm/cluster-test-suites/internal/common"
)

type ClusterValues struct {
	ControlPlane application.ControlPlane `yaml:"controlPlane"`
	NodePools    []application.NodePool   `yaml:"nodePools"`
}

var _ = Describe("Common tests", func() {
	common.Run(&common.TestConfig{
		// Disabled until https://github.com/giantswarm/roadmap/issues/2693
		AutoScalingSupported: false,
		BastionSupported:     false,
		TeleportSupported:    true,
		// Disabled until wildcard ingress support is added
		ExternalDnsSupported:         false,
		ControlPlaneMetricsSupported: true,
	})
})
