package common

import (
	"context"
	"fmt"
	"time"

	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/giantswarm/clustertest/v2/pkg/client"
	"github.com/giantswarm/clustertest/v2/pkg/failurehandler"
	"github.com/giantswarm/clustertest/v2/pkg/logger"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	batchv1 "k8s.io/api/batch/v1"

	"github.com/giantswarm/cluster-test-suites/v2/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v2/internal/state"

	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var clusterIssuers = []string{"selfsigned-giantswarm", "letsencrypt-giantswarm"}

func runCertManager() {
	Context("cert-manager ClusterIssuers", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			helper.SetResponsibleTeam(helper.TeamShield)

			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("cert-manager default ClusterIssuers are present and ready", func() {
			for _, clusterIssuerName := range clusterIssuers {
				Eventually(checkClusterIssuer(state.GetContext(), wcClient, clusterIssuerName)).
					WithTimeout(120*time.Second).
					WithPolling(1*time.Second).
					Should(Succeed(), failurehandler.LLMPrompt(state.GetFramework(), state.GetCluster(), fmt.Sprintf("Investigate cert-manager ClusterIssuer %s missing or not ready", clusterIssuerName)))
			}
		})
	})
}

func checkClusterIssuer(ctx context.Context, wcClient *client.Client, clusterIssuerName string) func() error {
	return func() error {
		logger.Log("Checking ClusterIssuer '%s'", clusterIssuerName)

		clusterIssuer := &certmanager.ClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterIssuerName,
			},
		}
		err := wcClient.Get(ctx, cr.ObjectKeyFromObject(clusterIssuer), clusterIssuer)
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
					logger.Log("Status of cluster issuer Job '%s': Succeeded:%t", clusterIssuerJob.Name, clusterIssuerJob.Status.Succeeded > 0)
				}

				// Get events related to the cluster issuer post-install Job
				events, nestedErr := wcClient.GetWarningEventsForResource(ctx, clusterIssuerJob)
				if nestedErr != nil {
					logger.Log("Failed to get events for cluster issuer Job: %v", nestedErr)
				} else {
					for _, event := range events.Items {
						logger.Log("Event: Reason='%s', Message='%s', Last Occurred='%v'", event.Reason, event.Message, event.LastTimestamp)
					}
				}
			}

			return err
		}
		logger.Log("ClusterIssuer '%s' is present", clusterIssuerName)

		for _, condition := range clusterIssuer.Status.Conditions {
			if condition.Type == certmanager.IssuerConditionReady && condition.Status == "True" {
				logger.Log("Found status.condition with type '%s' and status '%s' in ClusterIssuer '%s'", condition.Type, condition.Status, clusterIssuerName)
				return nil
			}
		}

		logger.Log("ClusterIssuer '%s' is not Ready", clusterIssuerName)
		return fmt.Errorf("ClusterIssuer '%s' is not Ready", clusterIssuerName)
	}
}
