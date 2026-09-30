package upgrade

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
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

func TestAKSUpgrade(t *testing.T) {
	suite.SetupWithOptions(true, &capz.ManagedClusterBuilder{},
		// AKS clusters take considerably longer to delete than the other providers, so we
		// give the teardown more headroom than the shared default.
		[]suite.Option{suite.WithTeardownTimeout(timeout.AKSTeardown)},
		func(client *clustertestclient.Client) {
			// AKS has a managed control plane, so we wait for the worker nodes (the System node
			// pool, which does not carry the control-plane label) to become ready. Provisioning
			// the managed control plane and its node pool is slow, hence the generous timeout.
			Eventually(
				wait.AreNumNodesReady(state.GetContext(), client, 1, clustertestclient.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"}),
				timeout.AKSNodesReady, 15*time.Second,
			).Should(BeTrue())
		})

	RegisterFailHandler(Fail)
	RunSpecs(t, "AKS Upgrade Suite")
}
