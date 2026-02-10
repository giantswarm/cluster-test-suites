package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/clustertest/v3/pkg/application"

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
)

type ClusterValues struct {
	ControlPlane application.ControlPlane `yaml:"controlPlane"`
	NodePools    []application.NodePool   `yaml:"nodePools"`
}

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	// Disabled until https://github.com/giantswarm/roadmap/issues/2693
	cfg.AutoScalingSupported = false
	// Disabled until wildcard ingress support is added
	cfg.ExternalDnsSupported = false
	common.Run(cfg)
})
