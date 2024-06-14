package china

import (
	"github.com/giantswarm/cluster-test-suites/internal/suite"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capa"
)

func TestCAPAChina(t *testing.T) {
	suite.Setup(false, &capa.ChinaBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA China Suite")
}
