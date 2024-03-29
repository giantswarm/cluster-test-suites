package capz

import (
	"path"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/organization"
	"github.com/giantswarm/clustertest/pkg/utils"
)

type ClusterBuilder struct{}

func (c *ClusterBuilder) NewClusterApp(clusterName string, orgName string, clusterValuesFile string, defaultAppsValuesFile string) *application.Cluster {
	if clusterName == "" {
		clusterName = utils.GenerateRandomName("t")
	}
	if orgName == "" {
		orgName = utils.GenerateRandomName("t")
	}

	return application.NewClusterApp(clusterName, application.ProviderAzure).
		WithOrg(organization.New(orgName)).
		WithAppValuesFile(path.Clean(clusterValuesFile), path.Clean(defaultAppsValuesFile), &application.TemplateValues{
			ClusterName:  clusterName,
			Organization: orgName,
		})
}
