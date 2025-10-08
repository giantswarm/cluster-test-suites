package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/internal/suite"

	"github.com/giantswarm/cluster-standup-teardown/v2/pkg/clusterbuilder/providers/capa"
)

func TestCAPAPrivate(t *testing.T) {
	suite.Setup(false, &capa.PrivateClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Private Suite")
}
