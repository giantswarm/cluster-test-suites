package eks

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

	return application.NewClusterApp(clusterName, application.ProviderEKS).
		WithOrg(organization.New(orgName)).
		WithAppValuesFile(path.Clean(clusterValuesFile), path.Clean(defaultAppsValuesFile), &application.TemplateValues{
			ClusterName:  clusterName,
			Organization: orgName,
		}).WithAppVersions("0.10.0-ed5ac1348d6b244573c71c323450089f5a68e419", "0.3.1-32fa6ef6f59b1418889134c77d0400840922db78")
}
