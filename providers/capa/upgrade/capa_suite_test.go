package upgrade

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capa"
)

const KubeContext = "capa"

func TestCAPAUpgrade(t *testing.T) {
	if strings.TrimSpace(os.Getenv("E2E_OVERRIDE_VERSIONS")) == "" {
		Skip("E2E_OVERRIDE_VERSIONS env var not set, skipping upgrade test")
	} else {
		suite.Setup(KubeContext, &capa.ClusterBuilder{})
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPA Upgrade Suite")
}
