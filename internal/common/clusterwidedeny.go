package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/clustertest/v5/pkg/application"
	"github.com/giantswarm/clustertest/v5/pkg/client"
	"github.com/giantswarm/clustertest/v5/pkg/env"
	"github.com/giantswarm/clustertest/v5/pkg/logger"

	"github.com/giantswarm/cluster-test-suites/v7/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

const (
	// networkPoliciesAppName is the network-policies app in the Release and the
	// suffix of its HelmRelease.
	networkPoliciesAppName = "network-policies"

	// clusterwideDenyPolicyName is the CiliumClusterwideNetworkPolicy the
	// network-policies app renders when denyByDefault.clusterWide.enabled is true.
	clusterwideDenyPolicyName = "default-deny-all-namespaces"
	// clusterwideAllowDNSPolicyName is rendered as well when
	// denyByDefault.clusterWide.allowDNS.enabled is true.
	clusterwideAllowDNSPolicyName = "allow-dns-all-namespaces"

	// ClusterwideDenyClusterValues creates the cluster with the clusterwide deny
	// and the DNS allow on, so every default app installs and starts under the
	// deny. Pass it with suite.WithExtraClusterValues.
	ClusterwideDenyClusterValues = `global:
  apps:
    networkPolicies:
      values:
        denyByDefault:
          clusterWide:
            enabled: true
            allowDNS:
              enabled: true
`
)

// clusterwideDenyMinVersion is the first network-policies release with the
// clusterwide deny.
var clusterwideDenyMinVersion = semver.MustParse("0.6.0")

// errNoNetworkPolicies marks a cluster without a network-policies HelmRelease.
var errNoNetworkPolicies = errors.New("the cluster has no network-policies HelmRelease")

// RunWithClusterwideDeny runs the common checks that don't need customer
// workloads on a cluster created with the clusterwide deny of the
// network-policies app on (ClusterwideDenyClusterValues): default apps, basic
// workload health, cluster values, cert-manager, DNS, metrics, Teleport and
// storage. A release whose default apps lose traffic they need under the deny
// fails here, before a customer turns it on.
//
// Left out: hello world and scale, which deploy customer-like workloads that
// would need their own allow labels, the compliance report, and
// provider-specific checks such as ECR.
func RunWithClusterwideDeny(cfg *TestConfig) {
	// ContinueOnFailure: one gap (an app losing traffic it needs) must not hide
	// the others, so every check runs even after one failed.
	Describe("with the clusterwide deny on", Ordered, ContinueOnFailure, func() {
		var anySpecFailed, denyOn bool

		BeforeAll(func() {
			helper.SetResponsibleTeam(helper.TeamCabbage)
			ctx := state.GetContext()

			// Backstop for clusters the suite didn't build itself (E2E_WC_NAME);
			// SkipWithoutClusterwideDeny skips old releases before standup. The
			// HelmRelease only reports its chart version once Flux reconciled it.
			var version *semver.Version
			var lastErr error
			Eventually(func() error {
				version, lastErr = networkPoliciesVersion(ctx)
				if errors.Is(lastErr, errNoNetworkPolicies) {
					allowAll, err := allowAllEgress(ctx)
					if err != nil {
						return err
					}
					if allowAll {
						return nil
					}
				}
				return lastErr
			}).
				WithTimeout(state.GetTestTimeout(timeout.DeployApps, 15*time.Minute)).
				WithPolling(15*time.Second).
				Should(Succeed(), "could not read the network-policies chart version")
			if errors.Is(lastErr, errNoNetworkPolicies) {
				Skip("allowAllEgress turns the network-policies app off on this cluster")
			}
			if reason := clusterwideDenySkipReason(version); reason != "" {
				Skip(reason)
			}

			// Proves the cluster really runs under the deny, so the checks below
			// don't pass on a cluster without it.
			expectClusterwideDenyOn(ctx)
			denyOn = true
		})

		ReportAfterEach(func(report SpecReport) {
			if report.Failed() {
				anySpecFailed = true
			}
		})

		AfterAll(func() {
			// Nothing to undo after a skip or when the deny never came up. Writing the
			// switch would also break a cluster whose network-policies has no
			// clusterWide value yet.
			if !denyOn {
				return
			}
			ctx := state.GetContext()
			// AfterAll runs before the last spec's ReportAfterEach, hence both.
			if anySpecFailed || CurrentSpecReport().Failed() {
				logClusterwideDenyState(ctx)
			}
			// Turn the deny off before AfterSuite: PV cleanup and cluster deletion need
			// the storage, load balancer and cloud controllers in the WC to reach the
			// cloud API, and uninstall hooks need the API server. The crust-gather
			// snapshot AfterSuite takes on failure shows the cluster with the deny off;
			// the log above has the state with it on.
			// A cluster kept for debugging (E2E_WC_KEEP) keeps the deny, like
			// DeleteCluster keeps the cluster.
			if keep := strings.ToLower(os.Getenv(env.KeepWorkloadCluster)); keep != "" && keep != "false" {
				logger.Log("%s is set, leaving the clusterwide deny on", env.KeepWorkloadCluster)
				return
			}
			turnClusterwideDenyOff(ctx)
		})

		RunApps(cfg)
		runBasic(cfg)
		runClusterValues(cfg)
		runCertManager(cfg.CertManagerSupported)
		runDNS(cfg)
		runMetrics(cfg)
		runTeleport(cfg.TeleportSupported)
		runStorage()
	})
}

