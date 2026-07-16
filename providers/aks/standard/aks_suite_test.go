package standard

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/clusterbuilder/providers/capz"
	clustertestclient "github.com/giantswarm/clustertest/v5/pkg/client"
	"github.com/giantswarm/clustertest/v5/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

func TestAKSStandard(t *testing.T) {
	suite.Setup(false, &capz.ManagedClusterBuilder{}, func(client *clustertestclient.Client) {
		// AKS has a managed control plane, so we wait for the worker nodes (the System and
		// User node pools, none of which carry the control-plane label) to become ready.
		Eventually(
			wait.AreNumNodesReady(state.GetContext(), client, 2, clustertestclient.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"}),
			20*time.Minute, 15*time.Second,
		).Should(BeTrue())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "AKS Standard Suite")
}
