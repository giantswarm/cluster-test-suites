package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v5/pkg/clusterbuilder/providers/capmox"
	"github.com/giantswarm/cluster-test-suites/v6/internal/suite"
)

func TestCAPMOXUpgrade(t *testing.T) {
	suite.Setup(true, &capmox.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPMOX Upgrade Suite")
}
