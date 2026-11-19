package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/giantswarm/clustertest/v5/pkg/application"
	"github.com/giantswarm/clustertest/v5/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v5/pkg/helmrelease"
	"github.com/giantswarm/clustertest/v5/pkg/logger"
	"github.com/giantswarm/clustertest/v5/pkg/net"
	"github.com/giantswarm/clustertest/v5/pkg/wait"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/v7/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

func runHelloWorldGateway(gatewayAPISupported bool) {
	Context("hello world via gateway api", Ordered, func() {
		var (
			helloHelmRelease      *helmv2.HelmRelease
			awsLBHelmRelease      *helmv2.HelmRelease
			gatewayAPIHelmRelease *helmv2.HelmRelease
			ociRepoName           string
			awsLBOCIRepoName      string
			gatewayAPIOCIRepoName string
			helloWorldHost        string
			helloWorldUrl         string
		)

		const appReadyInterval = 5 * time.Second

		BeforeEach(func() {
			if !gatewayAPISupported {
				Skip("Gateway API is not supported")
			}
		})

		It("should have cert-manager and external-dns deployed", func() {
			org := state.GetCluster().Organization

			// appReadyTimeout bounds the cert-manager/external-dns app readiness
			// waits. Overridable so slow clusters (app reconcile + LB
			// provisioning) don't fail on a merely-slow environment. Resolved
			// here (spec runtime) rather than at tree construction, since the
			// state context is only set in BeforeSuite.
			appReadyTimeout := state.GetTestTimeout(timeout.GatewayAppReady, 5*time.Minute)

			Eventually(helmrelease.IsAppOrHelmReleaseReady(state.GetContext(), state.GetFramework().MC(), fmt.Sprintf("%s-cert-manager", state.GetCluster().Name), org.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue())

			Eventually(helmrelease.IsAppOrHelmReleaseReady(state.GetContext(), state.GetFramework().MC(), fmt.Sprintf("%s-external-dns", state.GetCluster().Name), org.GetNamespace())).
				WithTimeout(appReadyTimeout).
				WithPolling(appReadyInterval).
				Should(BeTrue())
		})

		It("should deploy aws-lb-controller-bundle", func() {
			const awsLBValuesFile = "./test_data/aws-lb-controller-bundle_values.yaml"

			if !helper.FileExists(awsLBValuesFile) {
				Skip("aws-lb-controller-bundle values file not found, skipping")
			}

			clusterName := state.GetCluster().Name
			namespace := state.GetCluster().Organization.GetNamespace()

			awsLBOCIRepoName = fmt.Sprintf("%s-aws-lb-controller-bundle", clusterName)
			err := helmrelease.EnsureOCIRepository(state.GetContext(), state.GetFramework().MC(), awsLBOCIRepoName, namespace, "aws-lb-controller-bundle")
			Expect(err).To(BeNil())

			hrBuilder, err := helmrelease.New(
				fmt.Sprintf("%s-aws-lb-controller-bundle", clusterName),
				"aws-lb-controller-bundle",
			).
				WithNamespace(namespace).
				WithClusterName(clusterName).
				WithInCluster(true).
				WithTargetNamespace(namespace).
				WithServiceAccountName("automation").
				WithValuesFile(awsLBValuesFile, &helmrelease.TemplateValues{
					ClusterName: clusterName,
					ExtraValues: map[string]string{
						"Installation": state.GetFramework().MC().GetClusterName(),
					},
				})
			Expect(err).To(BeNil())
			awsLBHelmRelease, err = hrBuilder.Build()
			Expect(err).To(BeNil())

			err = state.GetFramework().MC().Create(state.GetContext(), awsLBHelmRelease)
			Expect(err).To(BeNil())

			Eventually(helmrelease.IsHelmReleaseReady(state.GetContext(), state.GetFramework().MC(), awsLBHelmRelease.GetName(), awsLBHelmRelease.GetNamespace())).
				WithTimeout(15 * time.Minute).
				WithPolling(10 * time.Second).
				Should(BeTrue())
		})

		It("should deploy gateway-api-bundle", func() {
			clusterName := state.GetCluster().Name
			namespace := state.GetCluster().Organization.GetNamespace()

			gatewayAPIOCIRepoName = fmt.Sprintf("%s-gateway-api-bundle", clusterName)
			err := helmrelease.EnsureOCIRepository(state.GetContext(), state.GetFramework().MC(), gatewayAPIOCIRepoName, namespace, "gateway-api-bundle")
			Expect(err).To(BeNil())

			hrBuilder, err := helmrelease.New(
				fmt.Sprintf("%s-gateway-api-bundle", clusterName),
				"gateway-api-bundle",
			).
				WithNamespace(namespace).
				WithClusterName(clusterName).
				WithInCluster(true).
				WithTargetNamespace(namespace).
				WithServiceAccountName("automation").
				WithValuesFile("./test_data/gateway-api-bundle_values.yaml", &helmrelease.TemplateValues{
					ClusterName: clusterName,
				})
			Expect(err).To(BeNil())
			gatewayAPIHelmRelease, err = hrBuilder.Build()
			Expect(err).To(BeNil())

			err = state.GetFramework().MC().Create(state.GetContext(), gatewayAPIHelmRelease)
			Expect(err).To(BeNil())

			Eventually(helmrelease.IsHelmReleaseReady(state.GetContext(), state.GetFramework().MC(), gatewayAPIHelmRelease.GetName(), gatewayAPIHelmRelease.GetNamespace())).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(BeTrue())

			childHelmReleases := []types.NamespacedName{
				{Name: fmt.Sprintf("%s-gateway-api-crds", clusterName), Namespace: namespace},
				{Name: fmt.Sprintf("%s-envoy-gateway", clusterName), Namespace: namespace},
				{Name: fmt.Sprintf("%s-gateway-api-config", clusterName), Namespace: namespace},
			}
			Eventually(helmrelease.AreAllReady(state.GetContext(), state.GetFramework().MC(), childHelmReleases)).
				WithTimeout(10*time.Minute).
				WithPolling(10*time.Second).
				Should(
					Succeed(),
					failurehandler.HelmReleasesNotReady(state.GetFramework(), state.GetCluster()),
				)
		})

		It("gateway giantswarm-default should be programmed", func() {
			const gatewayName = "giantswarm-default"
			const gatewayNamespace = "envoy-gateway-system"

			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).ShouldNot(HaveOccurred())

			// Return the reason as error, so that Gomega prints the last one on timeout
			Eventually(func() (bool, error) {
				gateway := &unstructured.Unstructured{}
				gateway.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "gateway.networking.k8s.io",
					Version: "v1",
					Kind:    "Gateway",
				})
				err := wcClient.Get(state.GetContext(), types.NamespacedName{Name: gatewayName, Namespace: gatewayNamespace}, gateway)
				if err != nil {
					logger.Log("Failed to get Gateway: %v", err)
					return false, err
				}

				conditions, found, err := unstructured.NestedSlice(gateway.Object, "status", "conditions")
				if err != nil || !found || len(conditions) == 0 {
					logger.Log("Gateway '%s' has no status conditions yet", gatewayName)
					return false, fmt.Errorf("gateway %q has no status conditions yet", gatewayName)
				}

				for _, c := range conditions {
					condition, ok := c.(map[string]interface{})
					if !ok {
						continue
					}
					condType, _ := condition["type"].(string)
					condStatus, _ := condition["status"].(string)
					if condType == "Programmed" && condStatus == "True" {
						return true, nil
					}
				}

				statusDescription := describeGatewayStatus(gateway)
				logger.Log("Gateway '%s' is not yet Programmed: %s", gatewayName, statusDescription)
				return false, fmt.Errorf("gateway %q is not Programmed: %s", gatewayName, statusDescription)
			}).
				WithTimeout(10*time.Minute).
				WithPolling(10*time.Second).
				Should(BeTrue(), gatewayNotProgrammedFailureHandler(gatewayNamespace))
		})

		// FlakeAttempts: depends on external/split-horizon DNS propagation.
		It("cluster wildcard DNS must be resolvable", FlakeAttempts(3), func() {
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
				WithTimeout(10*time.Minute).
				WithPolling(10*time.Second).
				Should(BeTrue(), failurehandler.ExternalDNSIssues(state.GetFramework(), state.GetCluster()))
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
					failurehandler.CertificatesNotReady(state.GetFramework(), state.GetCluster(), "envoy-gateway-system"),
				)
		})

		It("should deploy hello-world app with HTTPRoute", func() {
			org := state.GetCluster().Organization
			clusterName := state.GetCluster().Name
			namespace := org.GetNamespace()
			helloWorldHost = fmt.Sprintf("hello-world.%s", getWorkloadClusterDnsZone())
			helloWorldUrl = fmt.Sprintf("https://%s", helloWorldHost)

			ociRepoName = fmt.Sprintf("%s-hello-world-chart", clusterName)
			err := helmrelease.EnsureOCIRepository(state.GetContext(), state.GetFramework().MC(), ociRepoName, namespace, "hello-world")
			Expect(err).To(BeNil())

			hrBuilder, err := helmrelease.New(
				fmt.Sprintf("%s-hello-world-gateway", clusterName),
				"hello-world",
			).
				WithNamespace(namespace).
				WithReleaseName("hello-world").
				WithTargetNamespace("giantswarm").
				WithOCIRepoName(ociRepoName).
				WithClusterName(clusterName).
				WithValuesFile("./test_data/helloworld_route_values.yaml", &helmrelease.TemplateValues{
					ClusterName: clusterName,
					ExtraValues: map[string]string{"IngressUrl": helloWorldHost},
				})
			Expect(err).To(BeNil())
			helloHelmRelease, err = hrBuilder.Build()
			Expect(err).To(BeNil())

			err = state.GetFramework().MC().Create(state.GetContext(), helloHelmRelease)
			Expect(err).To(BeNil())

			Eventually(helmrelease.IsHelmReleaseReady(state.GetContext(), state.GetFramework().MC(), helloHelmRelease.GetName(), helloHelmRelease.GetNamespace())).
				WithTimeout(6 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		It("HTTPRoute should be accepted by gateway", func() {
			wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func() (bool, error) {
				httpRoute := &unstructured.Unstructured{}
				httpRoute.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "gateway.networking.k8s.io",
					Version: "v1",
					Kind:    "HTTPRoute",
				})
				err := wcClient.Get(state.GetContext(), types.NamespacedName{Name: "hello-world", Namespace: "giantswarm"}, httpRoute)
				if err != nil {
					logger.Log("Failed to get HTTPRoute: %v", err)
					return false, err
				}

				parents, found, err := unstructured.NestedSlice(httpRoute.Object, "status", "parents")
				if err != nil || !found || len(parents) == 0 {
					logger.Log("HTTPRoute has no parent status yet")
					return false, nil
				}

				accepted := false
				resolvedRefs := false
				if parent, ok := parents[0].(map[string]interface{}); ok {
					conditions, _, _ := unstructured.NestedSlice(parent, "conditions")
					for _, c := range conditions {
						condition, ok := c.(map[string]interface{})
						if !ok {
							continue
						}
						condType, _ := condition["type"].(string)
						condStatus, _ := condition["status"].(string)
						if condType == "Accepted" && condStatus == "True" {
							accepted = true
						}
						if condType == "ResolvedRefs" && condStatus == "True" {
							resolvedRefs = true
						}
					}
				}

				if !accepted || !resolvedRefs {
					logger.Log("HTTPRoute not yet accepted: accepted=%v resolvedRefs=%v", accepted, resolvedRefs)
					return false, nil
				}

				return true, nil
			}).
				WithTimeout(6 * time.Minute).
				WithPolling(5 * time.Second).
				Should(BeTrue())
		})

		// FlakeAttempts: performs a live HTTPS request through external DNS and
		// a cloud load balancer, both inherently transient.
		It("hello world app responds successfully", FlakeAttempts(3), func() {
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
				WithTimeout(15 * time.Minute).
				WithPolling(5 * time.Second).
				Should(
					ContainSubstring("Hello World"),
				)
		})

		It("uninstall apps", func() {
			if helloHelmRelease != nil {
				err := state.GetFramework().MC().Delete(state.GetContext(), helloHelmRelease)
				Expect(err).ShouldNot(HaveOccurred())

				err = helmrelease.DeleteOCIRepository(state.GetContext(), state.GetFramework().MC(), ociRepoName, helloHelmRelease.GetNamespace())
				Expect(err).ShouldNot(HaveOccurred())
			}
			if gatewayAPIHelmRelease != nil {
				err := state.GetFramework().MC().Delete(state.GetContext(), gatewayAPIHelmRelease)
				Expect(err).ShouldNot(HaveOccurred())

				err = helmrelease.DeleteOCIRepository(state.GetContext(), state.GetFramework().MC(), gatewayAPIOCIRepoName, gatewayAPIHelmRelease.GetNamespace())
				Expect(err).ShouldNot(HaveOccurred())
			}
			if awsLBHelmRelease != nil {
				err := state.GetFramework().MC().Delete(state.GetContext(), awsLBHelmRelease)
				Expect(err).ShouldNot(HaveOccurred())

				err = helmrelease.DeleteOCIRepository(state.GetContext(), state.GetFramework().MC(), awsLBOCIRepoName, awsLBHelmRelease.GetNamespace())
				Expect(err).ShouldNot(HaveOccurred())
			}
		})
	})
}

