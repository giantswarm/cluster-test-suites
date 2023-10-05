package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/organization"
	"github.com/giantswarm/clustertest/pkg/utils"
	"github.com/giantswarm/clustertest/pkg/wait"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/cmd/standup/types"
	"github.com/giantswarm/cluster-test-suites/providers/capa"
	"github.com/giantswarm/cluster-test-suites/providers/capv"
	"github.com/giantswarm/cluster-test-suites/providers/capvcd"
	"github.com/giantswarm/cluster-test-suites/providers/capz"
)

var (
	standupCmd = &cobra.Command{
		Use:     "standup",
		Long:    "Standup create a test workload cluster in a standard, reproducible way.\nA valid Management Cluster kubeconfig must be available and set to the `E2E_KUBECONFIG` environment variable.",
		Example: "standup --provider aws --context capa --cluster-values ./cluster_values.yaml --default-apps-values ./default-apps_values.yaml",
		Args:    cobra.NoArgs,
		RunE:    run,
	}

	provider          string
	kubeContext       string
	clusterValues     string
	defaultAppValues  string
	clusterVersion    string
	defaultAppVersion string
	outputDirectory   string

	controlPlaneNodes int
	workerNodes       int
)

func init() {
	standupCmd.Flags().StringVar(&provider, "provider", "", "The provider (required)")
	standupCmd.Flags().StringVar(&kubeContext, "context", "", "The kubernetes context to use (required)")
	standupCmd.Flags().StringVar(&clusterValues, "cluster-values", "", "The path to the cluster app values (required)")
	standupCmd.Flags().StringVar(&defaultAppValues, "default-apps-values", "", "The path to the default-apps app values (required)")

	standupCmd.Flags().IntVar(&controlPlaneNodes, "control-plane-nodes", 1, "The number of control plane nodes to wait for being ready")
	standupCmd.Flags().IntVar(&workerNodes, "worker-nodes", 1, "The number of worker nodes to wait for being ready")
	standupCmd.Flags().StringVar(&outputDirectory, "output", "./", "The directory to store the results.json and kubeconfig in")
	standupCmd.Flags().StringVar(&clusterVersion, "cluster-version", "latest", "The version of the cluster app to install")
	standupCmd.Flags().StringVar(&defaultAppVersion, "default-apps-version", "latest", "The version of the default-apps app to install")

	standupCmd.MarkFlagRequired("provider")
	standupCmd.MarkFlagRequired("context")
	standupCmd.MarkFlagRequired("cluster-values")
	standupCmd.MarkFlagRequired("default-apps-values")
}

type Timing struct {
	Name      string
	Timestamp time.Time
	Err       error
}

var timings = []Timing{}

