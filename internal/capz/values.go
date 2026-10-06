package capz

import (
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/clustertest/v5/pkg/env"

	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

// nodePoolAutoscalingMinRelease is the first release whose cluster-azure understands the
// `minSize` / `maxSize` node pool fields. Older releases ignore them and default the node
// pool to `replicas: 1`, while the tests expect at least `minSize` (2) worker nodes.
//
// TODO: remove this gate once v35.0.0 is the minimum release across CI.
var nodePoolAutoscalingMinRelease = semver.MustParse("v35.0.0")

// legacyNodePoolValues pins the node pool size for releases that don't support autoscaling.
const legacyNodePoolValues = `global:
  nodePools:
    nodepool-0:
      replicas: 2
`

// WithLegacyNodePoolReplicas returns a suite option that sets `replicas: 2` on the default
// node pool when the release the cluster is created with is older than v35.0.0.
//
// Upgrade suites create the cluster with E2E_RELEASE_PRE_UPGRADE, so that version takes
// precedence over E2E_RELEASE_VERSION. When no parseable release version is set (e.g. PRs
// to this repo) no override is applied.
func WithLegacyNodePoolReplicas(isUpgrade bool) suite.Option {
	return suite.WithExtraClusterValues(func() (string, error) {
		rv := ""
		if isUpgrade {
			rv = strings.TrimSpace(os.Getenv(env.ReleasePreUpgradeVersion))
		}
		if rv == "" {
			rv = strings.TrimSpace(os.Getenv(env.ReleaseVersion))
		}

		v, err := semver.NewVersion(rv)
		if err != nil {
			return "", nil
		}

		// Compare core MAJOR.MINOR.PATCH only so that v35.0.0 release candidates count as v35.
		core := semver.New(v.Major(), v.Minor(), v.Patch(), "", "")
		if core.LessThan(nodePoolAutoscalingMinRelease) {
			return legacyNodePoolValues, nil
		}
		return "", nil
	})
}
