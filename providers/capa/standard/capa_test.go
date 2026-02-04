package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v3/internal/common"
	"github.com/giantswarm/cluster-test-suites/v3/internal/ecr"
)

var _ = Describe("Common tests", func() {
	common.Run(common.NewTestConfigWithDefaults())

	// ECR Credential Provider specific tests
	ecr.Run()
})
