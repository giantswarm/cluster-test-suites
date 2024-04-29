package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capvcd"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestCAPVCDStandard(t *testing.T) {
	suite.Setup(false, &capvcd.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPVCD Standard Suite")
}
