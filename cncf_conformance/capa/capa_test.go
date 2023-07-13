package capa

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/cluster-test-suites/cncf_conformance/internal/conformance"
)

var _ = Describe("Conformance tests", func() {
	conformance.Run()
})
