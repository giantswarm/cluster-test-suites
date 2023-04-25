package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	Framework *clustertest.Framework
	Cluster   *application.Cluster
)

func Run() {
	var wcClient *client.Client

	BeforeEach(func() {
		var err error

		wcClient, err = Framework.WC(Cluster.Name)
		if err != nil {
			Fail(err.Error())
		}
	})

	It("should be able to connect to MC cluster", func() {
		Expect(Framework.MC().CheckConnection()).To(Succeed())
	})

	It("should be able to connect to WC cluster", func() {
		Expect(wcClient.CheckConnection()).To(Succeed())
	})

	It("has all of it's pods in the Running state", func() {
		Eventually(Consistent(checkPodSuccessfulPhase(wcClient), 10, time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})
}

func Consistent(action func() error, attempts int, pollInterval time.Duration) func() error {
	return func() error {
		ticker := time.NewTicker(pollInterval)
		for range ticker.C {
			if attempts <= 0 {
				ticker.Stop()
				break
			}

			err := action()
			if err != nil {
				return err
			}

			attempts--
		}

		return nil
	}
}

func checkPodSuccessfulPhase(wcClient *client.Client) func() error {
	return func() error {
		podList := &corev1.PodList{}
		err := wcClient.List(context.Background(), podList)
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				return fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		return nil
	}
}
