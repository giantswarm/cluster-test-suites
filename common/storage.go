package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	pvcName    = "pvc-test"
	pvcPodName = "pvc-test-pod"
)

func runStorage() {
	Context("basic", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = Framework.WC(Cluster.Name)
			if err != nil {
				Fail(err.Error())
			}
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

		It("has deleted a pod with a pvc", func() {
			Eventually(wait.Consistent(deletePodWithPVC(wcClient), 10, time.Second)).
				WithTimeout(wait.DefaultTimeout).
				WithPolling(wait.DefaultInterval).
				Should(Succeed())
		})
	})
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
				Name: "test-pvc",
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
			return err
		}

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pvc-test-pod",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "pvc-test-container",
						Image: "nginx",
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "test-volume",
								MountPath: "/data",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "test-volume",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
			},
		}

		err = wcClient.Create(context.Background(), pod)
		if err != nil {
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
