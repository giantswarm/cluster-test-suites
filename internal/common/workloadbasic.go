package common

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func runWorkloadBasic() {
	Context("workload basic", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("has all its Deployments Ready (means all replicas are running)", func() {
			Eventually(wait.Consistent(checkAllDeploymentsReady(wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its StatefulSets Ready (means all replicas are running)", func() {
			Eventually(wait.Consistent(checkAllStatefulSetsReady(wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all its DaemonSets Ready (means all daemon pods are running)", func() {
			Eventually(wait.Consistent(checkAllDaemonSetsReady(wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})

		It("has all of its Pods in the Running state", func() {
			Eventually(wait.Consistent(checkAllPodsSuccessfulPhase(wcClient), 10, time.Second)).
				WithTimeout(15 * time.Minute).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
}

func checkAllPodsSuccessfulPhase(wcClient *client.Client) func() error {
	return func() error {
		podList := &corev1.PodList{}
		err := wcClient.List(context.Background(), podList)
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				logger.Log("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
				return fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		logger.Log("All (%d) pods currently in a running or completed state", len(podList.Items))
		return nil
	}
}

func checkAllDeploymentsReady(wcClient *client.Client) func() error {
	return func() error {
		deploymentList := &appsv1.DeploymentList{}
		err := wcClient.List(context.Background(), deploymentList)
		if err != nil {
			return err
		}

		for _, deployment := range deploymentList.Items {
			available := deployment.Status.AvailableReplicas
			desired := *deployment.Spec.Replicas
			if available != desired {
				logger.Log("deployment %s/%s has %d/%d replicas available", deployment.Namespace, deployment.Name, available, desired)
				return fmt.Errorf("deployment %s/%s has %d/%d replicas available", deployment.Namespace, deployment.Name, available, desired)
			}
		}

		logger.Log("All (%d) deployments have all replicas running", len(deploymentList.Items))
		return nil
	}
}

func checkAllStatefulSetsReady(wcClient *client.Client) func() error {
	return func() error {
		statefulSetList := &appsv1.StatefulSetList{}
		err := wcClient.List(context.Background(), statefulSetList)
		if err != nil {
			return err
		}

		for _, statefulSet := range statefulSetList.Items {
			available := statefulSet.Status.AvailableReplicas
			desired := *statefulSet.Spec.Replicas
			if available != desired {
				logger.Log("statefulSet %s/%s has %d/%d replicas available", statefulSet.Namespace, statefulSet.Name, available, desired)
				return fmt.Errorf("statefulSet %s/%s has %d/%d replicas available", statefulSet.Namespace, statefulSet.Name, available, desired)
			}
		}

		logger.Log("All (%d) statefulSets have all replicas running", len(statefulSetList.Items))
		return nil
	}
}

func checkAllDaemonSetsReady(wcClient *client.Client) func() error {
	return func() error {
		daemonSetList := &appsv1.DaemonSetList{}
		err := wcClient.List(context.Background(), daemonSetList)
		if err != nil {
			return err
		}

		for _, daemonSet := range daemonSetList.Items {
			current := daemonSet.Status.CurrentNumberScheduled
			desired := daemonSet.Status.DesiredNumberScheduled
			if current != desired {
				logger.Log("daemonSet %s/%s has %d/%d daemon pods available", daemonSet.Namespace, daemonSet.Name, current, desired)
				return fmt.Errorf("daemonSet %s/%s has %d/%d daemon pods available", daemonSet.Namespace, daemonSet.Name, current, desired)
			}
		}

		logger.Log("All (%d) daemonSets have all daemon pods running", len(daemonSetList.Items))
		return nil
	}
}
