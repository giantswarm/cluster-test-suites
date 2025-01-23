package ecr

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/failurehandler"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	appsv1 "k8s.io/api/apps/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/cluster-test-suites/internal/helper"
	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func Run() {

	/*
		Note: These tests use a pre-created private ECR repository - 992382781567.dkr.ecr.eu-west-2.amazonaws.com/giantswarm/alpine
		This repository exists in an account controlled by us and has permissions set to allow any AWS account within the
		Giant Swarm AWS Organisation to be able to pull from it (based on `aws:ResourceOrgID`).

		The image being used is the latest (at time of creation) upstream `alpine` image with no changes.

		The test only uses it to check that the image can be pulled without hitting an unauthorized error.
	*/

	Context("ecr credential provider", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			helper.SetResponsibleTeam(helper.TeamPhoenix)

			var err error
			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("should be able to pull an image from a private ECR registry", func() {
			deploymentObj, err := helper.Deserialize(deploymentManifest)
			Expect(err).ToNot(HaveOccurred())
			deployment := deploymentObj.(*appsv1.Deployment)

			Eventually(func() error {
				logger.Log("Creating deployment with private ECR image...")
				err = wcClient.Create(state.GetContext(), deployment)
				if err != nil && !apierror.IsAlreadyExists(err) {
					return err
				}

				return nil
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())

			Eventually(func() error {
				logger.Log("Checking status of deployments replicas...")
				err := wcClient.Get(context.Background(), cr.ObjectKey{Name: deployment.ObjectMeta.Name, Namespace: deployment.ObjectMeta.Namespace}, deployment)
				if err != nil {
					return err
				}

				if deployment.Status.ReadyReplicas != deployment.Status.Replicas {
					logger.Log("Deployment isn't yet ready %d/%d pod replicas ready", deployment.Status.ReadyReplicas, deployment.Status.Replicas)
					return fmt.Errorf("deployment %s in namespace %s doesn't have all replicas ready", deployment.ObjectMeta.Name, deployment.ObjectMeta.Namespace)
				}

				if deployment.Status.Replicas == 0 {
					logger.Log("Deployment isn't yet ready - it currently has no replicas")
					return fmt.Errorf("deployment %s in namespace %s has no replicas", deployment.ObjectMeta.Name, deployment.ObjectMeta.Namespace)
				}

				logger.Log("Deployment has %d ready pod replicas", deployment.Status.ReadyReplicas)

				return nil
			}).
				WithTimeout(2*time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed(),
					failurehandler.DeploymentsNotReady(state.GetFramework(), state.GetCluster()))

			Eventually(func() error {
				logger.Log("Deleting deployment...")
				return wcClient.Delete(state.GetContext(), deployment)
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
}
