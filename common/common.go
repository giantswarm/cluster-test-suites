package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type TestConfig struct {
	BastionSupported bool
}

var (
	Framework *clustertest.Framework
	Cluster   *application.Cluster

	pvcName    = "pvc-test"
	pvcPodName = "pvc-test-pod"
)

func Run() {
	var wcClient *client.Client

	BeforeEach(func() {
		var err error

		wcClient, err = Framework.WC(Cluster.Name)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be able to connect to MC cluster", func() {
		Expect(Framework.MC().CheckConnection()).To(Succeed())
	})

	It("should be able to connect to WC cluster", func() {
		Expect(wcClient.CheckConnection()).To(Succeed())
	})

	It("has a at least one storage class available", func() {
		Eventually(wait.Consistent(checkStorageClassExists(wcClient), 10, time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("has created a pod with a pvc and the pvc is bound", func() {
		Eventually(wait.Consistent(createPodWithPVC(wcClient), 10, time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("has all of it's Pods in the Running state", func() {
		Eventually(wait.Consistent(checkAllPodsSuccessfulPhase(wcClient), 10, time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("has deleted a pod with a pvc", func() {
		Eventually(wait.Consistent(deletePodWithPVC(wcClient), 10, time.Second)).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})
}

func CheckControlPlaneNodesReady(wcClient *client.Client, values application.ControlPlane) func() error {
	expectedNodes := values.Replicas
	controlPlaneFunc := wait.AreNumNodesReady(context.Background(), wcClient, expectedNodes, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""})

	return func() error {
		ok, err := controlPlaneFunc()
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return err
	}
}

func CheckWorkerNodesReady(wcClient *client.Client, values *application.ClusterValues) func() error {
	minNodes := 0
	maxNodes := 0
	for _, pool := range values.NodePools {
		if pool.Replicas > 0 {
			minNodes += pool.Replicas
			maxNodes += pool.Replicas
			continue
		}

		minNodes += pool.MinSize
		maxNodes += pool.MaxSize
	}
	expectedNodes := wait.Range{
		Min: minNodes,
		Max: maxNodes,
	}

	workersFunc := wait.AreNumNodesReadyWithinRange(context.Background(), wcClient, expectedNodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})

	return func() error {
		ok, err := workersFunc()
		if err != nil {
			logger.Log("failed to get nodes: %s", err)
			return err
		}
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return nil
	}
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
				return fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		return nil
	}
}

func checkStorageClassExists(wcClient *client.Client) func() error {
	return func() error {
		// ensure we have at least one storage class available
		storageClasses := &storagev1.StorageClassList{}
		err := wcClient.List(context.Background(), storageClasses)
		if err != nil {
			return err
		}
		if len(storageClasses.Items) == 0 {
			return fmt.Errorf("no storage classes found")
		}

		return nil
	}
}

func createPodWithPVC(wcClient *client.Client) func() error {
	return func() error {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: corev1.NamespaceDefault,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		}

		err := wcClient.Create(context.Background(), pvc)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				return nil
			}
			return err
		}

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcPodName,
				Namespace: corev1.NamespaceDefault,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  pvcName,
						Image: "nginx",
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      pvcName,
								MountPath: "/data",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: pvcName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
			},
		}

		err = wcClient.Create(context.Background(), pod)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				return nil
			}
			// if pod creation fails, delete the PVC to avoid leaving a dangling PVC
			if deleteErr := wcClient.Delete(context.Background(), pvc); deleteErr != nil {
				return fmt.Errorf("failed to delete PVC after Pod creation failed: %v", deleteErr)
			}
			return err
		}

		return nil
	}
}

func deletePodWithPVC(wcClient *client.Client) func() error {
	return func() error {

		pod := &corev1.Pod{}
		err := wcClient.Get(context.Background(), types.NamespacedName{Name: pvcPodName, Namespace: corev1.NamespaceDefault}, pod)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		err = wcClient.Delete(context.Background(), pod)
		if err != nil {
			return err
		}

		pvc := &corev1.PersistentVolumeClaim{}
		err = wcClient.Get(context.Background(), types.NamespacedName{Name: pvcName, Namespace: corev1.NamespaceDefault}, pvc)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// If the PVC doesn't exist, return nil
				return nil
			}
			return err
		}

		err = wcClient.Delete(context.Background(), pvc)
		if err != nil {
			return err
		}

		return nil
	}
}
