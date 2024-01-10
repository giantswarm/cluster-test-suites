package upgrade

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/providers/eks"
)

const KubeContext = "eks"

func TestEKSUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EKS Upgrade Suite")
}

var _ = BeforeSuite(func() {
	if strings.TrimSpace(os.Getenv("E2E_OVERRIDE_VERSIONS")) == "" {
		Skip("E2E_OVERRIDE_VERSIONS env var not set, skipping upgrade test")
		return
	}

	logger.LogWriter = GinkgoWriter

	state.SetContext(context.Background())

	framework, err := clustertest.New(KubeContext)
	Expect(err).NotTo(HaveOccurred())
	state.SetFramework(framework)

	cluster := setUpWorkloadCluster()
	state.SetCluster(cluster)
})

func setUpWorkloadCluster() *application.Cluster {
	cluster, err := state.GetFramework().LoadCluster()
	Expect(err).NotTo(HaveOccurred())
	if cluster != nil {
		logger.Log("Using existing cluster %s/%s", cluster.Name, cluster.GetNamespace())
		return cluster
	}

	return createCluster()
}

func createCluster() *application.Cluster {
	cluster := eks.NewClusterApp("", "", "./test_data/cluster_values.yaml", "./test_data/default-apps_values.yaml").
		WithAppVersions("0.10.0-bf5df5293128aab88e5c59d2440b1049935eb897", "latest")
	logger.Log("Workload cluster name: %s", cluster.Name)
	state.SetCluster(cluster)

	applyCtx, cancelApplyCtx := context.WithTimeout(state.GetContext(), 20*time.Minute)
	defer cancelApplyCtx()

	client, err := state.GetFramework().ApplyCluster(applyCtx, state.GetCluster())
	Expect(err).NotTo(HaveOccurred())

	Eventually(
		wait.AreNumNodesReady(state.GetContext(), client, 3, &cr.MatchingLabels{"node-role.kubernetes.io/worker": ""}),
		20*time.Minute, 15*time.Second,
	).Should(BeTrue())

	DeferCleanup(func() {
		Expect(state.GetFramework().DeleteCluster(state.GetContext(), state.GetCluster())).To(Succeed())
	})

	return state.GetCluster()
}
