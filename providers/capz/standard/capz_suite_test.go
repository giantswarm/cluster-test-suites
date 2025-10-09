package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v2/pkg/clusterbuilder/providers/capz"

	"github.com/giantswarm/cluster-test-suites/v2/internal/suite"
)

func TestCAPZStandard(t *testing.T) {
	suite.Setup(false, &capz.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Standard Suite")
}
