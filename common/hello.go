package common

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/organization"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	v1 "k8s.io/api/core/v1"

	"github.com/giantswarm/cluster-test-suites/internal/state"
)

func helloWorld() {
	Context("hello world", func() {
		var customFormatterKey format.CustomFormatterKey
		var nginxApp, helloApp *v1alpha1.App
		var nginxConfigMap, helloConfigMap *v1.ConfigMap

		BeforeEach(func() {
			customFormatterKey = format.RegisterCustomFormatter(func(value interface{}) (string, bool) {
				app, ok := value.(v1alpha1.App)
				if ok {
					return fmt.Sprintf("App: %s/%s, Status: %s, Reason: %s", app.Namespace, app.Name, app.Status.Release.Status, app.Status.Release.Reason), true
				}

				return "", false
			})
			ctx := context.Background()
			org := organization.New("giantswarm")

			nginxApp, nginxConfigMap = deployApp(ctx, "ingress-nginx", "kube-system", org, "3.0.0", "./test_data/nginx_values.yaml")
			Eventually(state.GetFramework().GetApp, "5m", "10s").WithContext(ctx).WithArguments(nginxApp.Name, nginxApp.Namespace).Should(HaveAppStatus("deployed"))
			helloApp, helloConfigMap = deployApp(ctx, "hello-world", "giantswarm", org, "2.0.0", "./test_data/helloworld_values.yaml")
			Eventually(state.GetFramework().GetApp, "5m", "10s").WithContext(ctx).WithArguments(helloApp.Name, helloApp.Namespace).Should(HaveAppStatus("deployed"))
		})

		It("hello world app responds successfully", func() {
			Eventually(func() (*http.Response, error) {
				return http.Get(fmt.Sprintf("https://hello-world.%s.gaws.gigantic.io", state.GetCluster().Name))
			}, "5m", "5s").Should(HaveAppStatus("200"))
		})

		AfterEach(func() {
			ctx := context.Background()
			format.UnregisterCustomFormatter(customFormatterKey)

			managementClusterKubeClient := state.GetFramework().MC()
			err := managementClusterKubeClient.Delete(ctx, nginxApp)
			Expect(err).ShouldNot(HaveOccurred())
			err = managementClusterKubeClient.Delete(ctx, nginxConfigMap)
			Expect(err).ShouldNot(HaveOccurred())
			err = managementClusterKubeClient.Delete(ctx, helloApp)
			Expect(err).ShouldNot(HaveOccurred())
			err = managementClusterKubeClient.Delete(ctx, helloConfigMap)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
}

func deployApp(ctx context.Context, name, namespace string, organization *organization.Org, version, valuesFile string) (*v1alpha1.App, *v1.ConfigMap) {
	managementClusterKubeClient := state.GetFramework().MC()
	appTemplate, err := application.New(fmt.Sprintf("%s-%s", state.GetCluster().Name, name), name).
		WithCatalog("giantswarm").
		WithOrganization(*organization).
		WithVersion(version).
		WithInCluster(false).
		WithAppLabels(map[string]string{"giantswarm.io/cluster": state.GetCluster().Name}).
		WithValuesFile(path.Clean(valuesFile), &application.TemplateValues{
			ClusterName:  state.GetCluster().Name,
			Organization: state.GetCluster().Organization.Name,
		})
	Expect(err).ShouldNot(HaveOccurred())
	app, configMap, err := appTemplate.Build()
	Expect(err).ShouldNot(HaveOccurred())

	err = managementClusterKubeClient.Create(ctx, configMap)
	Expect(err).ShouldNot(HaveOccurred())

	app.Spec.Config.ConfigMap.Name = fmt.Sprintf("%s-cluster-values", state.GetCluster().Name)
	app.Spec.KubeConfig.Context.Name = fmt.Sprintf("%s-admin@%s", state.GetCluster().Name, state.GetCluster().Name)
	app.Spec.KubeConfig.Secret.Name = fmt.Sprintf("%s-kubeconfig", state.GetCluster().Name)
	app.Spec.Namespace = namespace
	err = managementClusterKubeClient.Create(ctx, app)
	Expect(err).ShouldNot(HaveOccurred())

	return app, configMap
}
