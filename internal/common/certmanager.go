package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var clusterIssuers = []string{"selfsigned-giantswarm", "letsencrypt-giantswarm"}

func runCertManager() {
	Context("cert-manager ClusterIssuers", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("cert-manager default ClusterIssuers are present and ready", func() {
			for _, clusterIssuerName := range clusterIssuers {
				Eventually(checkClusterIssuer(wcClient, clusterIssuerName)).
					WithTimeout(15 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			}
		})
	})
}

func checkClusterIssuer(wcClient *client.Client, clusterIssuerName string) func() error {
	return func() error {
		logger.Log("Checking ClusterIssuer '%s'", clusterIssuerName)
		// Using a unstructured object.
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "cert-manager.io",
			Kind:    "ClusterIssuer",
			Version: "v1",
		})
		err := wcClient.Get(context.Background(), cr.ObjectKey{
			Name: clusterIssuerName,
		}, u)
		if err != nil {
			return err
		}
		logger.Log("ClusterIssuer '%s' is present", clusterIssuerName)

		conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("ClusterIssuer '%s' does not have status.conditions", clusterIssuerName)
		}

		for _, condition := range conditions {
			c := condition.(map[string]interface{})
			conditionType := c["type"]
			status := c["status"]
			if conditionType == "Ready" && status == "True" {
				logger.Log("Found status.condition with type '%s' and status '%s' in ClusterIssuer '%s'", conditionType, status, clusterIssuerName)
				return nil
			}
		}

		logger.Log("ClusterIssuer '%s' is not Ready", clusterIssuerName)
		return fmt.Errorf("ClusterIssuer '%s' is not Ready", clusterIssuerName)
	}
}
