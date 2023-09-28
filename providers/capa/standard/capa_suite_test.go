package standard

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/providers/capa"
)

const KubeContext = "capa"

func TestCAPAStandard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Standard Suite")
}

var _ = SynchronizedBeforeSuite(
	func() []byte {
		logger.LogWriter = GinkgoWriter

		state.SetContext(context.Background())

		framework, err := clustertest.New(KubeContext)
		Expect(err).NotTo(HaveOccurred())
		state.SetFramework(framework)

		cluster := setUpWorkloadCluster()
		state.SetCluster(cluster)

		return []byte(fmt.Sprintf("%s/%s", cluster.GetNamespace(), cluster.Name))
	},
	func(clusterDetails []byte) {
		state.SetContext(context.Background())

		framework, err := clustertest.New(KubeContext)
		Expect(err).NotTo(HaveOccurred())
		state.SetFramework(framework)

		clusterParts := strings.Split(string(clusterDetails), "/")
		os.Setenv(clustertest.EnvWorkloadClusterNamespace, clusterParts[0])
		os.Setenv(clustertest.EnvWorkloadClusterName, clusterParts[1])
		cluster, err := state.GetFramework().LoadCluster()
		Expect(err).NotTo(HaveOccurred())
		Expect(cluster).NotTo(BeNil())
		state.SetCluster(cluster)
	},
)

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
	cluster := capa.NewClusterApp("", "", "./test_data/cluster_values.yaml", "./test_data/default-apps_values.yaml")
	logger.Log("Workload cluster name: %s", cluster.Name)
	state.SetCluster(cluster)

	applyCtx, cancelApplyCtx := context.WithTimeout(state.GetContext(), 20*time.Minute)
	defer cancelApplyCtx()

	client, err := state.GetFramework().ApplyCluster(applyCtx, state.GetCluster())
	Expect(err).NotTo(HaveOccurred())

	Eventually(
		wait.AreNumNodesReady(state.GetContext(), client, 1, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
		20*time.Minute, 15*time.Second,
	).Should(BeTrue())

	DeferCleanup(func() {
		Expect(state.GetFramework().DeleteCluster(state.GetContext(), state.GetCluster())).To(Succeed())
	})

	return state.GetCluster()
}
