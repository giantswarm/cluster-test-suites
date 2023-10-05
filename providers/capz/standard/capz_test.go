package standard

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/clustertest/pkg/application"

	"github.com/giantswarm/cluster-test-suites/internal/common"
)

type ClusterValues struct {
	ControlPlane application.ControlPlane `yaml:"controlPlane"`
	NodePools    []application.NodePool   `yaml:"nodePools"`
}

var _ = Describe("Common tests", func() {
	common.Run(&common.TestConfig{
		BastionSupported: true,
		// Disabled until https://github.com/giantswarm/default-apps-azure/pull/150
		ExternalDnsSupported: false,
	})
})
