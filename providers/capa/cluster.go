package capa

import (
	"fmt"
	"math/rand"
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

	return application.NewClusterApp(clusterName, application.ProviderAWS).
		WithOrg(organization.New(orgName)).
		WithAppValuesFile(path.Clean(clusterValuesFile), path.Clean(defaultAppsValuesFile), &application.TemplateValues{
			ClusterName:  clusterName,
			Organization: orgName,
		})
}

type PrivateClusterBuilder struct{}

func (c *PrivateClusterBuilder) NewClusterApp(clusterName string, orgName string, clusterValuesFile string, defaultAppsValuesFile string) *application.Cluster {
	if clusterName == "" {
		clusterName = utils.GenerateRandomName("t")
	}
	if orgName == "" {
		orgName = utils.GenerateRandomName("t")
	}

	// WC CIDRs have to not overlap and be in the 10.225. - 10.255. range, so
	// we select a random number in that range and set it as the second octet.
	randomOctet := rand.Intn(30) + 225
	cidrOctet := fmt.Sprintf("%d", randomOctet)
	values := &application.TemplateValues{
		ClusterName:  clusterName,
		Organization: orgName,
		ExtraValues: map[string]string{
			"CIDRSecondOctet": cidrOctet,
		},
	}

	return application.NewClusterApp(clusterName, application.ProviderAWS).
		WithOrg(organization.New(orgName)).
		WithAppValuesFile(path.Clean(clusterValuesFile), path.Clean(defaultAppsValuesFile), values)
}
