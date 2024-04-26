package standard

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder/providers/capa"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/internal/suite"
)

const KubeContext = "eks"

func TestEKSStandard(t *testing.T) {
	suite.Setup(false, KubeContext, &capa.ManagedClusterBuilder{}, func(client *client.Client) {
		Eventually(
			wait.AreNumNodesReady(state.GetContext(), client, 2, &cr.MatchingLabels{"node-role.kubernetes.io/worker": ""}),
			20*time.Minute, 15*time.Second,
		).Should(BeTrue())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "EKS Standard Suite")
}
