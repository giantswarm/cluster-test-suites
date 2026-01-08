package china

import (
	"testing"

	"github.com/giantswarm/cluster-test-suites/v3/internal/suite"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v4/pkg/clusterbuilder/providers/capa"
)

func TestCAPAChina(t *testing.T) {
	suite.Setup(false, &capa.ChinaBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA China Suite")
}
