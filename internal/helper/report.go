package helper

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

// RecordNodeRolling annotates the current test spec with a boolean indicating if nodes were rolled during the test.
func RecordNodeRolling(rolled bool) {
	// Store as a simple string "yes" or "no" to avoid any parsing issues
	value := "no"
	if rolled {
		value = "yes"
	}
	AddReportEntry("NODES_ROLLED", value)
}
