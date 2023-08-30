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

	cmapiutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
)

var clusterIssuers = []string{"letsencrypt-giantswarm", "selfsigned-giantswarm"}

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
				logger.Log("checking ClusterIssuer '%s'", clusterIssuerName)
				Eventually(checkClusterIssuer(wcClient, clusterIssuerName)).
					WithTimeout(30 * time.Second).
					WithPolling(50 * time.Millisecond).
					Should(Succeed())
			}
		})
	})
}

func checkClusterIssuer(wcClient *client.Client, clusterIssuerName string) error {
	clusterIssuer := &cmv1.ClusterIssuer{}
	err := wcClient.Get(context.Background(), cr.ObjectKey{Name: clusterIssuerName}, clusterIssuer)
	if err != nil {
		return err
	}

	if !cmapiutil.IssuerHasCondition(clusterIssuer, cmv1.IssuerCondition{
		Type:   cmv1.IssuerConditionReady,
		Status: cmmeta.ConditionTrue,
	}) {
		return fmt.Errorf("ClusterIssuer '%s' is not Ready", clusterIssuerName)
	}

	return nil
}
