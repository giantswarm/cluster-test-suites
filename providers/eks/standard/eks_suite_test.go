package standard

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capa"
	clustertestclient "github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

func TestEKSStandard(t *testing.T) {
	suite.Setup(false, &capa.ManagedClusterBuilder{}, func(client *clustertestclient.Client) {
		Eventually(
			wait.AreNumNodesReady(state.GetContext(), client, 2, clustertestclient.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"}),
			20*time.Minute, 15*time.Second,
		).Should(BeTrue())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "EKS Standard Suite")
}
