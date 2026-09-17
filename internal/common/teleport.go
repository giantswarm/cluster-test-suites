package common

import (
	"os"
	"strings"
	"time"

	"github.com/giantswarm/clustertest/v5/pkg/logger"
	"github.com/giantswarm/clustertest/v5/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/v7/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/teleport"

	tc "github.com/gravitational/teleport/api/client"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

func runTeleport(teleportSupported bool) {
	Context("teleport", func() {
		var teleportClient *tc.Client

		BeforeEach(func() {
			helper.SetResponsibleTeam(helper.TeamShield)

			if !teleportSupported {
				Skip("Teleport is not supported.")
			}
			teleportIdentityFile := strings.TrimSpace(os.Getenv("TELEPORT_IDENTITY_FILE"))
			if teleportIdentityFile == "" {
				Skip("TELEPORT_IDENTITY_FILE env var not set, skipping teleport test")
			}
			// Building the Teleport client talks to the Teleport proxy, which can
			// transiently fail; retry so a blip doesn't fail the spec.
			Eventually(func() error {
				var err error
				teleportClient, err = teleport.New(state.GetContext(), teleportIdentityFile)
				return err
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		// FlakeAttempts: Teleport registration depends on the external Teleport
		// control plane observing and registering the cluster, which is
		// inherently eventually-consistent.
		It("cluster is registered", FlakeAttempts(3), func() {
			Eventually(func() (bool, error) {
				clusters, err := teleportClient.GetKubernetesServers(state.GetContext())
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
