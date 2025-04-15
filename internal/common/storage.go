package common

import (
	"fmt"
	"time"

	"github.com/giantswarm/cluster-test-suites/assets/storage"
	"github.com/giantswarm/cluster-test-suites/internal/helper"
	"github.com/giantswarm/cluster-test-suites/internal/state"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

var (
	namespace = "test-storage"
)

func runStorage() {
	Context("storage", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			helper.SetResponsibleTeam(helper.TeamTenet)

			var err error

			wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		When("a pod uses a persistent volume claim", func() {
			var (
				pvc *corev1.PersistentVolumeClaim
			)

			It("has a at least one storage class available", func() {
				Eventually(wait.Consistent(checkStorageClassExists(wcClient), 10, time.Second)).
					WithTimeout(5 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			})

			It("creates the new namespace for the test", func() {
				Eventually(
					func() error {
						namespaceObj, err := helper.Deserialize(storage.Namespace)
						if err != nil {
							return err
						}
						namespace := namespaceObj.(*corev1.Namespace)
						logger.Log("Creating Namespace '%s'", namespace.ObjectMeta.Name)
						err = wcClient.Create(state.GetContext(), namespace)
						if err != nil && !apierror.IsAlreadyExists(err) {
							logger.Log("Failed to create Namespace '%s' - %v", namespace.ObjectMeta.Name, err)
							return err
						}
						logger.Log("Created Namespace '%s' successfully", namespace.ObjectMeta.Name)
						return nil
					}).
					WithTimeout(1 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			})

			It("creates the PVC", func() {
				Eventually(
					func() error {
						pvcObj, err := helper.Deserialize(storage.PVC)
						if err != nil {
							return err
						}
						pvc = pvcObj.(*corev1.PersistentVolumeClaim)
						logger.Log("Creating PersistentVolumeClaim")
						err = wcClient.Create(state.GetContext(), pvc)
						if err != nil && !apierror.IsAlreadyExists(err) {
							logger.Log("Failed to create PersistentVolumeClaim - %v", err)
							return err
						}

						logger.Log("PersistentVolumeClaim created")

						return nil
					}).
					WithTimeout(1 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			})

			It("creates the pod using the PVC", func() {
				if pvc == nil {
					Skip("PVC wasn't created")
					return
				}

				Eventually(
					func() error {
						podObj, err := helper.Deserialize(storage.Pod)
						if err != nil {
							return err
						}
						pod := podObj.(*corev1.Pod)
						logger.Log("Creating Pod '%s'", pod.ObjectMeta.Name)
						err = wcClient.Create(state.GetContext(), pod)
						if err != nil && !apierror.IsAlreadyExists(err) {
							logger.Log("Failed to create Pod '%s' - %v", pod.ObjectMeta.Name, err)
							return err
						}

						logger.Log("Created Pod '%s' succesfully", pod.ObjectMeta.Name)
						return nil
					}).
					WithTimeout(1 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			})

			It("binds the PVC", func() {
				Eventually(
					func() error {
						err := wcClient.Get(state.GetContext(), cr.ObjectKeyFromObject(pvc), pvc)
						if err != nil {
							logger.Log("Failed to get PersistentVolumeClaim - %v", err)
							return err
						}

						if pvc.Status.Phase != corev1.ClaimBound {
							logger.Log("PersistentVolumeClaim not yet bound to a volume")
							return fmt.Errorf("PVC not yet bound")
						}

						if pvc.Spec.VolumeName == "" {
							logger.Log("PersistentVolumeClaim doesn't yet have an associated PV volume name")
							return fmt.Errorf("no volume name available for PVC yet")
						}

						logger.Log("PersistentVolumeClaim created and has PV volume name '%s'", pvc.Spec.VolumeName)

						return nil
					}).
					WithTimeout(1 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			})

			It("runs successfully", func() {
				if pvc == nil {
					Skip("PVC wasn't created")
					return
				}
				Eventually(wait.Consistent(verifyPodState(wcClient, "pvc-test-pod", namespace), 10, time.Second)).
					WithTimeout(20 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			})

			It("deletes all resources correct", func() {
				Eventually(
					func() error {
						logger.Log("Deleting Namespace '%s'", namespace)
						err := wcClient.Delete(state.GetContext(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
						if err != nil && !apierror.IsNotFound(err) {
							logger.Log("Failed to delete Namespace '%s'", namespace)
							return err
						}

						if pvc != nil {
							pvName := pvc.Spec.VolumeName
							logger.Log("Deleting PersistentVolume '%s'", pvName)
							err = wcClient.Delete(state.GetContext(), &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: pvName}})
							if err != nil && !apierror.IsNotFound(err) {
								logger.Log("Failed to delete PersistentVolume '%s'", pvName)
								return err
							}

							Eventually(wait.IsResourceDeleted(state.GetContext(), wcClient, &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: pvName}})).
								WithTimeout(5 * time.Minute).
								WithPolling(wait.DefaultInterval).
								Should(BeTrue())
						}

						logger.Log("All associated resources deleted")
						return nil
					}).
					WithTimeout(15 * time.Minute).
					WithPolling(wait.DefaultInterval).
					Should(Succeed())
			})
		})

	})
}

func checkStorageClassExists(wcClient *client.Client) func() error {
	return func() error {
		// ensure we have at least one storage class available
		storageClasses := &storagev1.StorageClassList{}
		err := wcClient.List(state.GetContext(), storageClasses)
		if err != nil {
			return err
		}
		if len(storageClasses.Items) == 0 {
			return fmt.Errorf("no storage classes found")
		}

		return nil
	}
}

func verifyPodState(wcClient *client.Client, podName, podNamespace string) func() error {
	return func() error {

		pod := &corev1.Pod{}
		logger.Log("Getting pod '%s' in namespace '%s'", podName, podNamespace)
		err := wcClient.Get(state.GetContext(), cr.ObjectKey{Name: podName, Namespace: podNamespace}, pod)
		if err != nil {
			logger.Log("Failed to get pod '%s' in namespace '%s' - %v", podName, podNamespace, err)
			return err
		}

		if pod.Status.Phase != corev1.PodRunning {
			logger.Log("Pod '%s' in namespace '%s' is not running", podName, podNamespace)
			return fmt.Errorf("pod %s in namespace %s is not running", podName, podNamespace)
		}

		logger.Log("Pod '%s' in namespace '%s' is running successfully", podName, podNamespace)

		return nil
	}
}