func describeCondition(condition interface{}) (description string, isTrue bool) {
	conditionMap, ok := condition.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("unparsable condition %v", condition), false
	}
	conditionType, _ := conditionMap["type"].(string)
	conditionStatus, _ := conditionMap["status"].(string)
	conditionReason, _ := conditionMap["reason"].(string)
	conditionMessage, _ := conditionMap["message"].(string)
	return fmt.Sprintf("%s=%s (%s): %s", conditionType, conditionStatus, conditionReason, conditionMessage), conditionStatus == "True"
}

// describeGatewayStatus lists the Gateway and listener conditions that are not True
func describeGatewayStatus(gateway *unstructured.Unstructured) string {
	var ret []string

	conditions, _, _ := unstructured.NestedSlice(gateway.Object, "status", "conditions")
	hasProgrammedCondition := false
	for _, condition := range conditions {
		if conditionMap, ok := condition.(map[string]interface{}); ok && conditionMap["type"] == "Programmed" {
			hasProgrammedCondition = true
		}
		description, isTrue := describeCondition(condition)
		if !isTrue {
			ret = append(ret, "gateway "+description)
		}
	}
	if !hasProgrammedCondition {
		ret = append(ret, "gateway has no Programmed condition")
	}

	listeners, _, _ := unstructured.NestedSlice(gateway.Object, "status", "listeners")
	for _, listener := range listeners {
		listenerMap, ok := listener.(map[string]interface{})
		if !ok {
			continue
		}
		listenerName, _ := listenerMap["name"].(string)
		listenerConditions, _ := listenerMap["conditions"].([]interface{})
		for _, condition := range listenerConditions {
			description, isTrue := describeCondition(condition)
			if !isTrue {
				ret = append(ret, fmt.Sprintf("listener %q %s", listenerName, description))
			}
		}
	}

	return strings.Join(ret, "; ")
}

