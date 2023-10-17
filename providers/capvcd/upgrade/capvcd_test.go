package upgrade

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/internal/common"
	"github.com/giantswarm/cluster-test-suites/internal/upgrade"
)

var _ = PDescribe("Basic upgrade test", Ordered, func() {
	upgrade.Run()

	// Finally run the common tests after upgrade is completed
	common.Run(&common.TestConfig{
		BastionSupported: true,
		// Disabled until https://github.com/giantswarm/roadmap/issues/1037
		ExternalDnsSupported: false,
	})
})
