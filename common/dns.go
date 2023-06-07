package common

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/giantswarm/clustertest/pkg/application"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func runDNS() {
	Context("dns", func() {
		var (
			resolver *net.Resolver
			values   *application.DefaultAppsValues
		)

		BeforeEach(func() {
			values = &application.DefaultAppsValues{}
			err := Framework.MC().GetHelmValues(Cluster.Name, Cluster.Namespace, values)
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
				records, err = resolver.LookupIP(context.Background(), "ip", apiDomain)
				return err
			}).Should(Succeed())
			Expect(records).ToNot(BeEmpty())
		})

		It("sets up the bastion DNS records", func() {
			bastionDomain := fmt.Sprintf("bastion1.%s", values.BaseDomain)
			var records []net.IP
			Eventually(func() error {
				var err error
				records, err = resolver.LookupIP(context.Background(), "ip", bastionDomain)
				return err
			}).Should(Succeed())
			Expect(records).ToNot(BeEmpty())
		})

		It("sets up the wildcard DNS records", func() {
			ingressDomain := fmt.Sprintf("ingress.%s.", values.BaseDomain)
			wildcardDomain := fmt.Sprintf("test.%s.", values.BaseDomain)

			fmt.Println(ingressDomain, wildcardDomain)
			dnsMessage := new(dns.Msg)
			dnsMessage.SetQuestion(wildcardDomain, dns.TypeCNAME)
			dnsMessage.RecursionDesired = true

			dnsClient := new(dns.Client)
			var dnsResponse *dns.Msg
			Eventually(func() error {
				var err error
				dnsResponse, _, err = dnsClient.Exchange(dnsMessage, "8.8.8.8:53")
				return err
			}).Should(Succeed())

			Expect(dnsResponse.Answer[0].(*dns.CNAME).Target).To(Equal(ingressDomain))
		})
	})
}
