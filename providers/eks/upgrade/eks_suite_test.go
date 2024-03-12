package upgrade

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/eks"
)

const KubeContext = "eks"

func TestEKSUpgrade(t *testing.T) {
	suite.Setup(true, KubeContext, &eks.ClusterBuilder{}, func(client *client.Client) {
		Eventually(
			wait.AreNumNodesReady(state.GetContext(), client, 2, &cr.MatchingLabels{"node-role.kubernetes.io/worker": ""}),
			40*time.Minute, 30*time.Second,
		).Should(BeTrue())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "EKS Upgrade Suite")
}
