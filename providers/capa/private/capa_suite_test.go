package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capa"
)

const KubeContext = "capa-private-proxy"

func TestCAPAPrivate(t *testing.T) {
	suite.Setup(false, KubeContext, &capa.PrivateClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Private Suite")
}
