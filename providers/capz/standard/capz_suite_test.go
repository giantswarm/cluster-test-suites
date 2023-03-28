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
	"github.com/giantswarm/clustertest/pkg/utils"
	"github.com/giantswarm/clustertest/pkg/wait"
)

const KubeContext = "capz"

var (
	framework *clustertest.Framework
	cluster   *application.Cluster
)

func TestCAPZStandard(t *testing.T) {
	var err error
	ctx := context.Background()

	logger.LogWriter = GinkgoWriter

	framework, err = clustertest.New(KubeContext)
	if err != nil {
		panic(err)
	}

	cluster = application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderAzure).
		WithAppValuesFile(path.Clean("./test_data/cluster_values.yaml"), path.Clean("./test_data/default-apps_values.yaml"))

	BeforeSuite(func() {
		logger.Log("Workload cluster name: %s", cluster.Name)

		applyCtx, cancelApplyCtx := context.WithTimeout(ctx, 20*time.Minute)
		defer cancelApplyCtx()

		client, err := framework.ApplyCluster(applyCtx, cluster)
		Expect(err).ToNot(HaveOccurred())

		Eventually(
			wait.IsNumNodesReady(ctx, client, 1, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
			20*time.Minute, 15*time.Second,
		).Should(BeTrue())
	})

	AfterSuite(func() {
		Expect(framework.DeleteCluster(ctx, cluster)).To(Succeed())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Standard Suite")
}
