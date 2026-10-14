package capz

import (
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/clustertest/v5/pkg/env"

	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

// autoscalingMinRelease is the first release whose cluster-azure understands the
// `minSize` / `maxSize` node pool fields and deploys cluster-autoscaler. Older releases
// ignore those fields and default the node pool to `replicas: 1`, while the tests expect
// at least `minSize` (2) worker nodes.
//
// TODO: remove this gate once v35.0.0 is the minimum release across CI.
var autoscalingMinRelease = semver.MustParse("v35.0.0")

// legacyNodePoolValues pins the node pool size for releases that don't support autoscaling.
const legacyNodePoolValues = `global:
  nodePools:
    nodepool-0:
      replicas: 2
`

// releaseSupportsAutoscaling reports whether the given release version supports node pool
// autoscaling. An empty or unparseable version (e.g. PRs to this repo, where the latest
// cluster chart is used) is treated as supported. Release candidates of
// autoscalingMinRelease count as supported, as only MAJOR.MINOR.PATCH is compared.
func releaseSupportsAutoscaling(release string) bool {
	v, err := semver.NewVersion(strings.TrimSpace(release))
	if err != nil {
		return true
	}
	core := semver.New(v.Major(), v.Minor(), v.Patch(), "", "")
	return !core.LessThan(autoscalingMinRelease)
}

// AutoScalingSupported reports whether the release under test (E2E_RELEASE_VERSION)
// supports node pool autoscaling. For upgrade suites this is the release the cluster is
// upgraded to.
func AutoScalingSupported() bool {
	return releaseSupportsAutoscaling(os.Getenv(env.ReleaseVersion))
}

// WithLegacyNodePoolReplicas returns a suite option that sets `replicas: 2` on the default
// node pool when the release the cluster is created with doesn't support autoscaling.
//
// Upgrade suites create the cluster with E2E_RELEASE_PRE_UPGRADE, so that version takes
// precedence over E2E_RELEASE_VERSION.
func WithLegacyNodePoolReplicas(isUpgrade bool) suite.Option {
	return suite.WithExtraClusterValues(func() (string, error) {
		rv := ""
		if isUpgrade {
			rv = strings.TrimSpace(os.Getenv(env.ReleasePreUpgradeVersion))
		}
		if rv == "" {
			rv = os.Getenv(env.ReleaseVersion)
		}

		if releaseSupportsAutoscaling(rv) {
			return "", nil
		}
		return legacyNodePoolValues, nil
	})
}
