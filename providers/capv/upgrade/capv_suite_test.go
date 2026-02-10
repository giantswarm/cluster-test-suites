package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v4/pkg/clusterbuilder/providers/capv"

	"github.com/giantswarm/cluster-test-suites/v4/internal/suite"
)

func TestCAPVUpgrade(t *testing.T) {
	suite.Setup(true, &capv.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPV Upgrade Suite")
}
