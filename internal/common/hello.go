package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/net"
	"github.com/giantswarm/clustertest/pkg/wait"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/helper"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runHelloWorld(externalDnsSupported bool) {
	Context("hello world", Ordered, func() {
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
		})

		It("should have cert-manager and external-dns deployed", func() {
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
		})

		It("should deploy ingress-nginx", func() {
			org := state.GetCluster().Organization

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

			// check if ./test_data/ingress-nginx_values.yaml exists
			path := "./test_data/ingress-nginx_values.yaml"
			if helper.FileExists(path) {
				nginxApp = nginxApp.MustWithValuesFile(path, &application.TemplateValues{})
			}

			err := state.GetFramework().MC().DeployApp(state.GetContext(), *nginxApp)
			Expect(err).To(BeNil())

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), nginxApp.InstallName, nginxApp.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue())
		})

		It("cluster wildcard ingress DNS must be resolvable", func() {
			resolver := net.NewResolver()
			Eventually(func() (bool, error) {
				result, err := resolver.LookupIP(context.Background(), "ip", fmt.Sprintf("hello-world.%s", getWorkloadClusterDnsZone()))
				if err != nil {
					return false, err
				}
				if len(result) == 0 {
					return false, fmt.Errorf("no IP found for ingress.%s", getWorkloadClusterDnsZone())
				}
				var resultString []string
				for _, ip := range result {
					resultString = append(resultString, ip.String())
				}
				logger.Log("DNS record 'hello-world.%s' resolved to %s", getWorkloadClusterDnsZone(), resultString)
				return true, nil
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(BeTrue())
		})

		It("should deploy the hello-world app", func() {
			org := state.GetCluster().Organization

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

			err := state.GetFramework().MC().DeployApp(state.GetContext(), *helloApp)
			Expect(err).To(BeNil())

			Eventually(func() (bool, error) {
				// The hello-world app creates `Ingress` resources, and the `ingress-nginx` app installed above has created some admission webhooks for `Ingress`. While `nginx` webhooks are booting, requests to them will fail to respond successfully,
				// and `Ingress` resources won't be able to be created until the webhooks are up and running. The first time we try to install the `hello-world` app, it will fail because of this.
				// `chart-operator` reconciles `charts` every 5 minutes, which means that the `hello-world` app won't be retried again until the next chart-operator reconciliation loop 5 minutes later.
				// To speed things up, we keep patching the hello-world `App` CR by adding a label. That way, we trigger reconciliation loops in chart-operator.
				managementClusterKubeClient := state.GetFramework().MC()

				helloApplication := &v1alpha1.App{}
				err := managementClusterKubeClient.Get(state.GetContext(), types.NamespacedName{Name: helloApp.InstallName, Namespace: helloApp.GetNamespace()}, helloApplication)
				if err != nil {
					return false, err
				}

				now := time.Now()
				patchedApp := helloApplication.DeepCopy()
				labels := patchedApp.GetLabels()
				labels["update"] = fmt.Sprintf("%d", now.Unix())
				patchedApp.SetLabels(labels)

				err = managementClusterKubeClient.Patch(state.GetContext(), patchedApp, ctrl.MergeFrom(helloApplication))
				if err != nil {
					return false, err
				}

				return wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), helloApp.InstallName, helloApp.GetNamespace())()
			}).
				WithTimeout(6 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		It("ingress resource has load balancer in status", func() {
			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func() (bool, error) {
				logger.Log("Checking if ingress has load balancer set in status")
				helloIngress := networkingv1.Ingress{}
				err := wcClient.Get(state.GetContext(), types.NamespacedName{Name: "hello-world", Namespace: "giantswarm"}, &helloIngress)
				if err != nil {
					logger.Log("Failed to get ingress: %v", err)
					return false, err
				}

				if len(helloIngress.Status.LoadBalancer.Ingress) > 0 &&
					helloIngress.Status.LoadBalancer.Ingress[0].Hostname != "" {

					logger.Log("Load balancer hostname found in ingress status: %s", helloIngress.Status.LoadBalancer.Ingress[0].Hostname)
					return true, nil
				}

				return false, nil
			}).
				WithTimeout(6 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		It("should have a ready Certificate generated", func() {
			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).ShouldNot(HaveOccurred())

			certificateName := "hello-world-tls"
			certificateNamespace := "giantswarm"

			Eventually(func() error {
				logger.Log("Checking for certificate '%s' in namespace '%s'", certificateName, certificateNamespace)

				cert := &certmanager.Certificate{
					ObjectMeta: v1.ObjectMeta{
						Name:      certificateName,
						Namespace: certificateNamespace,
					},
				}
				err := wcClient.Get(state.GetContext(), ctrl.ObjectKeyFromObject(cert), cert)
				if err != nil {
					return err
				}

				conditionMessage := "(no status message found)"
				for _, condition := range cert.Status.Conditions {
					if condition.Type == certmanager.CertificateConditionReady && condition.Status == "True" {
						logger.Log("Found status.condition with type '%s' and status '%s' in Certificate '%s'", condition.Type, condition.Status, certificateName)
						return nil
					} else if condition.Type == certmanager.CertificateConditionReady {
						conditionMessage = condition.Message
					}
				}

				logger.Log("Certificate '%s' is not Ready - '%s'", certificateName, conditionMessage)

				return fmt.Errorf("certificate is not ready")
			}).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("hello world app responds successfully", func() {
			httpClient := net.NewHttpClient()

			Eventually(func() (string, error) {
				logger.Log("Trying to get a successful response from %s", helloWorldIngressUrl)
				resp, err := httpClient.Get(helloWorldIngressUrl)
				if err != nil {
					return "", err
				}
				defer resp.Body.Close() // nolint:errcheck

				if resp.StatusCode != http.StatusOK {
					logger.Log("Was expecting status code '%d' but actually got '%d'", http.StatusOK, resp.StatusCode)
					return "", err
				}

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Log("Was not expecting the response body to be empty")
					return "", err
				}

				return string(bodyBytes), nil
			}).
				WithTimeout(15 * time.Minute).
				WithPolling(5 * time.Second).
				Should(ContainSubstring("Hello World"))
		})

		It("uninstall apps", func() {
			err := state.GetFramework().MC().DeleteApp(state.GetContext(), *nginxApp)
			Expect(err).ShouldNot(HaveOccurred())
			err = state.GetFramework().MC().DeleteApp(state.GetContext(), *helloApp)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
}

func getWorkloadClusterDnsZone() string {
	values := &application.ClusterValues{}
	err := state.GetFramework().MC().GetHelmValues(state.GetCluster().Name, state.GetCluster().GetNamespace(), values)
	Expect(err).NotTo(HaveOccurred())

	if values.BaseDomain == "" {
		Fail("baseDomain field missing from cluster helm values")
	}

	return fmt.Sprintf("%s.%s", state.GetCluster().Name, values.BaseDomain)
}
