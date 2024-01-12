package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capa"
)

const KubeContext = "capa"

func TestCAPAStandard(t *testing.T) {
	suite.Setup(KubeContext, &capa.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Standard Suite")
}
