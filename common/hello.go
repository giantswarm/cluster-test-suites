package common

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/organization"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runHelloWorld(externalDnsSupported bool) {
	Context("hello world", func() {
		var nginxApp, helloApp *v1alpha1.App
		var nginxConfigMap, helloConfigMap *v1.ConfigMap
		var helloWorldIngressHost string

		BeforeEach(func() {
			if !externalDnsSupported {
				Skip("external-dns is not supported")
			}

			ctx := context.Background()
			org := state.GetCluster().Organization

			// The hello-world app ingress requires a `Certificate` and a DNS record, so we need to make sure `cert-manager` and `external-dns` are deployed.
			Eventually(func() error {
				app, err := state.GetFramework().GetApp(ctx, fmt.Sprintf("%s-cert-manager", state.GetCluster().Name), org.GetNamespace())
				if err != nil {
					return err
				}
				return checkAppStatus(app)
			}).
				WithTimeout(3 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())

			Eventually(func() error {
				app, err := state.GetFramework().GetApp(ctx, fmt.Sprintf("%s-external-dns", state.GetCluster().Name), org.GetNamespace())
				if err != nil {
					return err
				}
				return checkAppStatus(app)
			}).
				WithTimeout(3 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())

			// By default, `external-dns` will only create dns records for Services on the `kube-system` namespace, because that's the default value for `namespaceFilter`.
			// That's why we install the nginx app in that namespace.
			// https://github.com/giantswarm/external-dns-app/blob/main/helm/external-dns-app/values.yaml#L114-L117
			nginxApp, nginxConfigMap = deployApp(ctx, "ingress-nginx", "kube-system", org, "3.0.0", "", map[string]string{})
			Eventually(func() error {
				app, err := state.GetFramework().GetApp(ctx, nginxApp.Name, nginxApp.Namespace)
				if err != nil {
					return err
				}
				return checkAppStatus(app)
			}).
				WithTimeout(3 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())

			helloWorldIngressHost = fmt.Sprintf("hello-world.%s", getWorkloadClusterDnsZone())
			helloAppValues := map[string]string{"IngressUrl": helloWorldIngressHost}
			helloApp, helloConfigMap = deployApp(ctx, "hello-world", "giantswarm", org, "2.0.0", "./test_data/helloworld_values.yaml", helloAppValues)
			Eventually(func() error {
				// The hello-world app creates `Ingress` resources, and the `ingress-nginx` app installed above has created some admission webhooks for `Ingress`. While `nginx` webhooks are booting, requests to them will fail to respond successfully,
				// and `Ingress` resources won't be able to be created until the webhooks are up and running. The first time we try to install the `hello-world` app, it will fail because of this.
				// `chart-operator` reconciles `charts` every 5 minutes, which means that the `hello-world` app won't be retried again until the next chart-operator reconciliation loop 5 minutes later.
				// To speed things up, we keep patching the hello-world `App` CR by adding a label. That way, we trigger reconciliation loops in chart-operator.
				now := time.Now()
				patchedApp := helloApp.DeepCopy()
				labels := patchedApp.GetLabels()
				labels["update"] = fmt.Sprintf("%d", now.Unix())
				patchedApp.SetLabels(labels)
				managementClusterKubeClient := state.GetFramework().MC()
				err := managementClusterKubeClient.Patch(ctx, patchedApp, ctrl.MergeFrom(helloApp))
				if err != nil {
					return err
				}
				app, err := state.GetFramework().GetApp(ctx, helloApp.Name, helloApp.Namespace)
				if err != nil {
					return err
				}
				return checkAppStatus(app)
			}).
				WithTimeout(6 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("hello world app responds successfully", func() {
			if !externalDnsSupported {
				Skip("external-dns is not supported")
			}

			Eventually(func() (*http.Response, error) {
				helloWorldIngressUrl := fmt.Sprintf("https://%s", helloWorldIngressHost)
				logger.Log("Trying to get a successful response from %s", helloWorldIngressUrl)
				return http.Get(helloWorldIngressUrl)
			}, "10m", "5s").Should(And(HaveHTTPStatus(http.StatusOK), ContainSubstring("Hello World")))
		})

		AfterEach(func() {
			if !externalDnsSupported {
				Skip("external-dns is not supported")
			}

			ctx := context.Background()

			managementClusterKubeClient := state.GetFramework().MC()
			err := managementClusterKubeClient.Delete(ctx, nginxApp)
			Expect(err).ShouldNot(HaveOccurred())
			err = managementClusterKubeClient.Delete(ctx, nginxConfigMap)
			Expect(err).ShouldNot(HaveOccurred())
			err = managementClusterKubeClient.Delete(ctx, helloApp)
			Expect(err).ShouldNot(HaveOccurred())
			err = managementClusterKubeClient.Delete(ctx, helloConfigMap)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
}

// deployApp creates an `App` CR for the desired application.
// It currently has some needed workarounds until we improve our `clustertest` framework.
func deployApp(ctx context.Context, name, namespace string, organization *organization.Org, version, valuesFile string, values map[string]string) (*v1alpha1.App, *v1.ConfigMap) {
	var err error
	managementClusterKubeClient := state.GetFramework().MC()
	appBuilder := application.New(fmt.Sprintf("%s-%s", state.GetCluster().Name, name), name).
		WithCatalog("giantswarm").
		WithOrganization(*organization).
		// If we don't pass any version value when building the app, the latest version will be calculated and used. But the calculation will fail when the app name doesn't match the repository name.
		WithVersion(version).
		WithInCluster(false).
		// We need to manually set this label that should be automatically added once we can set the `app.Config` property.
		WithAppLabels(map[string]string{"giantswarm.io/cluster": state.GetCluster().Name})

	if valuesFile != "" {
		appBuilder, err = appBuilder.WithValuesFile(path.Clean(valuesFile), &application.TemplateValues{
			ClusterName:  state.GetCluster().Name,
			Organization: state.GetCluster().Organization.Name,
			ExtraValues:  values,
		})
		Expect(err).ShouldNot(HaveOccurred())
	}

	app, configMap, err := appBuilder.Build()
	Expect(err).ShouldNot(HaveOccurred())

	err = managementClusterKubeClient.Create(ctx, configMap)
	Expect(err).ShouldNot(HaveOccurred())

	// We need to set these properties manually after building, because they are calculated using the `app.Config` property that we currently can't set.
	app.Spec.Config.ConfigMap.Name = fmt.Sprintf("%s-cluster-values", state.GetCluster().Name)
	app.Spec.Config.ConfigMap.Namespace = organization.GetNamespace()
	app.Spec.KubeConfig.Context.Name = fmt.Sprintf("%s-admin@%s", state.GetCluster().Name, state.GetCluster().Name)
	app.Spec.KubeConfig.Secret.Name = fmt.Sprintf("%s-kubeconfig", state.GetCluster().Name)

	// We need to set the namespace that will be used when installing the chart in the WC. It's not the organization namespace, because we don't have orgs in WCs.
	app.Spec.Namespace = namespace
	err = managementClusterKubeClient.Create(ctx, app)
	Expect(err).ShouldNot(HaveOccurred())

	return app, configMap
}

func getWorkloadClusterDnsZone() string {
	values := &application.DefaultAppsValues{}
	err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
	Expect(err).NotTo(HaveOccurred())

	if values.BaseDomain == "" {
		Fail("baseDomain field missing from cluster helm values")
	}

	return values.BaseDomain
}
