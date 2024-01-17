package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capvcd"
)

const KubeContext = "capvcd"

func TestCAPVCDUpgrade(t *testing.T) {
	suite.Setup(true, KubeContext, &capvcd.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPVCD Upgrade Suite")
}
