package clusterwidedeny

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/clusterbuilder/providers/capz"
	clustertestclient "github.com/giantswarm/clustertest/v5/pkg/client"
	"github.com/giantswarm/clustertest/v5/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

func TestAKSClusterwideDeny(t *testing.T) {
	suite.SetupWithOptions(false, &capz.ManagedClusterBuilder{},
		[]suite.Option{
			// Releases without the clusterwide deny skip before a cluster is built.
			suite.WithSkipBeforeStandup(common.SkipWithoutClusterwideDeny),
			// AKS clusters take considerably longer to delete than the other providers.
			suite.WithTeardownTimeout(timeout.AKSTeardown),
			// The cluster is created with the clusterwide deny on.
			suite.WithExtraClusterValues(func() (string, error) {
				return common.ClusterwideDenyClusterValues, nil
			}),
		},
		func(client *clustertestclient.Client) {
			// AKS has a managed control plane, so wait for the worker nodes instead.
			Eventually(
				wait.AreNumNodesReady(state.GetContext(), client, 1, clustertestclient.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"}),
				timeout.AKSNodesReady, 15*time.Second,
			).Should(BeTrue())
		})

	RegisterFailHandler(Fail)
	RunSpecs(t, "AKS Clusterwide Deny Suite")
}
