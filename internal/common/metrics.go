package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/helper"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runMetrics(controlPlaneMetricsSupported bool) {
	Context("metrics", func() {
		var mcClient *client.Client
		var metrics []string
		var testPodName string
		var testPodNamespace string
		var prometheusBaseUrl string

		BeforeEach(func() {
			mcClient = state.GetFramework().MC()

			// List of metrics that must be present.
			metrics = []string{
				// Kubelet
				"kube_node_status_condition",
				"kube_node_spec_unschedulable",
				"kube_node_created",

				// Coredns
				"coredns_dns_request_duration_seconds_count",
				"coredns_dns_request_duration_seconds_bucket",

				// Net exporter
				"network_latency_seconds_sum",
			}

			if controlPlaneMetricsSupported {
				metrics = append(metrics, []string{
					// API server metrics in prometheus-rules
					"apiserver_flowcontrol_dispatched_requests_total",
					"apiserver_flowcontrol_request_concurrency_limit",
					"apiserver_request_duration_seconds_bucket",
					"apiserver_admission_webhook_request_total",
					"apiserver_admission_webhook_admission_duration_seconds_sum",
					"apiserver_admission_webhook_admission_duration_seconds_count",
					"apiserver_request_total",
					"apiserver_audit_event_total",

					// Controller manager
					"workqueue_queue_duration_seconds_count",
					"workqueue_queue_duration_seconds_bucket",

					// Scheduler
					"scheduler_pod_scheduling_duration_seconds_count",
					"scheduler_pod_scheduling_duration_seconds_bucket",

					// ETCD
					"etcd_request_duration_seconds_count",
					"etcd_request_duration_seconds_bucket",
				}...)
			}
		})

		It("creates test pod", func() {
			// Run a pod with alpine in the default namespace of the MC.
			testPodName = fmt.Sprintf("%s-metrics-test", state.GetCluster().Name)
			testPodNamespace = "default"

			err := runTestPod(mcClient, testPodName, testPodNamespace)
			Expect(err).NotTo(HaveOccurred())

			prometheusBaseUrl, err = helper.DetectPrometheusBaseURL(context.TODO(), mcClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("ensure key metrics are available on prometheus", func() {
			for _, metric := range metrics {
				Eventually(checkMetricPresent(mcClient, metric, prometheusBaseUrl, testPodName, testPodNamespace)).
					WithTimeout(10 * time.Minute).
					WithPolling(10 * time.Second).
					Should(Succeed())
			}
		})

		It("clean up test pod", func() {
			err := cleanupTestPod(mcClient, testPodName, testPodNamespace)
			Expect(err).NotTo(HaveOccurred())
		})
	})
}

func checkMetricPresent(mcClient *client.Client, metric string, prometheusBaseUrl string, testPodName string, testPodNamespace string) func() error {
	return func() error {
		query := fmt.Sprintf("absent(%[1]s{cluster_id=\"%[2]s\"}) or label_replace(vector(0), \"cluster_id\", \"%[2]s\", \"\", \"\")", metric, state.GetCluster().Name)

		cmd := []string{"wget", "-q", "-O-", "-Y", "off", fmt.Sprintf("%[1]s/api/v1/query?query=%[2]s", prometheusBaseUrl, url.QueryEscape(query))}
		stdout, _, err := mcClient.ExecInPod(context.Background(), testPodName, testPodNamespace, "test", cmd)
		if err != nil {
			return fmt.Errorf("can't exec command in pod %s: %s", testPodName, err)
		}

		// {"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1681718763.145,"1"]}]}}

		type result struct {
			Value []any
		}

		response := struct {
			Status string
			Data   struct {
				ResultType string
				Result     []result
			}
		}{}

		err = json.Unmarshal([]byte(stdout), &response)
		if err != nil {
			return fmt.Errorf("Can't parse prometheus query output: %s", err)
		}

		if response.Status != "success" {
			return fmt.Errorf("Unexpected response status %s when running query %q", response.Status, query)
		}

		if response.Data.ResultType != "vector" {
			return fmt.Errorf("Unexpected response type %s when running query %q (wanted vector)", response.Status, query)
		}

		if len(response.Data.Result) != 1 {
			return fmt.Errorf("Unexpected count of results when running query %q (wanted 1, got %d)", query, len(response.Data.Result))
		}

		// Second field of first result is the metric value. [1681718763.145,"1"] => "1"
		str, ok := (response.Data.Result[0].Value[1]).(string)
		if !ok {
			return fmt.Errorf("Cannot cast result value to string when running query %q", query)
		}
		if str != "0" {
			return fmt.Errorf("Unexpected value for query %q (wanted '0', got %q)", query, str)
		}

		logger.Log("Metric %q was found", metric)
		return nil
	}
}

func runTestPod(mcClient *client.Client, podName string, ns string) error {
	t := true
	f := false
	userAndGroup := int64(35)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:    &userAndGroup,
				RunAsGroup:   &userAndGroup,
				RunAsNonRoot: &t,
				SeccompProfile: &corev1.SeccompProfile{
					Type: "RuntimeDefault",
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "alpine:latest",
					Args:  []string{"sleep", "99999999"},
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								"ALL",
							},
						},
						AllowPrivilegeEscalation: &f,
					},
				},
			},
		},
	}
	// Check if pods exists already.
	create := false
	existing := corev1.Pod{}
	err := mcClient.Get(context.Background(), client2.ObjectKey{Namespace: ns, Name: podName}, &existing)
	if errors.IsNotFound(err) {
		create = true
	} else if err != nil {
		return fmt.Errorf("error ensuring test pod is deleted %s: %s", podName, err)
	}

	if !create {
		// Check if pod is running.
		if existing.Status.Phase != corev1.PodRunning {
			// Pod unhealthy, delete and recreate it.
			err := cleanupTestPod(mcClient, podName, ns)
			if err != nil {
				return err
			}

			create = true
		}
	}

	if create {
		// Create the pod.
		err = mcClient.Create(context.Background(), &pod)
		if err != nil {
			return fmt.Errorf("can't create test pod %s: %s", podName, err)
		}
	}

	// Wait for pod to be running.
	Eventually(func() (bool, error) {
		err = mcClient.Get(context.Background(), client2.ObjectKey{Namespace: ns, Name: podName}, &existing)
		if err != nil {
			return false, fmt.Errorf("error ensuring test pod is running %s: %s", podName, err)
		}

		if existing.Status.Phase == corev1.PodRunning {
			return true, nil
		}

		return false, fmt.Errorf("waiting for pod %s to be running", podName)
	}).
		WithTimeout(5 * time.Minute).
		WithPolling(5 * time.Second).
		Should(BeTrue())

	return nil
}

func cleanupTestPod(mcClient *client.Client, podName string, ns string) error {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
	}

	// Pod unhealthy, delete and recreate it.
	err := mcClient.Delete(context.Background(), &pod)
	if errors.IsNotFound(err) {
		// Fallthrough (in case the pod was deleting already).
	} else if err != nil {
		return fmt.Errorf("error deleting test pod %s: %s", podName, err)
	}

	// Wait for pod to be deleted.
	Eventually(func() (bool, error) {
		err = mcClient.Get(context.Background(), client2.ObjectKey{Namespace: ns, Name: podName}, &pod)
		if errors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, fmt.Errorf("error ensuring test pod %s is deleted: %s", podName, err)
		}

		return false, fmt.Errorf("waiting for pod %s to be deleted", podName)
	}).
		WithTimeout(5 * time.Minute).
		WithPolling(5 * time.Second).
		Should(BeTrue())

	return nil
}
