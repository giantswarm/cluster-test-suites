package common

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func runDNS(bastionSuppoted bool) {
	Context("dns", func() {
		var (
			resolver *net.Resolver
			values   *application.DefaultAppsValues
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
			values = &application.DefaultAppsValues{}
			defaultAppsName := fmt.Sprintf("%s-default-apps", state.Get().GetCluster().Name)
			err := state.Get().GetFramework().MC().GetHelmValues(defaultAppsName, state.Get().GetCluster().Namespace, values)
			Expect(err).NotTo(HaveOccurred())

			resolver = &net.Resolver{
				PreferGo:     true,
				StrictErrors: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: time.Millisecond * time.Duration(10000),
					}
					return d.DialContext(ctx, "udp", "8.8.4.4:53")
				},
			}
		})

		It("sets up the api DNS records", func() {
			apiDomain := fmt.Sprintf("api.%s", values.BaseDomain)
			var records []net.IP
			Eventually(func() error {
				var err error
				records, err = getARecords(apiDomain)
				return err
			}).WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
			Expect(records).ToNot(BeEmpty())
		})

		It("sets up the bastion DNS records", func() {
			if !bastionSuppoted {
				Skip("Bastion is not supported.")
			}
			bastionDomain := fmt.Sprintf("bastion1.%s", values.BaseDomain)
			var records []net.IP
			Eventually(func() error {
				var err error
				records, err = getARecords(bastionDomain)
				return err
			}).WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
			Expect(records).ToNot(BeEmpty())
		})
	})
}
