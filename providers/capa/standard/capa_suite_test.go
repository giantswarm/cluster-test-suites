package standard

import (
	"fmt"
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/clusterbuilder/providers/capa"
	"github.com/giantswarm/cluster-standup-teardown/v6/pkg/values"
	"github.com/giantswarm/clustertest/v5/pkg/env"

	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

// armMinRelease is the first release that bundles cluster-aws 8.5.0+ (which adds the
// `architecture` / `instanceType` schema fields needed for the arm64 ASG nodepool) and
// the multi-arch net-exporter / cert-exporter app versions.
//
// TODO(arm64): remove this gate (and the cluster_values_arm.yaml overlay) once v35.0.0
// is the minimum release across CI. https://github.com/giantswarm/roadmap/issues/4302
var armMinRelease = semver.MustParse("v35.0.0")

// armSupported reports whether the release under test supports the ARM additions.
// Conservative default: empty E2E_RELEASE_VERSION (PRs to this repo, or any CI run
// without an explicit release pin) is treated as unsupported so we don't apply ARM-only
// fields against an older cluster-aws and break standup on older release lines
// (e.g. v33.x, v34.x). Same for unparseable versions.
//
// Release candidates of armMinRelease (e.g. v35.0.0-rc.1) are treated as supported: by
// the time an RC is being tested, the schema is finalized. We compare core MAJOR.MINOR.PATCH
// and ignore prerelease/build metadata.
func armSupported() bool {
	rv := os.Getenv(env.ReleaseVersion)
	if rv == "" {
		return false
	}
	v, err := semver.NewVersion(rv)
	if err != nil {
		return false
	}
	core, err := semver.NewVersion(fmt.Sprintf("v%d.%d.%d", v.Major(), v.Minor(), v.Patch()))
	if err != nil {
		return false
	}
	return !core.LessThan(armMinRelease)
}

func TestCAPAStandard(t *testing.T) {
	opts := []suite.Option{
		suite.WithExtraClusterValues(func() (string, error) {
			if !armSupported() {
				return "", nil
			}
			return values.MustLoadValuesFile("./test_data/cluster_values_arm.yaml"), nil
		}),
	}
	suite.SetupWithOptions(false, &capa.ClusterBuilder{}, opts)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Standard Suite")
}
