package helper

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

// RecordNodeRolling annotates the current test spec with a boolean indicating if nodes were rolled during the test.
func RecordNodeRolling(rolled bool) {
	AddReportEntry("NODES_ROLLED", fmt.Sprintf("%t", rolled))
}
