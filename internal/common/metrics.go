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

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func runMetrics() {
	Context("metrics", func() {
		var mcClient *client.Client
		var metrics []string

		BeforeEach(func() {
			mcClient = state.GetFramework().MC()

			// List of metrics that must be present.
			metrics = []string{
				// API server metrics in prometheus-rules
				"apiserver_flowcontrol_dispatched_requests_total",
				"apiserver_flowcontrol_request_concurrency_limit",
				"apiserver_request_duration_seconds_bucket",
				"apiserver_admission_webhook_request_total",
				"apiserver_admission_webhook_admission_duration_seconds_sum",
				"apiserver_admission_webhook_admission_duration_seconds_count",
				"apiserver_request_total",
				"apiserver_audit_event_total",

				// Kubelet
				"kube_node_status_condition",
				"kube_node_spec_unschedulable",
				"kube_node_created",

				// Controller manager
				"workqueue_queue_duration_seconds_count",
				"workqueue_queue_duration_seconds_bucket",

				// Scheduler
				"scheduler_pod_scheduling_duration_seconds_count",
				"scheduler_pod_scheduling_duration_seconds_bucket",

				// ETCD
				"etcd_request_duration_seconds_count",
				"etcd_request_duration_seconds_bucket",

				// Coredns
				"coredns_dns_request_duration_seconds_count",
				"coredns_dns_request_duration_seconds_bucket",

				// Net exporter
				"network_latency_seconds_sum",
			}
		})

		It("ensure key metrics are available on prometheus", func() {
			namespace := fmt.Sprintf("%s-prometheus", state.GetCluster().Name)
			podName := fmt.Sprintf("prometheus-%s-0", state.GetCluster().Name)

			for _, metric := range metrics {
				Eventually(checkMetricPresent(mcClient, namespace, podName, metric)).
					WithTimeout(10 * time.Minute).
					WithPolling(10 * time.Second).
					Should(Succeed())
			}
		})
	})
}

func checkMetricPresent(mcClient *client.Client, namespace string, podName string, metric string) func() error {
	return func() error {
		query := fmt.Sprintf("absent(%s) or vector(0)", metric)
		cmd := []string{"wget", "-q", "-O-", "-Y", "off", fmt.Sprintf("prometheus-operated.%s-prometheus:9090/%s/api/v1/query?query=%s", state.GetCluster().Name, state.GetCluster().Name, url.QueryEscape(query))}
		stdout, _, err := mcClient.ExecInPod(context.Background(), podName, namespace, "prometheus", cmd)
		if err != nil {
			return fmt.Errorf("can't exec command in pod %s: %s", podName, err)
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
