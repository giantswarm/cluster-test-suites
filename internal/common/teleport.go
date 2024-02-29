package common

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/cluster-test-suites/internal/teleport"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	tc "github.com/gravitational/teleport/api/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func runTeleport(teleportSupported bool) {
	Context("teleport", func() {
		var teleportClient *tc.Client

		BeforeEach(func() {
			if !teleportSupported {
				Skip("Teleport is not supported.")
			}
			teleportIdentityFile := strings.TrimSpace(os.Getenv("TELEPORT_IDENTITY_FILE"))
			if teleportIdentityFile == "" {
				Skip("TELEPORT_IDENTITY_FILE env var not set, skipping teleport test")
			}
			var err error
			teleportClient, err = teleport.New(context.Background(), teleportIdentityFile)
			Expect(err).To(BeNil())
		})

		It("cluster is registered", func() {
			Eventually(func() (bool, error) {
				clusters, err := teleportClient.GetKubernetesServers(context.Background())
				if err != nil {
					return false, err
				}
				for _, cluster := range clusters {
					if strings.Contains(cluster.GetName(), state.GetCluster().Name) {
						logger.Log("cluster registered %v", cluster)
						return true, nil
					}
				}
				logger.Log("cluster %s still not registered", state.GetCluster().Name)
				return false, nil
			}).
				WithTimeout(5 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(BeTrue())
		})
	})
}
