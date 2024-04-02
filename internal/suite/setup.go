package suite

import (
	"context"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

// ClusterBuilder is an interface that provides a function for building provider-specific Cluster apps
type ClusterBuilder interface {
	NewClusterApp(clusterName string, orgName string, clusterValuesFile string, defaultAppsValuesFile string) *application.Cluster
}

// Setup handles the creation of the BeforeSuite and AfterSuite handlers. This covers the creations and cleanup of the test cluster.
// `clusterReadyFns` can be provided if the cluster requires custom checks for cluster-ready status. If not provided the cluster will
// be checked for at least a single control plane node being marked as ready.
func Setup(isUpgrade bool, kubeContext string, clusterBuilder ClusterBuilder, clusterReadyFns ...func(client *client.Client)) {
	BeforeSuite(func() {
		if isUpgrade && strings.TrimSpace(os.Getenv("E2E_OVERRIDE_VERSIONS")) == "" {
			Skip("E2E_OVERRIDE_VERSIONS env var not set, skipping upgrade test")
			return
		}

		logger.LogWriter = GinkgoWriter

		state.SetContext(context.Background())

		framework, err := clustertest.New(kubeContext)
		Expect(err).NotTo(HaveOccurred())
		state.SetFramework(framework)

		// In certain cases, when connecting over the VPN, it is possible that the tunnel
		// isn't ready and can take a short while to become usable. This attempts to wait
		// for the connection to be usable before starting the tests.
		Eventually(func() error {
			logger.Log("Checking connection to MC is available. API Endpoint: %s", framework.MC().GetAPIServerEndpoint())
			return framework.MC().CheckConnection()
		}).
			WithTimeout(5 * time.Minute).
			WithPolling(5 * time.Second).
			Should(Succeed())

		cluster := setUpWorkloadCluster(isUpgrade, clusterBuilder, clusterReadyFns...)
		state.SetCluster(cluster)
	})

	AfterSuite(func() {
		if isUpgrade && strings.TrimSpace(os.Getenv("E2E_OVERRIDE_VERSIONS")) == "" {
			return
		}

		Expect(state.GetFramework().DeleteCluster(state.GetContext(), state.GetCluster())).To(Succeed())
	})
}

func setUpWorkloadCluster(isUpgrade bool, clusterBuilder ClusterBuilder, clusterReadyFns ...func(client *client.Client)) *application.Cluster {
	cluster, err := state.GetFramework().LoadCluster()
	Expect(err).NotTo(HaveOccurred())
	if cluster != nil {
		logger.Log("Using existing cluster %s/%s", cluster.Name, cluster.GetNamespace())
		return cluster
	}

	return createCluster(isUpgrade, clusterBuilder, clusterReadyFns...)
}

func createCluster(isUpgrade bool, clusterBuilder ClusterBuilder, clusterReadyFns ...func(client *client.Client)) *application.Cluster {
	cluster := clusterBuilder.NewClusterApp("", "", "./test_data/cluster_values.yaml", "./test_data/default-apps_values.yaml")
	if isUpgrade {
		cluster = cluster.WithAppVersions("latest", "latest")
	}
	logger.Log("Workload cluster name: %s", cluster.Name)
	state.SetCluster(cluster)

	applyCtx, cancelApplyCtx := context.WithTimeout(state.GetContext(), 20*time.Minute)
	defer cancelApplyCtx()

	client, err := state.GetFramework().ApplyCluster(applyCtx, state.GetCluster())
	Expect(err).NotTo(HaveOccurred())

	if len(clusterReadyFns) > 0 {
		// Use provided functions to check if cluster is ready.
		// This is mainly used for managed clusters such as EKS that need to check for worker nodes rather than control plane nodes.
		for _, fn := range clusterReadyFns {
			fn(client)
		}
	} else {
		// If no custom check functions are provided we default to checking for a single control plane node being ready
		Eventually(
			wait.AreNumNodesReady(state.GetContext(), client, 1, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
			20*time.Minute, 15*time.Second,
		).Should(BeTrue())
	}

	return state.GetCluster()
}
