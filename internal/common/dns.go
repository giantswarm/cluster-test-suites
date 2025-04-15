package common

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	clustertestnet "github.com/giantswarm/clustertest/pkg/net"
	"github.com/giantswarm/clustertest/pkg/wait"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/internal/state"
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
			err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
			Expect(err).NotTo(HaveOccurred())

			resolver = clustertestnet.NewResolver()
		})

		It("sets up the api DNS records", func() {
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

		It("sets up the bastion DNS records", func() {
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
