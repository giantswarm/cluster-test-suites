package suite

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	azureMCEndpoint        = "apiserver.glippy.azuretest.gigantic.io"
	azureWCEndpoint        = "apiserver.t-55xm0p0xgecw27656r.azuretest.gigantic.io"
	azureDeletedWCEndpoint = "apiserver.t-deleted0000000000000.azuretest.gigantic.io"
	azurePublicEndpoint    = "apiserver.public0000000000000000.azuretest.gigantic.io"
	azureLookalikeEndpoint = "apiserver.notazuretest.gigantic.io"
	awsInternalEndpoint    = "internal-garm-apiserver-1416956139.eu-central-1.elb.amazonaws.com"
	publicEndpoint         = "api.example.com"
	publicAzureEndpoint    = "example-94df58c6.westeurope.cloudapp.azure.com"
	goneEndpoint           = "apiserver.gone.example.com"
	splitHorizonEndpoint   = "split.example.com"
)

// dnsAnswers is the fixed resolver table shared by the table test below. The Azure, AWS and
// Azure public entries use addresses those endpoint shapes really serve.
var dnsAnswers = map[string][]string{
	azureMCEndpoint:      {"10.223.0.138"},
	azureWCEndpoint:      {"10.223.0.21"},
	azurePublicEndpoint:  {"20.56.229.213"},
	awsInternalEndpoint:  {"10.128.206.21", "10.128.156.208", "10.128.116.251"},
	publicEndpoint:       {"93.184.216.34"},
	publicAzureEndpoint:  {"20.56.229.213"},
	splitHorizonEndpoint: {"93.184.216.34", "10.5.0.7"},
}

// stubResolver answers from a fixed table and records every host it was asked about. When
// block is set it never answers, which stands in for a hanging resolver.
type stubResolver struct {
	t       *testing.T
	answers map[string][]string
	block   bool
	calls   []string
}

func (s *stubResolver) resolve(ctx context.Context, host string) ([]net.IP, error) {
	s.t.Helper()
	s.calls = append(s.calls, host)

	if _, ok := ctx.Deadline(); !ok {
		s.t.Errorf("resolver called without a deadline for host %q", host)
	}

	if s.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	addrs, ok := s.answers[host]
	if !ok {
		return nil, fmt.Errorf("lookup %s: no such host", host)
	}

	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, net.ParseIP(addr))
	}
	return ips, nil
}

// kubeconfigFor builds a minimal kubeconfig holding one cluster entry per server URL.
func kubeconfigFor(servers ...string) []byte {
	var b strings.Builder
	b.WriteString("apiVersion: v1\nkind: Config\nclusters:\n")
	for i, server := range servers {
		fmt.Fprintf(&b, "- name: cluster-%d\n  cluster:\n    server: %s\n", i, server)
	}
	b.WriteString("contexts: []\nusers: []\n")
	return []byte(b.String())
}

func TestHasPrivateEndpoint(t *testing.T) {
	tests := []struct {
		name string
		// servers are the API server URLs placed in the kubeconfig.
		servers []string
		want    bool
		// wantLookups is the set of hosts the resolver must be asked about. A nil value
		// skips the check, which is needed when an early return makes the set depend on
		// the kubeconfig's map iteration order.
		wantLookups []string
		// wantNoLookup asserts that the resolver was never called at all.
		wantNoLookup bool
	}{
		{
			name:        "azure management cluster name resolving to a private address",
			servers:     []string{"https://" + azureMCEndpoint + ":6443"},
			want:        true,
			wantLookups: []string{azureMCEndpoint},
		},
		{
			name:        "azure workload cluster name resolving to a private address",
			servers:     []string{"https://" + azureWCEndpoint + ":6443"},
			want:        true,
			wantLookups: []string{azureWCEndpoint},
		},
		{
			name:         "aws internal load balancer name is settled without a lookup",
			servers:      []string{"https://" + awsInternalEndpoint + ":443"},
			want:         true,
			wantNoLookup: true,
		},
		{
			name:         "dot local name is settled without a lookup",
			servers:      []string{"https://apiserver.cluster.local:6443"},
			want:         true,
			wantNoLookup: true,
		},
		{
			name:         "literal rfc1918 address",
			servers:      []string{"https://10.201.25.1:6443"},
			want:         true,
			wantNoLookup: true,
		},
		{
			name:         "literal public address",
			servers:      []string{"https://52.1.2.3:443"},
			want:         false,
			wantNoLookup: true,
		},
		{
			name:        "public endpoint resolving to a public address",
			servers:     []string{"https://" + publicEndpoint + ":443"},
			want:        false,
			wantLookups: []string{publicEndpoint},
		},
		{
			name:        "public azure endpoint resolving to a public address",
			servers:     []string{"https://" + publicAzureEndpoint + ":443"},
			want:        false,
			wantLookups: []string{publicAzureEndpoint},
		},
		{
			name:        "a resolved public address beats the private domain list",
			servers:     []string{"https://" + azurePublicEndpoint + ":6443"},
			want:        false,
			wantLookups: []string{azurePublicEndpoint},
		},
		{
			name:        "deleted azure cluster falls back to its private domain",
			servers:     []string{"https://" + azureDeletedWCEndpoint + ":6443"},
			want:        true,
			wantLookups: []string{azureDeletedWCEndpoint},
		},
		{
			name:        "unknown name that does not resolve stays public",
			servers:     []string{"https://" + goneEndpoint + ":443"},
			want:        false,
			wantLookups: []string{goneEndpoint},
		},
		{
			name:        "a domain that merely ends in the same letters does not match",
			servers:     []string{"https://" + azureLookalikeEndpoint + ":6443"},
			want:        false,
			wantLookups: []string{azureLookalikeEndpoint},
		},
		{
			name:        "mix of private and public addresses counts as private",
			servers:     []string{"https://" + splitHorizonEndpoint + ":443"},
			want:        true,
			wantLookups: []string{splitHorizonEndpoint},
		},
		{
			name:        "several clusters that are all public",
			servers:     []string{"https://" + publicEndpoint + ":443", "https://" + goneEndpoint + ":443"},
			want:        false,
			wantLookups: []string{publicEndpoint, goneEndpoint},
		},
		{
			name:    "several clusters where one is private",
			servers: []string{"https://" + publicEndpoint + ":443", "https://" + azureMCEndpoint + ":6443"},
			want:    true,
		},
		{
			name:    "several clusters where one is a literal private address",
			servers: []string{"https://" + publicEndpoint + ":443", "https://192.168.4.10:6443"},
			want:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &stubResolver{t: t, answers: dnsAnswers}

			got := hasPrivateEndpointWithResolver(kubeconfigFor(tc.servers...), resolver.resolve)
			if got != tc.want {
				t.Errorf("hasPrivateEndpointWithResolver() = %v, want %v", got, tc.want)
			}

			if tc.wantNoLookup && len(resolver.calls) > 0 {
				t.Errorf("resolver was called for %v, want no lookup at all", resolver.calls)
			}

			if tc.wantLookups != nil {
				gotCalls := append([]string{}, resolver.calls...)
				sort.Strings(gotCalls)
				wantCalls := append([]string{}, tc.wantLookups...)
				sort.Strings(wantCalls)
				if !equalStrings(gotCalls, wantCalls) {
					t.Errorf("resolver was asked about %v, want %v", gotCalls, wantCalls)
				}
			}
		})
	}
}