// SkipWithoutClusterwideDeny is a suite.WithSkipBeforeStandup check: it skips
// the suite before a cluster is built when the release under test ships a
// network-policies version without the clusterwide deny.
func SkipWithoutClusterwideDeny(cluster *application.Cluster) (string, error) {
	// GetRelease resolves "latest" and catalog overrides on the cluster app in
	// place; restore them so standup does its own resolution.
	version, catalog := cluster.ClusterApp.Version, cluster.ClusterApp.Catalog
	release, err := cluster.GetRelease()
	cluster.ClusterApp.Version, cluster.ClusterApp.Catalog = version, catalog
	if err != nil {
		return "", fmt.Errorf("failed to get the release under test: %w", err)
	}
	for _, app := range release.Spec.Apps {
		if strings.EqualFold(app.Name, networkPoliciesAppName) {
			v, err := semver.NewVersion(app.Version)
			if err != nil {
				return "", fmt.Errorf("unusable %s version %q: %w", networkPoliciesAppName, app.Version, err)
			}
			return clusterwideDenySkipReason(v), nil
		}
	}
	return fmt.Sprintf("the release under test has no %s app", networkPoliciesAppName), nil
}

// clusterwideDenySkipReason compares the core version only, like the rest of the
// repo, so release candidates and dev builds of 0.6.0 run the suite.
func clusterwideDenySkipReason(version *semver.Version) string {
	core := semver.New(version.Major(), version.Minor(), version.Patch(), "", "")
	if core.LessThan(clusterwideDenyMinVersion) {
		return fmt.Sprintf("%s %s has no clusterwide deny (needs %s or newer)", networkPoliciesAppName, version, clusterwideDenyMinVersion)
	}
	return ""
}

// networkPoliciesVersion returns the chart version of the cluster's
// network-policies HelmRelease. The cluster chart references the chart through
// an OCIRepository (spec.chartRef), so the version is only in the status.
func networkPoliciesVersion(ctx context.Context) (*semver.Version, error) {
	cluster := state.GetCluster()
	key := types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", cluster.Name, networkPoliciesAppName),
		Namespace: cluster.Organization.GetNamespace(),
	}
	release := &helmv2.HelmRelease{}
	if err := state.GetFramework().MC().Get(ctx, key, release); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errNoNetworkPolicies
		}
		return nil, fmt.Errorf("failed to get HelmRelease %s: %w", key, err)
	}
	raw := release.Status.LastAttemptedRevision
	if raw == "" && len(release.Status.History) > 0 {
		raw = release.Status.History[0].ChartVersion
	}
	version, err := semver.NewVersion(raw)
	if err != nil {
		return nil, fmt.Errorf("HelmRelease %s has no usable version %q yet: %w", key, raw, err)
	}
	return version, nil
}

// clusterValues returns the cluster app's user config values and their
// ConfigMap.
func clusterValues(ctx context.Context) (map[string]any, *corev1.ConfigMap, error) {
	cluster := state.GetCluster()
	key := types.NamespacedName{
		Name:      fmt.Sprintf("%s-userconfig", cluster.ClusterApp.InstallName),
		Namespace: cluster.ClusterApp.GetNamespace(),
	}
	cm := &corev1.ConfigMap{}
	if err := state.GetFramework().MC().Get(ctx, key, cm); err != nil {
		return nil, nil, err
	}
	values := map[string]any{}
	if err := yaml.Unmarshal([]byte(cm.Data["values"]), &values); err != nil {
		return nil, nil, fmt.Errorf("failed to parse the cluster values in %s: %w", key, err)
	}
	return values, cm, nil
}

// allowAllEgress reports whether the cluster values turn the network-policies
// app off with global.connectivity.network.allowAllEgress.
func allowAllEgress(ctx context.Context) (bool, error) {
	values, _, err := clusterValues(ctx)
	if err != nil {
		return false, err
	}
	allowAll, _, err := unstructured.NestedBool(values, "global", "connectivity", "network", "allowAllEgress")
	return allowAll, err
}

