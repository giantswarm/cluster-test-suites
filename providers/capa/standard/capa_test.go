package standard

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
)

var _ = Describe("Common tests", func() {
	common.Run(&common.TestConfig{
		BastionSupported:     true,
		ExternalDnsSupported: true,
	})
})
