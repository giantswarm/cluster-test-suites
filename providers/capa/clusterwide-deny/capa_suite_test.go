package clusterwidedeny

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/clusterbuilder/providers/capa"

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

func TestCAPAClusterwideDeny(t *testing.T) {
	// Releases without the clusterwide deny skip before a cluster is built.
	suite.SetupWithOptions(false, &capa.ClusterBuilder{}, []suite.Option{
		suite.WithSkipBeforeStandup(common.SkipWithoutClusterwideDeny),
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Clusterwide Deny Suite")
}
