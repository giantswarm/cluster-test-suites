package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/clusterbuilder/providers/capmox"
	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

func TestCAPMOXStandard(t *testing.T) {
	suite.Setup(false, &capmox.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPMOX Standard Suite")
}
