package standard

import (
	"context"
	"path"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/organization"
	"github.com/giantswarm/clustertest/pkg/utils"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/common"
)

const (
	KubeContext = "capv"
)

var (
	ctx       context.Context
	framework *clustertest.Framework
	cluster   *application.Cluster
)

func TestCAPVStandard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPV Standard Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	logger.LogWriter = GinkgoWriter

	var err error
	framework, err = clustertest.New(KubeContext)
	Expect(err).NotTo(HaveOccurred())

	cluster = setUpWorkloadCluster()

	common.Framework = framework
	common.Cluster = cluster
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
	cluster = application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderVSphere).
		WithOrg(organization.New("giantswarm")). // Uses the `giantswarm` org (and namespace) as it requires a credentials secret to exist already
		WithAppValuesFile(path.Clean("./test_data/cluster_values.yaml"), path.Clean("./test_data/default-apps_values.yaml")).
		WithUserConfigSecret("vsphere-credentials")

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