func main() {
	if err := standupCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	defer func() {
		durations := []string{}
		w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
		for i, timing := range timings {
			duration := time.Duration(0)
			if i > 0 {
				duration = timing.Timestamp.Sub(timings[i-1].Timestamp)
			}
			fmt.Fprintf(w, "%d\t| %s\t| %v\t| %v\n", i, timing.Name, timing.Timestamp, duration)
			if timing.Err != nil {
				durations = append(durations, "X")
			} else {
				durations = append(durations, fmt.Sprintf("%f", duration.Seconds()))
			}
		}
		w.Flush()
		fmt.Println(strings.Join(durations, ","))
	}()

	cmd.SilenceUsage = true

	ctx := context.Background()

	framework, err := clustertest.New(kubeContext)
	if err != nil {
		return err
	}

	provider := application.Provider(provider)
	clusterName := utils.GenerateRandomName("t")
	orgName := utils.GenerateRandomName("t")

	fmt.Printf("Standing up cluster...\n\nProvider:\t\t%s\nCluster Name:\t\t%s\nOrg Name:\t\t%s\nResults Directory:\t%s\n\n", provider, clusterName, orgName, outputDirectory)

	var cluster *application.Cluster
	switch provider {
	case application.ProviderVSphere:
		cluster = capv.NewClusterApp(clusterName, orgName, clusterValues, defaultAppValues)
	case application.ProviderCloudDirector:
		cluster = capvcd.NewClusterApp(clusterName, orgName, clusterValues, defaultAppValues)
	case application.ProviderAWS:
		cluster = capa.NewClusterApp(clusterName, orgName, clusterValues, defaultAppValues)
	case application.ProviderAzure:
		cluster = capz.NewClusterApp(clusterName, orgName, clusterValues, defaultAppValues)
	default:
		cluster = application.NewClusterApp(clusterName, provider).
			WithAppVersions(clusterVersion, defaultAppVersion).
			WithOrg(organization.New(orgName)).
			WithAppValuesFile(path.Clean(clusterValues), path.Clean(defaultAppValues), &application.TemplateValues{
				ClusterName:  clusterName,
				Organization: orgName,
			})
	}

	applyCtx, cancelApplyCtx := context.WithTimeout(ctx, 20*time.Minute)
	defer cancelApplyCtx()

	timings = append(timings, Timing{Name: "0-start", Timestamp: time.Now()})
	wcClient, err := framework.ApplyCluster(applyCtx, cluster)
	if err != nil {
		return err
	}
	timings = append(timings, Timing{Name: "1-api-available", Timestamp: time.Now(), Err: err})

	kubeconfigFile, err := os.Create(path.Join(outputDirectory, "kubeconfig"))
	if err != nil {
		return err
	}
	defer kubeconfigFile.Close()

	kubeconfig, err := framework.MC().GetClusterKubeConfig(ctx, cluster.Name, cluster.GetNamespace())
	if err != nil {
		return err
	}
	_, err = kubeconfigFile.Write([]byte(kubeconfig))
	if err != nil {
		return err
	}

	resultsFile, err := os.Create(path.Join(outputDirectory, "results.json"))
	if err != nil {
		return err
	}
	defer resultsFile.Close()
	result := types.StandupResult{
		Provider:       string(provider),
		ClusterName:    clusterName,
		OrgName:        orgName,
		Namespace:      cluster.GetNamespace(),
		ClusterVersion: cluster.ClusterApp.Version,
		KubeconfigPath: kubeconfigFile.Name(),
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = resultsFile.Write(resultBytes)

	wait.For(
		wait.AreNumNodesReady(ctx, wcClient, controlPlaneNodes, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
		wait.WithTimeout(wait.DefaultTimeout),
		wait.WithInterval(15*time.Second),
	)
	timings = append(timings, Timing{Name: "2-single-control-plane-node-ready", Timestamp: time.Now(), Err: err})

	wait.For(
		wait.AreNumNodesReady(ctx, wcClient, workerNodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"}),
		wait.WithTimeout(wait.DefaultTimeout),
		wait.WithInterval(15*time.Second),
	)
	timings = append(timings, Timing{Name: "3-single-worker-node-ready", Timestamp: time.Now(), Err: err})

	defaultAppsAppName := fmt.Sprintf("%s-%s", cluster.Name, "default-apps")
	wait.For(
		wait.IsAppDeployed(ctx, framework.MC(), defaultAppsAppName, cluster.Organization.GetNamespace()),
		wait.WithTimeout(wait.DefaultTimeout),
		wait.WithInterval(50*time.Millisecond),
	)
	timings = append(timings, Timing{Name: "4-default-apps-deployed", Timestamp: time.Now(), Err: err})

	appList := &v1alpha1.AppList{}
	err = framework.MC().List(ctx, appList, cr.InNamespace(cluster.Organization.GetNamespace()), cr.MatchingLabels{"giantswarm.io/managed-by": defaultAppsAppName})
	if err != nil {
		return err
	}
	appNamespacedNames := []apitypes.NamespacedName{}
	for _, app := range appList.Items {
		appNamespacedNames = append(appNamespacedNames, apitypes.NamespacedName{Name: app.Name, Namespace: app.Namespace})
	}
	wait.For(
		wait.IsAllAppDeployed(ctx, framework.MC(), appNamespacedNames),
		wait.WithTimeout(wait.DefaultTimeout),
		wait.WithInterval(10*time.Second),
	)
	timings = append(timings, Timing{Name: "5-all-default-apps-deployed", Timestamp: time.Now(), Err: err})

	wait.For(func() (bool, error) {
		podList := &corev1.PodList{}
		err := wcClient.List(context.Background(), podList)
		if err != nil {
			return false, err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				return false, fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		return true, nil
	},
		wait.WithTimeout(wait.DefaultTimeout),
		wait.WithInterval(wait.DefaultInterval),
	)
	timings = append(timings, Timing{Name: "6-all-pods-running", Timestamp: time.Now(), Err: err})

	return err
}
