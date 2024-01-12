package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capa"
)

const KubeContext = "capa"

func TestCAPAUpgrade(t *testing.T) {
	suite.Setup(true, KubeContext, &capa.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Upgrade Suite")
}
