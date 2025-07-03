package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capv"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestCAPVStandard(t *testing.T) {
	suite.Setup(false, "vsphere", &capv.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPV Standard Suite")
}
