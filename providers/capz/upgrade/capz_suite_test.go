package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capz"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestCAPZUpgrade(t *testing.T) {
	suite.Setup(true, &capz.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Upgrade Suite")
}
