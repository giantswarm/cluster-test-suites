package upgrade

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/upgrade"
)

var _ = Describe("Basic upgrade test", Ordered, func() {
	upgrade.Run()

	// Finally run the common tests after upgrade is completed
	common.Run(&common.TestConfig{
		BastionSupported:     true,
		ExternalDnsSupported: true,
	})
})
