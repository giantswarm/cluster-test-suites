package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v2/internal/suite"

	"github.com/giantswarm/cluster-standup-teardown/v4/pkg/clusterbuilder/providers/capa"
)

func TestCAPAStandard(t *testing.T) {
	suite.Setup(false, &capa.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Standard Suite")
}
