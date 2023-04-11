package standard

import (
	"context"
	"fmt"
	"os"
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

const (
	KubeContext                 = "capa"
	EnvWorkloadClusterName      = "E2E_WC_NAME"
	EnvWorkloadClusterNamespace = "E2E_WC_NAMESPACE"
)

var (
	ctx       context.Context
	framework *clustertest.Framework
	cluster   *application.Cluster
)

func TestCAPAStandard(t *testing.T) {
	ctx := context.Background()

	workloadClusterName := os.Getenv(EnvWorkloadClusterName)
	workloadClusterNamespace := os.Getenv(EnvWorkloadClusterNamespace)

	var err error
	framework, err = clustertest.New(KubeContext)
	if err != nil {
		Fail(fmt.Sprintf("Failed to initialize clustertest framework: %v", err))
	}

	if workloadClusterName != "" && workloadClusterNamespace != "" {
		cluster, err = framework.LoadCluster(workloadClusterName, workloadClusterNamespace)
		if err != nil {
			Fail(fmt.Sprintf("Failed to initialize clustertest framework: %v", err))
		}
	} else {
		cluster = application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderAWS).
			WithAppValuesFile(path.Clean("./test_data/cluster_values.yaml"), path.Clean("./test_data/default-apps_values.yaml"))
		logger.Log("Workload cluster name: %s", cluster.Name)
	}

	BeforeSuite(func() {
		logger.LogWriter = GinkgoWriter

		if workloadClusterName != "" && workloadClusterNamespace != "" {
			return
		}

		applyCtx, cancelApplyCtx := context.WithTimeout(ctx, 20*time.Minute)
		defer cancelApplyCtx()

		client, err := framework.ApplyCluster(applyCtx, cluster)
		Expect(err).NotTo(HaveOccurred())

		Eventually(
			wait.IsNumNodesReady(ctx, client, 1, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
			20*time.Minute, 15*time.Second,
		).Should(BeTrue())
	})

	AfterSuite(func() {
		if workloadClusterName != "" && workloadClusterNamespace != "" {
			return
		}
		Expect(framework.DeleteCluster(ctx, cluster)).To(Succeed())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Standard Suite")
}
