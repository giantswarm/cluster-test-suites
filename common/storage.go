package common

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/cluster-test-suites/assets/storage"
	"github.com/giantswarm/cluster-test-suites/helper"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	namespace = "test-storage"
)

func runStorage() {
	Context("storage", func() {
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

	})
}

func cleanupStorage() {
	Context("storage", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			var err error

			wcClient, err = Framework.WC(Cluster.Name)
			if err != nil {
				Fail(err.Error())
			}
		})

		It("has deleted all objects in namespace test-storage", func() {
			Eventually(wait.Consistent(deleteStorage(wcClient, namespace), 10, time.Second)).
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

		namespaceObj, err := helper.Deserialize(storage.Namespace)
		if err != nil {
			return err
		}
		namespace := namespaceObj.(*corev1.Namespace)
		err = wcClient.Create(context.Background(), namespace)
		if err != nil {
			if apierror.IsAlreadyExists(err) {
				// fall through
			}
			return nil
		}

		pvcObj, err := helper.Deserialize(storage.PVC)
		if err != nil {
			return err
		}
		pvc := pvcObj.(*corev1.PersistentVolumeClaim)
		err = wcClient.Create(context.Background(), pvc)
		if err != nil {
			if apierror.IsAlreadyExists(err) {
				// fall through
			}
			return err
		}

		podObj, err := helper.Deserialize(storage.Pod)
		if err != nil {
			return err
		}
		pod := podObj.(*corev1.Pod)
		err = wcClient.Create(context.Background(), pod)
		if err != nil {
			if apierror.IsAlreadyExists(err) {
				// fall through
			}
			return err
		}

		return nil
	}
}

func deleteStorage(wcClient *client.Client, namespace string) func() error {
	return func() error {
		return wcClient.DeleteAllOf(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
	}
}
