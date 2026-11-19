package common

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/clustertest/v5/pkg/application"
	"github.com/giantswarm/clustertest/v5/pkg/client"
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

	// giantswarmNamespace holds Giant Swarm components next to kube-system.
	giantswarmNamespace = "giantswarm"

	// enabledKey is the switch field of denyByDefault.clusterWide and its allowDNS.
	enabledKey = "enabled"

	// restartedAtAnnotation is what `kubectl rollout restart` sets on the pod template.
	restartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"

	// clusterwideDenySettleTime is how long the deny has to be in force before the
	// checks run, so they can't pass on state from before it: Mimir answers instant
	// queries with samples up to 5 minutes old, and Teleport lists a cluster until
	// its heartbeat expires after about 10 minutes.
	clusterwideDenySettleTime = 11 * time.Minute
)

// clusterwideDenyMinVersion is the first network-policies release with the
// clusterwide deny.
var clusterwideDenyMinVersion = semver.MustParse("0.6.0")

// errNoNetworkPolicies marks a cluster without a network-policies HelmRelease.
var errNoNetworkPolicies = errors.New("the cluster has no network-policies HelmRelease")

// RunWithClusterwideDeny turns on the clusterwide deny of the network-policies
// app for the running cluster, the way a customer would, restarts the
// workloads of the default apps under it, and then runs the common checks that
// don't need customer workloads: default apps, basic workload health, cluster
// values, cert-manager, DNS, metrics, Teleport and storage. A release whose
// default apps lose traffic they need under the deny fails here, before a
// customer turns it on.
//
// The cluster is created with the deny off: a deny from the cluster's first
// minute stops default apps from installing (giantswarm/roadmap#4346). Left
// out: hello world and scale, which deploy customer-like workloads that would
// need their own allow labels, the compliance report, and provider-specific
// checks such as ECR.
func RunWithClusterwideDeny(cfg *TestConfig) {
	// ContinueOnFailure: one gap (an app losing traffic it needs) must not hide
	// the others, so every check runs even after one failed.
	Describe("with the clusterwide deny on", Ordered, ContinueOnFailure, func() {
		var (
			denyEnabled   bool
			anySpecFailed bool
			denyInForce   time.Time
			restarted     []cr.Object
		)

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

			namespaces, err := defaultAppNamespaces(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Set before the call: an update that was stored but returned an error
			// must still be undone in AfterAll, and turning it off is idempotent.
			denyEnabled = true
			// DNS in the same change as the deny, as the docs ask customers to.
			Eventually(func() error {
				return setNetworkPoliciesValues(ctx, map[string]any{
					enabledKey: true,
					"allowDNS": map[string]any{enabledKey: true},
				})
			}).
				WithTimeout(2*time.Minute).
				WithPolling(10*time.Second).
				Should(Succeed(), "could not turn the clusterwide deny on in the cluster values")

			expectClusterwideDenyOn(ctx)
			denyInForce = time.Now()
			restarted = restartWorkloads(ctx, namespaces)
		})

		ReportAfterEach(func(report SpecReport) {
			if report.Failed() {
				anySpecFailed = true
			}
		})

		AfterAll(func() {
			if !denyEnabled {
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
			turnClusterwideDenyOff(ctx)
		})

		// Its own spec, so a workload that can't restart under the deny fails here
		// without skipping the other checks.
		It("restarts the default apps' workloads under the clusterwide deny", func() {
			// Wait out the settle time even when the rollout fails, so the following
			// checks still only see state from with the deny in force.
			defer func() {
				if wait := clusterwideDenySettleTime - time.Since(denyInForce); wait > 0 {
					logger.Log("Waiting %s so the checks only see state from with the deny on", wait.Round(time.Second))
					time.Sleep(wait)
				}
			}()
			ctx := state.GetContext()
			c, err := denyWCClient()
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error { return allRolledOut(ctx, c, restarted) }).
				WithTimeout(20*time.Minute).
				WithPolling(15*time.Second).
				Should(Succeed(), "workloads did not roll out again under the clusterwide deny")
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

// defaultAppNamespaces returns the workload cluster namespaces of the default
// apps' HelmReleases, plus kube-system and giantswarm.
func defaultAppNamespaces(ctx context.Context) ([]string, error) {
	releases := &helmv2.HelmReleaseList{}
	if err := state.GetFramework().MC().List(ctx, releases, cr.InNamespace(state.GetCluster().Organization.GetNamespace())); err != nil {
		return nil, fmt.Errorf("failed to list the HelmReleases: %w", err)
	}
	seen := map[string]bool{metav1.NamespaceSystem: true, giantswarmNamespace: true}
	for _, release := range releases.Items {
		if release.Spec.KubeConfig == nil || release.Spec.TargetNamespace == "" {
			continue
		}
		seen[release.Spec.TargetNamespace] = true
	}
	namespaces := make([]string, 0, len(seen))
	for namespace := range seen {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces, nil
}

// clusterValues returns the cluster app's user config values and the key of
// their ConfigMap.
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

// setNetworkPoliciesValues merges the given fields into
// global.apps.networkPolicies.values.denyByDefault.clusterWide of the cluster
// app's user config. The cluster chart passes them to the network-policies
// HelmRelease through its user values ConfigMap.
func setNetworkPoliciesValues(ctx context.Context, clusterWide map[string]any) error {
	logger.Log("Setting denyByDefault.clusterWide %v in the cluster values", clusterWide)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		values, cm, err := clusterValues(ctx)
		if err != nil {
			return err
		}
		path := []string{"global", "apps", "networkPolicies", "values", "denyByDefault", "clusterWide"}
		existing, _, err := unstructured.NestedMap(values, path...)
		if err != nil {
			return fmt.Errorf("unexpected type at %s in the cluster values: %w", strings.Join(path, "."), err)
		}
		if existing == nil {
			existing = map[string]any{}
		}
		for k, v := range clusterWide {
			existing[k] = v
		}
		if err := unstructured.SetNestedMap(values, existing, path...); err != nil {
			return err
		}
		out, err := yaml.Marshal(values)
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["values"] = string(out)
		return state.GetFramework().MC().Update(ctx, cm)
	})
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

// expectClusterwideDenyOn waits for the deny and the DNS allow. Both existing
// proves the new values were rendered and applied: the HelmRelease spec itself
// doesn't change, the values come through a ConfigMap.
func expectClusterwideDenyOn(ctx context.Context) {
	Eventually(func() error {
		for _, name := range []string{clusterwideDenyPolicyName, clusterwideAllowDNSPolicyName} {
			exists, err := clusterwidePolicyExists(ctx, name)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("the CiliumClusterwideNetworkPolicy %s doesn't exist yet", name)
			}
		}
		return nil
	}).
		WithTimeout(15*time.Minute).
		WithPolling(15*time.Second).
		Should(Succeed(), "the clusterwide deny did not appear after changing the cluster values")
}

// restartWorkloads restarts every Deployment, DaemonSet and StatefulSet in the
// namespaces, like `kubectl rollout restart`, so their startup paths (init
// containers, leader election, webhook registration) run under the deny.
// Workloads with the OnDelete update strategy are left out, a restart doesn't
// replace their pods.
func restartWorkloads(ctx context.Context, namespaces []string) []cr.Object {
	c, err := denyWCClient()
	Expect(err).NotTo(HaveOccurred())
	now := time.Now().Format(time.RFC3339)
	patch := cr.RawPatch(types.MergePatchType, fmt.Appendf(nil,
		`{"spec":{"template":{"metadata":{"annotations":{%q:%q}}}}}`, restartedAtAnnotation, now))

	var restarted []cr.Object
	for _, namespace := range namespaces {
		deployments := &appsv1.DeploymentList{}
		Expect(c.List(ctx, deployments, cr.InNamespace(namespace))).To(Succeed())
		for i := range deployments.Items {
			restarted = append(restarted, &deployments.Items[i])
		}
		daemonSets := &appsv1.DaemonSetList{}
		Expect(c.List(ctx, daemonSets, cr.InNamespace(namespace))).To(Succeed())
		for i := range daemonSets.Items {
			if daemonSets.Items[i].Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
				logger.Log("Not restarting daemonset %s/%s, it uses the OnDelete update strategy", namespace, daemonSets.Items[i].Name)
				continue
			}
			restarted = append(restarted, &daemonSets.Items[i])
		}
		statefulSets := &appsv1.StatefulSetList{}
		Expect(c.List(ctx, statefulSets, cr.InNamespace(namespace))).To(Succeed())
		for i := range statefulSets.Items {
			if statefulSets.Items[i].Spec.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType {
				logger.Log("Not restarting statefulset %s/%s, it uses the OnDelete update strategy", namespace, statefulSets.Items[i].Name)
				continue
			}
			restarted = append(restarted, &statefulSets.Items[i])
		}
	}
	logger.Log("Restarting %d workloads in %v under the clusterwide deny", len(restarted), namespaces)
	for _, obj := range restarted {
		Expect(c.Patch(ctx, obj, patch)).To(Succeed(), "failed to restart %s/%s", obj.GetNamespace(), obj.GetName())
	}
	return restarted
}

// allRolledOut returns an error naming every workload that hasn't rolled out.
func allRolledOut(ctx context.Context, c *client.Client, workloads []cr.Object) error {
	var pending []string
	for _, obj := range workloads {
		if err := rolledOut(ctx, c, obj); err != nil {
			pending = append(pending, err.Error())
		}
	}
	if len(pending) > 0 {
		return fmt.Errorf("%d of %d workloads not rolled out:\n%s", len(pending), len(workloads), strings.Join(pending, "\n"))
	}
	return nil
}

// rolledOut reports whether a restarted workload has all its replicas updated
// and available, like `kubectl rollout status`.
func rolledOut(ctx context.Context, c *client.Client, obj cr.Object) error {
	key := cr.ObjectKeyFromObject(obj)
	switch obj.(type) {
	case *appsv1.Deployment:
		d := &appsv1.Deployment{}
		if err := c.Get(ctx, key, d); err != nil {
			return err
		}
		for _, condition := range d.Status.Conditions {
			if condition.Type == appsv1.DeploymentProgressing && condition.Reason == "ProgressDeadlineExceeded" {
				return fmt.Errorf("deployment %s exceeded its progress deadline: %s", key, condition.Message)
			}
		}
		want := int32(1)
		if d.Spec.Replicas != nil {
			want = *d.Spec.Replicas
		}
		if d.Status.ObservedGeneration < d.Generation || d.Status.UpdatedReplicas < want || d.Status.AvailableReplicas < want || d.Status.Replicas > want {
			return fmt.Errorf("deployment %s: %d/%d updated, %d available", key, d.Status.UpdatedReplicas, want, d.Status.AvailableReplicas)
		}
	case *appsv1.DaemonSet:
		ds := &appsv1.DaemonSet{}
		if err := c.Get(ctx, key, ds); err != nil {
			return err
		}
		want := ds.Status.DesiredNumberScheduled
		if ds.Status.ObservedGeneration < ds.Generation || ds.Status.UpdatedNumberScheduled < want || ds.Status.NumberAvailable < want {
			return fmt.Errorf("daemonset %s: %d/%d updated, %d available", key, ds.Status.UpdatedNumberScheduled, want, ds.Status.NumberAvailable)
		}
	case *appsv1.StatefulSet:
		sts := &appsv1.StatefulSet{}
		if err := c.Get(ctx, key, sts); err != nil {
			return err
		}
		replicas := int32(1)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}
		// With a partition, only the pods at or above it are replaced.
		updated := replicas
		if ru := sts.Spec.UpdateStrategy.RollingUpdate; ru != nil && ru.Partition != nil && *ru.Partition > 0 {
			updated = max(replicas-*ru.Partition, 0)
		}
		if sts.Status.ObservedGeneration < sts.Generation || sts.Status.UpdatedReplicas < updated || sts.Status.ReadyReplicas < replicas {
			return fmt.Errorf("statefulset %s: %d/%d updated, %d/%d ready", key, sts.Status.UpdatedReplicas, updated, sts.Status.ReadyReplicas, replicas)
		}
	}
	return nil
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

// turnClusterwideDenyOff is best effort and bounded: it only logs, since the
// cluster is deleted either way.
func turnClusterwideDenyOff(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := setNetworkPoliciesValues(ctx, map[string]any{enabledKey: false}); err != nil {
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
