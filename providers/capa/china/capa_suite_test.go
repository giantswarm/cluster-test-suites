package china

import (
	"testing"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capa"
)

func TestCAPAChina(t *testing.T) {
	suite.Setup(false, "capa", &capa.ChinaBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA China Suite")
}
