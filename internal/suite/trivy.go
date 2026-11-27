package suite

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/clustertest/v5/pkg/application"
	"sigs.k8s.io/yaml"
)

const securityBundleAppName = "security-bundle"

// securityBundleDependsOnMapVersion is the first security-bundle version in
// which `apps.<name>.dependsOn` is a map from dependency name to true/false
// instead of a list. Its schema rejects the other form.
var securityBundleDependsOnMapVersion = semver.MustParse("2.5.0")

// The security-bundle makes trivy-operator depend on cilium, kyverno-crds and
// trivy. The suites enable trivy-operator without the trivy server, so that
// HelmRelease never exists and trivy-operator would wait for it forever.
const (
	trivyOperatorDependsOnMap = `global:
  apps:
    securityBundle:
      values:
        apps:
          trivyOperator:
            dependsOn:
              trivy: false
`
	trivyOperatorDependsOnList = `global:
  apps:
    securityBundle:
      values:
        apps:
          trivyOperator:
            dependsOn:
              - cilium
              - kyverno-crds
`
)

// enablesTrivyOperator reports whether the cluster values turn on trivy-operator
// in the security-bundle.
func enablesTrivyOperator(clusterValues string) bool {
	var parsed struct {
		Global struct {
			Apps struct {
				SecurityBundle struct {
					Values struct {
						Apps struct {
							TrivyOperator struct {
								Enabled bool `json:"enabled"`
							} `json:"trivyOperator"`
						} `json:"apps"`
					} `json:"values"`
				} `json:"securityBundle"`
			} `json:"apps"`
		} `json:"global"`
	}
	if err := yaml.Unmarshal([]byte(clusterValues), &parsed); err != nil {
		return false
	}
	return parsed.Global.Apps.SecurityBundle.Values.Apps.TrivyOperator.Enabled
}

// trivyOperatorDependsOnValues returns the cluster values that drop trivy from
// the dependencies of trivy-operator, in the form the given security-bundle
// version accepts. Only MAJOR.MINOR.PATCH is compared, like in the rest of the
// repo, so release candidates and dev builds count as their final release.
func trivyOperatorDependsOnValues(securityBundleVersion string) (string, error) {
	v, err := semver.NewVersion(strings.TrimSpace(securityBundleVersion))
	if err != nil {
		return "", fmt.Errorf("unusable %s version %q: %w", securityBundleAppName, securityBundleVersion, err)
	}
	core := semver.New(v.Major(), v.Minor(), v.Patch(), "", "")
	if core.LessThan(securityBundleDependsOnMapVersion) {
		return trivyOperatorDependsOnList, nil
	}
	return trivyOperatorDependsOnMap, nil
}

// securityBundleVersion returns the security-bundle version of the release the
// cluster is about to be created with, including overrides from the environment.
func securityBundleVersion(cluster *application.Cluster) (string, error) {
	// GetRelease resolves "latest" and catalog overrides on the cluster app in
	// place; restore them so standup does its own resolution.
	version, catalog := cluster.ClusterApp.Version, cluster.ClusterApp.Catalog
	release, err := cluster.GetRelease()
	cluster.ClusterApp.Version, cluster.ClusterApp.Catalog = version, catalog
	if err != nil {
		return "", fmt.Errorf("failed to get the release under test: %w", err)
	}
	for _, app := range release.Spec.Apps {
		if strings.EqualFold(app.Name, securityBundleAppName) {
			return app.Version, nil
		}
	}
	return "", fmt.Errorf("the release under test has no %s app", securityBundleAppName)
}
