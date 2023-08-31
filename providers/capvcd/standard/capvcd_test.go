package standard

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/common"
)

var _ = PDescribe("Common tests", func() {
	common.Run(&common.TestConfig{
		BastionSupported: true,
	})
})
