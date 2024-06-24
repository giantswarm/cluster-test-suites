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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"

	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				Eventually(checkClusterIssuer(state.GetContext(), wcClient, clusterIssuerName)).
					WithTimeout(120 * time.Second).
					WithPolling(1 * time.Second).
					Should(Succeed())
			}
		})
	})
}

func checkClusterIssuer(ctx context.Context, wcClient *client.Client, clusterIssuerName string) func() error {
	return func() error {
		logger.Log("Checking ClusterIssuer '%s'", clusterIssuerName)
		// Using a unstructured object.
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "cert-manager.io",
			Kind:    "ClusterIssuer",
			Version: "v1",
		})
		err := wcClient.Get(ctx, cr.ObjectKey{Name: clusterIssuerName}, u)
		if err != nil {
			if errors.IsNotFound(err) {
				// Cluster Issuer was not found so we'll check the status of the Job that creates it
				var nestedErr error
				logger.Log("ClusterIssuer '%s' is not yet found", clusterIssuerName)

				// Get status of cluster issuer post-install Job
				clusterIssuerJob := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-manager-giantswarm-clusterissuer",
						Namespace: "kube-system",
					},
				}
				nestedErr = wcClient.Get(ctx, cr.ObjectKeyFromObject(clusterIssuerJob), clusterIssuerJob)
				if nestedErr != nil {
					logger.Log("Failed to get cluster issuer Job, it may have already completed: %v", nestedErr)
				} else {
					logger.Log("Status of cluster issuer Job '%s': Succeeded:%t", clusterIssuerJob.ObjectMeta.Name, clusterIssuerJob.Status.Succeeded > 0)
				}

				// Get events related to the cluster issuer post-install Job
				events := &corev1.EventList{}
				nestedErr = wcClient.List(ctx, events, cr.MatchingFieldsSelector{
					Selector: fields.AndSelectors(
						fields.OneTermEqualSelector("involvedObject.kind", "Job"),
						fields.OneTermEqualSelector("involvedObject.name", clusterIssuerJob.ObjectMeta.Name),
					),
				})
				if nestedErr != nil {
					logger.Log("Failed to get events for cluster issuer Job: %v", nestedErr)
				} else {
					for _, event := range events.Items {
						if event.Type != corev1.EventTypeNormal {
							logger.Log("Event: Reason='%s', Message='%s', Last Occurred='%v'", event.Reason, event.Message, event.LastTimestamp)
						}
					}
				}
			}

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
