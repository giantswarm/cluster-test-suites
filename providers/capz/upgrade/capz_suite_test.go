package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capz"
)

const KubeContext = "capz"

func TestCAPZUpgrade(t *testing.T) {
	suite.Setup(true, KubeContext, &capz.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Upgrade Suite")
}