// TestHasPrivateEndpointCoversEveryProvider pins the answer for every control plane
// endpoint shape the CI fleet uses. The shapes were read from live management clusters and
// from one `crust-gather` archive per provider, so this is a record, not a guess.
func TestHasPrivateEndpointCoversEveryProvider(t *testing.T) {
	tests := []struct {
		provider string
		server   string
		// addresses is what the stub resolver answers. A nil value means the host never
		// reaches the resolver, because a literal address or a name rule settles it.
		addresses []string
		want      bool
	}{
		{
			provider:  "capa standard, public AWS load balancer",
			server:    "https://t-example0000000000-apiserver-771132154.eu-north-1.elb.amazonaws.com:443",
			addresses: []string{"51.20.217.208"},
			want:      false,
		},
		{
			provider: "capa private, internal AWS load balancer",
			server:   "https://internal-t-example0000000000-apiserver-1361009679.eu-central-1.elb.amazonaws.com:443",
			want:     true,
		},
		{
			provider:  "capz standard, public Azure cloudapp name",
			server:    "https://t-example0000000000-5e420eae.westeurope.cloudapp.azure.com:6443",
			addresses: []string{"20.56.229.213"},
			want:      false,
		},
		{
			provider:  "capz private, public name for a private address",
			server:    "https://apiserver.t-example0000000000.azuretest.gigantic.io:6443",
			addresses: []string{"10.223.0.21"},
			want:      true,
		},
		{
			provider:  "aks standard, public Azure managed control plane name",
			server:    "https://t-example0000000000-v9et2e8m.hcp.westeurope.azmk8s.io:443",
			addresses: []string{"20.86.0.10"},
			want:      false,
		},
		{
			provider:  "eks standard, public EKS name",
			server:    "https://0123456789ABCDEF0123456789ABCDEF.gr7.eu-north-1.eks.amazonaws.com:443",
			addresses: []string{"51.20.217.208"},
			want:      false,
		},
		{
			provider: "capv standard, literal vSphere address",
			server:   "https://10.201.24.233:6443",
			want:     true,
		},
		{
			provider: "capvcd standard, literal Cloud Director address",
			server:   "https://10.205.9.2:6443",
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.provider, func(t *testing.T) {
			answers := map[string][]string{}
			if tc.addresses != nil {
				u, err := url.Parse(tc.server)
				if err != nil {
					t.Fatalf("could not parse server %q: %v", tc.server, err)
				}
				answers[u.Hostname()] = tc.addresses
			}
			resolver := &stubResolver{t: t, answers: answers}

			if got := hasPrivateEndpointWithResolver(kubeconfigFor(tc.server), resolver.resolve); got != tc.want {
				t.Errorf("hasPrivateEndpointWithResolver() = %v, want %v", got, tc.want)
			}
			if tc.addresses == nil && len(resolver.calls) > 0 {
				t.Errorf("resolver was called for %v, want no lookup at all", resolver.calls)
			}
		})
	}
}

func TestHasPrivateEndpointUnparseableKubeconfig(t *testing.T) {
	resolver := &stubResolver{t: t, answers: dnsAnswers}

	if hasPrivateEndpointWithResolver([]byte("this is not a kubeconfig"), resolver.resolve) {
		t.Error("hasPrivateEndpointWithResolver() = true, want false for an unparseable kubeconfig")
	}
	if len(resolver.calls) > 0 {
		t.Errorf("resolver was called for %v, want no lookup at all", resolver.calls)
	}
}

func TestHasPrivateEndpointResolverNeverAnswers(t *testing.T) {
	resolver := &stubResolver{t: t, answers: dnsAnswers, block: true}

	start := time.Now()
	got := hasPrivateEndpointWithResolver(kubeconfigFor("https://"+publicEndpoint+":443"), resolver.resolve)
	elapsed := time.Since(start)

	if got {
		t.Error("hasPrivateEndpointWithResolver() = true, want false when the resolver never answers")
	}
	if elapsed > 4*endpointLookupTimeout {
		t.Errorf("lookup took %s, want it bounded by %s", elapsed, endpointLookupTimeout)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
