package upgrade

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/common"
	"github.com/giantswarm/cluster-test-suites/internal/upgrade"
	"github.com/giantswarm/cluster-test-suites/providers/capvcd"
)

const KubeContext = "capvcd"

var (
	ctx       context.Context
	framework *clustertest.Framework
	cluster   *application.Cluster
)

func TestCAPVCDUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPVCD Upgrade Suite")
}

var _ = BeforeSuite(func() {
	if os.Getenv("E2E_OVERRIDE_VERSIONS") == "" {
		Skip("E2E_OVERRIDE_VERSIONS env var not set, skipping upgrade test")
		return
	}

	ctx = context.Background()
	logger.LogWriter = GinkgoWriter

	var err error
	framework, err = clustertest.New(KubeContext)
	Expect(err).NotTo(HaveOccurred())

	cluster = setUpWorkloadCluster()

	common.Framework = framework
	common.Cluster = cluster
	upgrade.Framework = framework
	upgrade.Cluster = cluster
})

func setUpWorkloadCluster() *application.Cluster {
	cluster, err := framework.LoadCluster()
	Expect(err).NotTo(HaveOccurred())
	if cluster != nil {
		logger.Log("Using existing cluster %s/%s", cluster.Name, cluster.Namespace)
		return cluster
	}

	return createCluster()
}

func createCluster() *application.Cluster {
	cluster = capvcd.NewClusterApp("", "", "./test_data/cluster_values.yaml", "./test_data/default-apps_values.yaml").
		WithAppVersions("latest", "latest")
	logger.Log("Workload cluster name: %s", cluster.Name)

	applyCtx, cancelApplyCtx := context.WithTimeout(ctx, 20*time.Minute)
	defer cancelApplyCtx()

	client, err := framework.ApplyCluster(applyCtx, cluster)
	Expect(err).NotTo(HaveOccurred())

	Eventually(
		wait.AreNumNodesReady(ctx, client, 1, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
		20*time.Minute, 15*time.Second,
	).Should(BeTrue())

	DeferCleanup(func() {
		Expect(framework.DeleteCluster(ctx, cluster)).To(Succeed())
	})

	return cluster
}
