package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capvcd"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestCAPVCDUpgrade(t *testing.T) {
	suite.Setup(true, &capvcd.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPVCD Upgrade Suite")
}
