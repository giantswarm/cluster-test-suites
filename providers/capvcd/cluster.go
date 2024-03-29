package capvcd

import (
	"path"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/organization"
	"github.com/giantswarm/clustertest/pkg/utils"
)

const (
	RegCredSecretName      = "container-registries-configuration"
	RegCredSecretNamespace = "default"
)

type ClusterBuilder struct{}

func (c *ClusterBuilder) NewClusterApp(clusterName string, orgName string, clusterValuesFile string, defaultAppsValuesFile string) *application.Cluster {
	if clusterName == "" {
		clusterName = utils.GenerateRandomName("t")
	}
	if orgName == "" {
		orgName = utils.GenerateRandomName("t")
	}

	return application.NewClusterApp(clusterName, application.ProviderCloudDirector).
		WithOrg(organization.New(orgName)).
		WithAppValuesFile(path.Clean(clusterValuesFile), path.Clean(defaultAppsValuesFile), &application.TemplateValues{
			ClusterName:  clusterName,
			Organization: orgName,
		}).
		WithExtraConfigs([]applicationv1alpha1.AppExtraConfig{
			{
				Kind:      "secret",
				Name:      RegCredSecretName,
				Namespace: RegCredSecretNamespace,
				Priority:  25,
			},
		})
}
