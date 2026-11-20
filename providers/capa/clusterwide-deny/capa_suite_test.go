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
	suite.SetupWithOptions(false, &capa.ClusterBuilder{}, []suite.Option{
		// Releases without the clusterwide deny skip before a cluster is built.
		suite.WithSkipBeforeStandup(common.SkipWithoutClusterwideDeny),
		// The cluster is created with the clusterwide deny on.
		suite.WithExtraClusterValues(func() (string, error) {
			return common.ClusterwideDenyClusterValues, nil
		}),
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Clusterwide Deny Suite")
}
