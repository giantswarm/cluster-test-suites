package upgrade

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capz"
)

const KubeContext = "capz"

func TestCAPZUpgrade(t *testing.T) {
	if strings.TrimSpace(os.Getenv("E2E_OVERRIDE_VERSIONS")) == "" {
		Skip("E2E_OVERRIDE_VERSIONS env var not set, skipping upgrade test")
	} else {
		suite.Setup(KubeContext, &capz.ClusterBuilder{})
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPZ Upgrade Suite")
}
