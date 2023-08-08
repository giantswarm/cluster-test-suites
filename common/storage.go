package common

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"

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

		decode := scheme.Codecs.UniversalDeserializer().Decode

		base := "assets/storage"

		files, err := os.ReadDir(base)
		if err != nil {
			return err
		}

		for _, file := range files {
			podPVCYAML, err := os.ReadFile(fmt.Sprintf("%s/%s", base, file.Name()))
			if err != nil {
				return err
			}
			obj, groupVersion, _ := decode(podPVCYAML, nil, nil)
			switch groupVersion.Kind {
			case "Namespace":
				namespace := obj.(*corev1.Namespace)
				return wcClient.Create(context.Background(), namespace)
			case "PersistentVolumeClaim":
				pvc := obj.(*corev1.PersistentVolumeClaim)
				return wcClient.Create(context.Background(), pvc)

			case "Pod":
				pod := obj.(*corev1.Pod)
				return wcClient.Create(context.Background(), pod)
			}

		}
		if err != nil {
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
