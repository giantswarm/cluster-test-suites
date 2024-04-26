package suite

import (
	"context"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cb "github.com/giantswarm/cluster-standup-teardown/pkg/clusterbuilder"
	"github.com/giantswarm/cluster-standup-teardown/pkg/standup"
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"

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

		cluster := cb.LoadOrBuildCluster(framework, clusterBuilder)

		standupClient := standup.New(framework, isUpgrade, clusterReadyFns...)
		cluster, err = standupClient.Standup(cluster)
		Expect(err).NotTo(HaveOccurred())

		state.SetCluster(cluster)
	})

	AfterSuite(func() {
		if isUpgrade && strings.TrimSpace(os.Getenv("E2E_OVERRIDE_VERSIONS")) == "" {
			return
		}

		Expect(state.GetFramework().DeleteCluster(state.GetContext(), state.GetCluster())).To(Succeed())
	})
}
