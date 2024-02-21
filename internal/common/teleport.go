package common

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/giantswarm/cluster-test-suites/internal/helper"
	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	tc "github.com/gravitational/teleport/api/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func runTeleport(teleportSupported bool) {
	Context("teleport", func() {
		var teleportClient *tc.Client

		isClusterRegistered := func(clusterName string) (bool, error) {
			clusters, err := teleportClient.GetKubernetesServers(context.Background())
			if err != nil {
				return false, err
			}
			for _, cluster := range clusters {
				if strings.Contains(cluster.GetName(), clusterName) {
					logger.Log("cluster registered %v", cluster)
					return true, nil
				}
			}
			logger.Log("cluster %s still not registered", clusterName)
			return false, nil
		}

		BeforeEach(func() {
			if strings.TrimSpace(os.Getenv("TELEPORT_IDENTITY_FILE")) == "" {
				Skip("TELEPORT_IDENTITY_FILE env var not set, skipping teleport test")
				return
			}
			var err error
			teleportClient, err = helper.NewTeleportClient(context.Background(), os.Getenv("TELEPORT_IDENTITY_FILE"))
			Expect(err).To(BeNil())
		})

		It("cluster is registered", func() {
			if !teleportSupported {
				Skip("Teleport is not supported.")
			}

			Eventually(func() (bool, error) {
				ok, err := isClusterRegistered(state.GetCluster().Name)
				return ok, err
			}).
				WithTimeout(5 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(BeTrue())
		})
	})
}
