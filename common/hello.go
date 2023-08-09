package common

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/application"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

func helloWorld() {
	Context("hello world", func() {
		var customFormatterKey format.CustomFormatterKey
		BeforeEach(func() {
			customFormatterKey = format.RegisterCustomFormatter(func(value interface{}) (string, bool) {
				app, ok := value.(v1alpha1.App)
				if ok {
					return fmt.Sprintf("App: %s/%s, Status: %s, Reason: %s", app.Namespace, app.Name, app.Status.Release.Status, app.Status.Release.Reason), true
				}

				return "", false
			})
		})

		It("hello world app is deployed and responds", func() {
			ctx := context.Background()
			managementClusterKubeClient := Framework.MC()

			app, err := application.New("ingress-nginx-app", "ingress-nginx-app").WithCatalog("giantswarm").WithValues("helloworld_values.yaml", &application.TemplateValues{
				ClusterName:  Cluster.Name,
				Organization: Cluster.Organization.Name,
			})
			Expect(err).ShouldNot(HaveOccurred())
			nginxApp, nginxConfigMap, err := app.Build()
			Expect(err).ShouldNot(HaveOccurred())

			err = managementClusterKubeClient.Create(ctx, nginxConfigMap)
			Expect(err).ShouldNot(HaveOccurred())

			err = managementClusterKubeClient.Create(ctx, nginxApp)
			Expect(err).ShouldNot(HaveOccurred())

			helloworldApp, err := application.New("hello-world-app", "hello-world-app").WithCatalog("giantswarm").WithValues("nginx_values.yaml", &application.TemplateValues{
				ClusterName:  Cluster.Name,
				Organization: Cluster.Organization.Name,
			})
			Expect(err).ShouldNot(HaveOccurred())
			helloApp, helloConfigMap, err := helloworldApp.Build()
			Expect(err).ShouldNot(HaveOccurred())

			err = managementClusterKubeClient.Create(ctx, helloConfigMap)
			Expect(err).ShouldNot(HaveOccurred())

			err = managementClusterKubeClient.Create(ctx, helloApp)
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(Framework.GetApp, "5m", "10s").WithContext(ctx).WithArguments(nginxApp.Name, nginxApp.Namespace).Should(HaveAppStatus("deployed"))
			Eventually(Framework.GetApp, "5m", "10s").WithContext(ctx).WithArguments(helloApp.Name, helloApp.Namespace).Should(HaveAppStatus("deployed"))
		})

		AfterEach(func() {
			format.UnregisterCustomFormatter(customFormatterKey)
		})
	})
}