// gatewayNotProgrammedFailureHandler logs the objects that usually explain why a Gateway is not
// Programmed: the Envoy Service without a load balancer address, unavailable Envoy replicas, or a
// GatewayClass that was not accepted
func gatewayNotProgrammedFailureHandler(gatewayNamespace string) failurehandler.FailureHandler {
	return failurehandler.Wrap(func() {
		ctx := state.GetContext()

		wcClient, err := state.GetFramework().WC(state.GetCluster().Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		services := &corev1.ServiceList{}
		err = wcClient.List(ctx, services, ctrl.InNamespace(gatewayNamespace))
		if err != nil {
			logger.Log("Failed to list Services in namespace '%s' - %v", gatewayNamespace, err)
		}
		for i := range services.Items {
			service := &services.Items[i]
			logger.Log("Service '%s': Type='%s', LoadBalancerClass='%s', Annotations='%v', LoadBalancerIngress='%v'",
				service.Name, service.Spec.Type, ptr.Deref(service.Spec.LoadBalancerClass, ""), service.Annotations, service.Status.LoadBalancer.Ingress)

			events, err := wcClient.GetEventsForResource(ctx, service)
			if err != nil {
				logger.Log("Failed to get Events for Service '%s' - %v", service.Name, err)
				continue
			}
			for _, event := range events.Items {
				logger.Log("Service '%s' Event: Type='%s', Reason='%s', Message='%s'", service.Name, event.Type, event.Reason, event.Message)
			}
		}

		deployments := &appsv1.DeploymentList{}
		err = wcClient.List(ctx, deployments, ctrl.InNamespace(gatewayNamespace))
		if err != nil {
			logger.Log("Failed to list Deployments in namespace '%s' - %v", gatewayNamespace, err)
		}
		for _, deployment := range deployments.Items {
			logger.Log("Deployment '%s': ReadyReplicas='%d', Replicas='%d'", deployment.Name, deployment.Status.ReadyReplicas, deployment.Status.Replicas)
		}

		gatewayClasses := &unstructured.UnstructuredList{}
		gatewayClasses.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "gateway.networking.k8s.io",
			Version: "v1",
			Kind:    "GatewayClassList",
		})
		err = wcClient.List(ctx, gatewayClasses)
		if err != nil {
			logger.Log("Failed to list GatewayClasses - %v", err)
		}
		for _, gatewayClass := range gatewayClasses.Items {
			conditions, _, _ := unstructured.NestedSlice(gatewayClass.Object, "status", "conditions")
			var descriptions []string
			for _, condition := range conditions {
				description, _ := describeCondition(condition)
				descriptions = append(descriptions, description)
			}
			logger.Log("GatewayClass '%s' conditions: %s", gatewayClass.GetName(), strings.Join(descriptions, "; "))
		}
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
