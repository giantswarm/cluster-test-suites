package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capv"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestCAPVStandard(t *testing.T) {
	suite.Setup(false, &capv.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPV Standard Suite")
}