// turnClusterwideDenyOff sets denyByDefault.clusterWide.enabled to false in the
// cluster app's user config and waits for the deny to go away. Best effort and
// bounded: it only logs, since the cluster is deleted either way.
func turnClusterwideDenyOff(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	logger.Log("Turning the clusterwide deny off before teardown")
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		values, cm, err := clusterValues(ctx)
		if err != nil {
			return err
		}
		if err := unstructured.SetNestedField(values, false, "global", "apps", "networkPolicies", "values", "denyByDefault", "clusterWide", "enabled"); err != nil {
			return err
		}
		out, err := yaml.Marshal(values)
		if err != nil {
			return err
		}
		cm.Data["values"] = string(out)
		return state.GetFramework().MC().Update(ctx, cm)
	})
	if err != nil {
		logger.Log("Could not turn the clusterwide deny off before teardown: %s", err)
		return
	}
	g := NewGomega(func(message string, _ ...int) { logger.Log("%s", message) })
	gone := g.Eventually(ctx, func() (bool, error) { return clusterwidePolicyExists(ctx, clusterwideDenyPolicyName) }).
		WithPolling(15 * time.Second).
		Should(BeFalse())
	if gone {
		logger.Log("The clusterwide deny is off")
	} else {
		logger.Log("The CiliumClusterwideNetworkPolicy %s was still there when the teardown budget ran out; tearing down anyway", clusterwideDenyPolicyName)
	}
}

func denyWCClient() (*client.Client, error) {
	return state.GetFramework().WC(state.GetCluster().Name)
}

func clusterwidePolicyExists(ctx context.Context, name string) (bool, error) {
	c, err := denyWCClient()
	if err != nil {
		return false, err
	}
	policy := &unstructured.Unstructured{}
	policy.SetAPIVersion("cilium.io/v2")
	policy.SetKind("CiliumClusterwideNetworkPolicy")
	err = c.Get(ctx, types.NamespacedName{Name: name}, policy)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

// expectClusterwideDenyOn waits for the deny and the DNS allow to exist.
func expectClusterwideDenyOn(ctx context.Context) {
	Eventually(func() error {
		for _, name := range []string{clusterwideDenyPolicyName, clusterwideAllowDNSPolicyName} {
			exists, err := clusterwidePolicyExists(ctx, name)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("the CiliumClusterwideNetworkPolicy %s doesn't exist", name)
			}
		}
		return nil
	}).
		WithTimeout(state.GetTestTimeout(timeout.DeployApps, 15*time.Minute)).
		WithPolling(15*time.Second).
		Should(Succeed(), "the clusterwide deny is not on; the cluster has to be created with ClusterwideDenyClusterValues")
}

// logClusterwideDenyState logs the clusterwide policies and the pods that aren't
// healthy, before the deny is turned off and the failure state changes.
func logClusterwideDenyState(ctx context.Context) {
	c, err := denyWCClient()
	if err != nil {
		logger.Log("Could not log the clusterwide deny state: %s", err)
		return
	}
	policies := &unstructured.UnstructuredList{}
	policies.SetAPIVersion("cilium.io/v2")
	policies.SetKind("CiliumClusterwideNetworkPolicyList")
	if err := c.List(ctx, policies); err != nil {
		logger.Log("Could not list the CiliumClusterwideNetworkPolicies: %s", err)
	} else {
		names := []string{}
		for _, p := range policies.Items {
			names = append(names, p.GetName())
		}
		logger.Log("CiliumClusterwideNetworkPolicies with the deny on: %v", names)
	}
	pods := &corev1.PodList{}
	if err := c.List(ctx, pods); err != nil {
		logger.Log("Could not list the pods: %s", err)
		return
	}
	for _, pod := range pods.Items {
		if problem := podProblem(pod); problem != "" {
			logger.Log("Pod %s/%s with the deny on: %s", pod.Namespace, pod.Name, problem)
		}
	}
}

// podProblem describes why a pod isn't healthy, or returns "". A crashlooping
// pod still has the phase Running, so containers are checked too.
func podProblem(pod corev1.Pod) string {
	if pod.Status.Phase == corev1.PodSucceeded {
		return ""
	}
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Sprintf("phase %s", pod.Status.Phase)
	}
	for _, status := range pod.Status.ContainerStatuses {
		if waiting := status.State.Waiting; waiting != nil {
			return fmt.Sprintf("container %s waiting (%s), %d restarts", status.Name, waiting.Reason, status.RestartCount)
		}
		if !status.Ready {
			return fmt.Sprintf("container %s not ready, %d restarts", status.Name, status.RestartCount)
		}
	}
	return ""
}
