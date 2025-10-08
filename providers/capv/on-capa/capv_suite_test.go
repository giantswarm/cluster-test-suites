package standard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v2/pkg/clusterbuilder/providers/capv"

	"github.com/giantswarm/cluster-test-suites/v2/internal/suite"
)

func TestCAPVOnCAPA(t *testing.T) {
	suite.Setup(false, &capv.ClusterBuilder{CustomKubeContext: "capv-on-capa"})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPV on CAPA Suite")
}
