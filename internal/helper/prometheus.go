package helper

import (
	"context"
	"fmt"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

const (
	mimirNamespace      = "mimir"
	mimirGatewayService = "mimir-gateway"
)

// DetectPrometheusBaseURL checks if the MC is running old Prometheus setup or mimir and returns the base URL to send PROMQL queries to.
func DetectPrometheusBaseURL(ctx context.Context, mcClient *client.Client) (string, error) {
	// Check if there is a Service named 'mimir-gateway' in the namespace 'mimir'.
	svc := &corev1.Service{}

	err := mcClient.Get(ctx, ctrlclient.ObjectKey{Namespace: mimirNamespace, Name: mimirGatewayService}, svc)
	if errors.IsNotFound(err) {
		logger.Log("Legacy prometheus setup detected")
		// Mimir service not found, we assume it's the old Prometheus setup.
		return fmt.Sprintf("prometheus-operated.%[1]s-prometheus:9090/%[1]s", state.GetCluster().Name), nil
	} else if err != nil {
		// Generic error.
		return "", err
	}

	logger.Log("Mimir detected")

	// Mimir service found, we use it.
	return fmt.Sprintf("%s.%s:%d/prometheus", mimirGatewayService, mimirNamespace, svc.Spec.Ports[0].Port), nil
}
