package upgrade

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v5/pkg/clusterbuilder/providers/capa"
	clustertestclient "github.com/giantswarm/clustertest/v4/pkg/client"
	"github.com/giantswarm/clustertest/v4/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v6/internal/state"
	"github.com/giantswarm/cluster-test-suites/v6/internal/suite"
)

func TestEKSUpgrade(t *testing.T) {
	suite.Setup(true, &capa.ManagedClusterBuilder{}, func(client *clustertestclient.Client) {
		Eventually(
			wait.AreNumNodesReady(state.GetContext(), client, 2, clustertestclient.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"}),
			20*time.Minute, 15*time.Second,
		).Should(BeTrue())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "EKS Upgrade Suite")
}
