package clusterwidedeny

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
)

var _ = Describe("Clusterwide deny", func() {
	cfg := common.NewTestConfigWithDefaults()
	common.RunWithClusterwideDeny(cfg)
})
