package standard

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck

	"github.com/giantswarm/cluster-test-suites/v7/internal/common"
	"github.com/giantswarm/cluster-test-suites/v7/internal/ecr"
)

var _ = Describe("Common tests", func() {
	cfg := common.NewTestConfigWithDefaults()
	cfg.GatewayAPISupported = false
	common.Run(cfg)

	// ECR Credential Provider specific tests
	ecr.Run()
})
