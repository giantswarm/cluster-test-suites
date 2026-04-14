package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/v4/pkg/application"
	"github.com/giantswarm/clustertest/v4/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v4/pkg/logger"
	"github.com/giantswarm/clustertest/v4/pkg/net"
	"github.com/giantswarm/clustertest/v4/pkg/wait"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/giantswarm/cluster-test-suites/v6/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v6/internal/state"
)

func runHelloWorldGateway(gatewayAPISupported bool) {
	Context("hello world via gateway api", Ordered, func() {
		var (
			awsLBControllerApp  *application.Application
			gatewayAPIApp       *application.Application
			helloHelmRelease    *unstructured.Unstructured
			ociRepoName         string
			helloWorldHost      string
			helloWorldUrl       string
			awsLBDeployed       bool
		)

		const (
			appReadyTimeout  = 3 * time.Minute
			appReadyInterval = 5 * time.Second
		)

		BeforeEach(func() {
			if !gatewayAPISupported {
				Skip("Gateway API is not supported")
			}
		})

		It("should have cert-manager and external-dns deployed", func() {
			org := state.GetCluster().Organization

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), fmt.Sprintf("%s-cert-manager", state.GetCluster().Name), org.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate 'cert-manager' App not ready"))

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), fmt.Sprintf("%s-external-dns", state.GetCluster().Name), org.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate 'external-dns' App not ready"))
		})

		It("should deploy aws-lb-controller-bundle", func() {
			const awsLBValuesFile = "./test_data/aws-lb-controller-bundle_values.yaml"

			if !helper.FileExists(awsLBValuesFile) {
				Skip("aws-lb-controller-bundle values file not found, skipping")
			}

			org := state.GetCluster().Organization
			bundleAppName := fmt.Sprintf("%s-aws-lb-controller-bundle", state.GetCluster().Name)

			awsLBControllerApp = application.New(bundleAppName, "aws-lb-controller-bundle").
				WithCatalog("giantswarm").
				WithOrganization(*org).
				WithVersion("latest").
				WithClusterName(state.GetCluster().Name).
				WithInCluster(false).
				WithInstallNamespace(org.GetNamespace()).
				MustWithValuesFile(awsLBValuesFile, &application.TemplateValues{
					ClusterName: state.GetCluster().Name,
					ExtraValues: map[string]string{
						"Installation": state.GetFramework().MC().GetClusterName(),
					},
				})

			err := state.GetFramework().MC().DeployApp(state.GetContext(), *awsLBControllerApp)
			Expect(err).To(BeNil())

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), awsLBControllerApp.InstallName, awsLBControllerApp.GetNamespace())).
				WithTimeout(5*time.Minute).
				WithPolling(10*time.Second).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate aws-lb-controller-bundle App not ready"))

			// Wait for child apps
			appList := &v1alpha1.AppList{}
			err = state.GetFramework().MC().List(state.GetContext(), appList, ctrl.InNamespace(org.GetNamespace()), ctrl.MatchingLabels{"giantswarm.io/managed-by": bundleAppName})
			Expect(err).NotTo(HaveOccurred())

			appNamespacedNames := []types.NamespacedName{}
			for _, app := range appList.Items {
				appNamespacedNames = append(appNamespacedNames, types.NamespacedName{Name: app.Name, Namespace: app.Namespace})
			}

			Eventually(wait.IsAllAppDeployed(state.GetContext(), state.GetFramework().MC(), appNamespacedNames)).
				WithTimeout(10*time.Minute).
				WithPolling(10*time.Second).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate aws-lb-controller-bundle child Apps not ready"))

			awsLBDeployed = true
		})

		It("should deploy gateway-api-bundle", func() {
			org := state.GetCluster().Organization
			bundleAppName := fmt.Sprintf("%s-gateway-api-bundle", state.GetCluster().Name)

			gatewayAPIApp = application.New(bundleAppName, "gateway-api-bundle").
				WithCatalog("giantswarm").
				WithOrganization(*org).
				WithVersion("latest").
				WithClusterName(state.GetCluster().Name).
				WithInCluster(false).
				WithInstallNamespace(org.GetNamespace()).
				MustWithValuesFile("./test_data/gateway-api-bundle_values.yaml", &application.TemplateValues{
					ClusterName: state.GetCluster().Name,
				})

			err := state.GetFramework().MC().DeployApp(state.GetContext(), *gatewayAPIApp)
			Expect(err).To(BeNil())

			Eventually(wait.IsAppDeployed(state.GetContext(), state.GetFramework().MC(), gatewayAPIApp.InstallName, gatewayAPIApp.GetNamespace())).
				WithTimeout(5*time.Minute).
				WithPolling(10*time.Second).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate gateway-api-bundle App not ready"))

			// Wait for child apps
			appList := &v1alpha1.AppList{}
			err = state.GetFramework().MC().List(state.GetContext(), appList, ctrl.InNamespace(org.GetNamespace()), ctrl.MatchingLabels{"giantswarm.io/managed-by": bundleAppName})
			Expect(err).NotTo(HaveOccurred())

			appNamespacedNames := []types.NamespacedName{}
			for _, app := range appList.Items {
				appNamespacedNames = append(appNamespacedNames, types.NamespacedName{Name: app.Name, Namespace: app.Namespace})
			}

			Eventually(wait.IsAllAppDeployed(state.GetContext(), state.GetFramework().MC(), appNamespacedNames)).
				WithTimeout(10*time.Minute).
				WithPolling(10*time.Second).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate gateway-api-bundle child Apps not ready"))
		})

		It("cluster wildcard DNS must be resolvable", func() {
			resolver := net.NewResolver()
			Eventually(func() (bool, error) {
				result, err := resolver.LookupIP(context.Background(), "ip", fmt.Sprintf("hello-world.%s", getWorkloadClusterDnsZone()))
				if err != nil {
					return false, err
				}
				if len(result) == 0 {
					return false, fmt.Errorf("no IP found for hello-world.%s", getWorkloadClusterDnsZone())
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

		It("certificate in envoy-gateway-system should be ready", func() {
			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func() error {
				certList := &certmanager.CertificateList{}
				err := wcClient.List(state.GetContext(), certList, ctrl.InNamespace("envoy-gateway-system"))
				if err != nil {
					return err
				}

				if len(certList.Items) == 0 {
					return fmt.Errorf("no certificates found in envoy-gateway-system")
				}

				for _, cert := range certList.Items {
					ready := false
					conditionMessage := "(no status message found)"
					for _, condition := range cert.Status.Conditions {
						if condition.Type == certmanager.CertificateConditionReady && condition.Status == "True" {
							ready = true
							break
						} else if condition.Type == certmanager.CertificateConditionReady {
							conditionMessage = condition.Message
						}
					}
					if !ready {
						logger.Log("Certificate '%s' in 'envoy-gateway-system' is not Ready - '%s'", cert.Name, conditionMessage)
						return fmt.Errorf("certificate '%s' is not ready", cert.Name)
					}
					logger.Log("Certificate '%s' in 'envoy-gateway-system' is Ready", cert.Name)
				}

				return nil
			}).
				WithTimeout(15*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(
					Succeed(),
					failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate why Certificate in envoy-gateway-system is not ready"),
				)
		})

		It("should deploy hello-world app with HTTPRoute", func() {
			org := state.GetCluster().Organization
			clusterName := state.GetCluster().Name
			namespace := org.GetNamespace()
			helloWorldHost = fmt.Sprintf("hello-world.%s", getWorkloadClusterDnsZone())
			helloWorldUrl = fmt.Sprintf("https://%s", helloWorldHost)

			ociRepoName = fmt.Sprintf("%s-hello-world-chart", clusterName)
			err := ensureTestOCIRepository(state.GetContext(), state.GetFramework().MC(), ociRepoName, namespace, "hello-world")
			Expect(err).To(BeNil())

			values, err := parseValuesFile("./test_data/helloworld_route_values.yaml", &HelmReleaseTemplateValues{
				ClusterName: clusterName,
				ExtraValues: map[string]string{"IngressUrl": helloWorldHost},
			})
			Expect(err).To(BeNil())

			helloHelmRelease = newTestHelmRelease(
				fmt.Sprintf("%s-hello-world-gateway", clusterName),
				namespace,
				"hello-world",
				"giantswarm",
				clusterName,
				ociRepoName,
				values,
			)

			err = state.GetFramework().MC().Create(state.GetContext(), helloHelmRelease)
			Expect(err).To(BeNil())

			Eventually(isHelmReleaseReady(state.GetContext(), state.GetFramework().MC(), types.NamespacedName{
				Name:      helloHelmRelease.GetName(),
				Namespace: helloHelmRelease.GetNamespace(),
			})).
				WithTimeout(6*time.Minute).
				WithPolling(5*time.Second).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate 'hello-world' gateway HelmRelease is not ready"))
		})

		It("HTTPRoute should be accepted by gateway", func() {
			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func() (bool, error) {
				httpRoute := &gatewayv1.HTTPRoute{}
				err := wcClient.Get(state.GetContext(), types.NamespacedName{Name: "hello-world", Namespace: "giantswarm"}, httpRoute)
				if err != nil {
					logger.Log("Failed to get HTTPRoute: %v", err)
					return false, err
				}

				if len(httpRoute.Status.Parents) == 0 {
					logger.Log("HTTPRoute has no parent status yet")
					return false, nil
				}

				accepted := false
				resolvedRefs := false
				for _, condition := range httpRoute.Status.Parents[0].Conditions {
					if condition.Type == "Accepted" && condition.Status == metav1.ConditionTrue {
						accepted = true
					}
					if condition.Type == "ResolvedRefs" && condition.Status == metav1.ConditionTrue {
						resolvedRefs = true
					}
				}

				if !accepted || !resolvedRefs {
					logger.Log("HTTPRoute not yet accepted: accepted=%v resolvedRefs=%v", accepted, resolvedRefs)
					return false, nil
				}

				return true, nil
			}).
				WithTimeout(6*time.Minute).
				WithPolling(5*time.Second).
				Should(BeTrue(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate HTTPRoute 'hello-world' not accepted by gateway"))
		})

		It("hello world app responds successfully", func() {
			httpClient := net.NewHttpClient()

			Eventually(func() (string, error) {
				logger.Log("Trying to get a successful response from %s", helloWorldUrl)
				resp, err := httpClient.Get(helloWorldUrl)
				if err != nil {
					return "", err
				}
				defer resp.Body.Close() // nolint:errcheck

				if resp.StatusCode != http.StatusOK {
					logger.Log("Was expecting status code '%d' but actually got '%d'", http.StatusOK, resp.StatusCode)
					return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				}

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return "", err
				}

				return string(bodyBytes), nil
			}).
				WithTimeout(15*time.Minute).
				WithPolling(5*time.Second).
				Should(
					ContainSubstring("Hello World"),
					failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), "Investigate why I don't get a successful response from the 'hello-world' gateway application deployed in the 'giantswarm' namespace"),
				)
		})

		It("uninstall apps", func() {
			if helloHelmRelease != nil {
				err := state.GetFramework().MC().Delete(state.GetContext(), helloHelmRelease)
				Expect(err).ShouldNot(HaveOccurred())

				err = deleteTestOCIRepository(state.GetContext(), state.GetFramework().MC(), ociRepoName, helloHelmRelease.GetNamespace())
				Expect(err).ShouldNot(HaveOccurred())
			}
			if gatewayAPIApp != nil {
				err := state.GetFramework().MC().DeleteApp(state.GetContext(), *gatewayAPIApp)
				Expect(err).ShouldNot(HaveOccurred())
			}
			if awsLBDeployed && awsLBControllerApp != nil {
				err := state.GetFramework().MC().DeleteApp(state.GetContext(), *awsLBControllerApp)
				Expect(err).ShouldNot(HaveOccurred())
			}
		})
	})
}
