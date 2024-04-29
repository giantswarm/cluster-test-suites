package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capz"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestCAPZStandard(t *testing.T) {
	suite.Setup(false, &capz.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Standard Suite")
}
