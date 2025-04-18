package cilium_eni_mode

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/internal/suite"

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capa"
)

func TestCAPACiliumEniMode(t *testing.T) {
	suite.Setup(false, &capa.ClusterBuilder{})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA 'Cilium ENI mode' Suite")
}
