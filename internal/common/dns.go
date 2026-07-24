package common

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/giantswarm/clustertest/v5/pkg/application"
	"github.com/giantswarm/clustertest/v5/pkg/logger"
	clustertestnet "github.com/giantswarm/clustertest/v5/pkg/net"
	"github.com/giantswarm/clustertest/v5/pkg/wait"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
)

func runDNS(bastionSuppoted bool) {
	Context("dns", func() {
		var (
			resolver *net.Resolver
			values   *application.ClusterValues
		)
		getARecords := func(domain string) ([]net.IP, error) {
			records, err := resolver.LookupIP(context.Background(), "ip", domain)
			if err != nil {
				logger.Log("domain %s still not available", domain)
				return nil, err
			}

			logger.Log("resolved domain %s to %+v", domain, records)
			return records, nil
		}

		BeforeEach(func() {
			values = &application.ClusterValues{}
			// Reading the cluster Helm values hits the MC API and can transiently
			// fail; retry so a blip doesn't fail the spec.
			Eventually(func() error {
				return state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())

			resolver = clustertestnet.NewResolver()
		})

		// FlakeAttempts: DNS resolution depends on external/split-horizon DNS
		// propagation that is inherently transient, so retry the spec a few
		// times before failing.
		It("sets up the api DNS records", FlakeAttempts(3), func() {
			apiDomain := fmt.Sprintf("api.%s.%s", state.GetCluster().Name, values.BaseDomain)
			var records []net.IP
			Eventually(func() error {
				var err error
				records, err = getARecords(apiDomain)
				return err
			}).
				WithTimeout(5 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
			Expect(records).ToNot(BeEmpty())
		})

		It("sets up the bastion DNS records", FlakeAttempts(3), func() {
			if !bastionSuppoted {
				Skip("Bastion is not supported.")
			}
			bastionDomain := fmt.Sprintf("bastion1.%s.%s", state.GetCluster().Name, values.BaseDomain)
			var records []net.IP
			Eventually(func() error {
				var err error
				records, err = getARecords(bastionDomain)
				return err
			}).
				WithTimeout(5 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
			Expect(records).ToNot(BeEmpty())
		})
	})
}
