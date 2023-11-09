package standard

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
)

var _ = Describe("Common tests", func() {
	common.Run(&common.TestConfig{
		AutoScalingSupported: true,
		BastionSupported:     true,
		ExternalDnsSupported: true,
	})
})
