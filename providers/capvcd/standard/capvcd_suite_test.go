package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v2/pkg/clusterbuilder/providers/capvcd"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestCAPVCDStandard(t *testing.T) {
	suite.Setup(false, &capvcd.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPVCD Standard Suite")
}
