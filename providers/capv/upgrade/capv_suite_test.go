package upgrade

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/internal/suite"
	"github.com/giantswarm/cluster-test-suites/providers/capv"
)

const KubeContext = "capv"

func TestCAPVUpgrade(t *testing.T) {
	if strings.TrimSpace(os.Getenv("E2E_OVERRIDE_VERSIONS")) == "" {
		Skip("E2E_OVERRIDE_VERSIONS env var not set, skipping upgrade test")
	} else {
		suite.Setup(KubeContext, &capv.ClusterBuilder{})
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPV Upgrade Suite")
}
