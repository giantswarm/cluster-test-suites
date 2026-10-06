package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	capzsuite "github.com/giantswarm/cluster-test-suites/v7/internal/capz"
	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"

	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/clusterbuilder/providers/capz"
)

func TestCAPZPrivate(t *testing.T) {
	suite.SetupWithOptions(false, &capz.PrivateClusterBuilder{}, []suite.Option{capzsuite.WithLegacyNodePoolReplicas(false)})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Private Suite")
}
