package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/clusterbuilder/providers/capz"

	capzsuite "github.com/giantswarm/cluster-test-suites/v7/internal/capz"
	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

func TestCAPZStandard(t *testing.T) {
	suite.SetupWithOptions(false, &capz.ClusterBuilder{}, []suite.Option{capzsuite.WithLegacyNodePoolReplicas(false)})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Standard Suite")
}
