package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capv"
)

const KubeContext = "capv"

func TestCAPVUpgrade(t *testing.T) {
	suite.Setup(KubeContext, &capv.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPV Upgrade Suite")
}
