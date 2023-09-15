package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runHelloWorld(externalDnsSupported bool) {
	Context("hello world", func() {
		var (
			nginxApp              *application.Application
			helloApp              *application.Application
			helloWorldIngressHost string
			helloWorldIngressUrl  string
		)

		const (
			appReadyTimeout  = 3 * time.Minute
			appReadyInterval = 5 * time.Second
		)

		BeforeEach(func() {
			if !externalDnsSupported {
				Skip("external-dns is not supported")
			}

			ctx := context.Background()
			org := state.GetCluster().Organization

			// The hello-world app ingress requires a `Certificate` and a DNS record, so we need to make sure `cert-manager` and `external-dns` are deployed.
			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), fmt.Sprintf("%s-cert-manager", state.GetCluster().Name), org.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue())

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), fmt.Sprintf("%s-external-dns", state.GetCluster().Name), org.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue())

			// By default, `external-dns` will only create dns records for Services on the `kube-system` namespace, because that's the default value for `namespaceFilter`.
			// That's why we install the nginx app in that namespace.
			// https://github.com/giantswarm/external-dns-app/blob/main/helm/external-dns-app/values.yaml#L114-L117
			nginxApp = application.New(fmt.Sprintf("%s-ingress-nginx", state.GetCluster().Name), "ingress-nginx").
				WithCatalog("giantswarm").
				WithOrganization(*org).
				WithVersion("latest").
				WithClusterName(state.GetCluster().Name).
				WithInCluster(false).
				WithInstallNamespace("kube-system")

			err := state.GetFramework().MC().DeployApp(ctx, *nginxApp)
			Expect(err).To(BeNil())

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), nginxApp.InstallName, nginxApp.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue())

			helloWorldIngressHost = fmt.Sprintf("hello-world.%s", getWorkloadClusterDnsZone())
			helloWorldIngressUrl = fmt.Sprintf("https://%s", helloWorldIngressHost)
			helloAppValues := map[string]string{"IngressUrl": helloWorldIngressHost}

			helloApp = application.New(fmt.Sprintf("%s-hello-world", state.GetCluster().Name), "hello-world").
				WithCatalog("giantswarm").
				WithOrganization(*org).
				WithVersion("latest").
				WithClusterName(state.GetCluster().Name).
				WithInCluster(false).
				WithInstallNamespace("giantswarm").
				MustWithValuesFile("./test_data/helloworld_values.yaml", &application.TemplateValues{
					ClusterName:  state.GetCluster().Name,
					Organization: state.GetCluster().Organization.Name,
					ExtraValues:  helloAppValues,
				})

			err = state.GetFramework().MC().DeployApp(ctx, *helloApp)
			Expect(err).To(BeNil())

			Eventually(func() (bool, error) {
				// The hello-world app creates `Ingress` resources, and the `ingress-nginx` app installed above has created some admission webhooks for `Ingress`. While `nginx` webhooks are booting, requests to them will fail to respond successfully,
				// and `Ingress` resources won't be able to be created until the webhooks are up and running. The first time we try to install the `hello-world` app, it will fail because of this.
				// `chart-operator` reconciles `charts` every 5 minutes, which means that the `hello-world` app won't be retried again until the next chart-operator reconciliation loop 5 minutes later.
				// To speed things up, we keep patching the hello-world `App` CR by adding a label. That way, we trigger reconciliation loops in chart-operator.
				managementClusterKubeClient := state.GetFramework().MC()

				helloApplication := &v1alpha1.App{}
				err := managementClusterKubeClient.Get(ctx, types.NamespacedName{Name: helloApp.InstallName, Namespace: helloApp.GetNamespace()}, helloApplication)
				if err != nil {
					return false, err
				}

				now := time.Now()
				patchedApp := helloApplication.DeepCopy()
				labels := patchedApp.GetLabels()
				labels["update"] = fmt.Sprintf("%d", now.Unix())
				patchedApp.SetLabels(labels)

				err = managementClusterKubeClient.Patch(ctx, patchedApp, ctrl.MergeFrom(helloApplication))
				if err != nil {
					return false, err
				}

				return wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), helloApp.InstallName, helloApp.GetNamespace())()
			}).
				WithTimeout(6 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		It("hello world app responds successfully", func() {
			if !externalDnsSupported {
				Skip("external-dns is not supported")
			}

			Eventually(func() (string, error) {
				logger.Log("Trying to get a successful response from %s", helloWorldIngressUrl)
				resp, err := http.Get(helloWorldIngressUrl)
				if err != nil {
					return "", err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return "", err
				}

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return "", err
				}

				return string(bodyBytes), nil
			}, "10m", "5s").Should(ContainSubstring("Hello World"))
		})

		AfterEach(func() {
			if !externalDnsSupported {
				Skip("external-dns is not supported")
			}

			err := state.GetFramework().MC().DeleteApp(state.GetContext(), *nginxApp)
			Expect(err).ShouldNot(HaveOccurred())
			err = state.GetFramework().MC().DeleteApp(state.GetContext(), *helloApp)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
}

func getWorkloadClusterDnsZone() string {
	values := &application.DefaultAppsValues{}
	err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
	Expect(err).NotTo(HaveOccurred())

	if values.BaseDomain == "" {
		Fail("baseDomain field missing from cluster helm values")
	}

	return fmt.Sprintf("%s.%s", state.GetCluster().Name, values.BaseDomain)
}
