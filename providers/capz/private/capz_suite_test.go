package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capz"
)

func TestCAPZPrivate(t *testing.T) {
	suite.Setup(false, &capz.PrivateClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Private Suite")
}
