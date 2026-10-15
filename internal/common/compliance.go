package common

import (
	"sort"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/giantswarm/clustertest/v5/pkg/client"
	"github.com/giantswarm/clustertest/v5/pkg/logger"

	"github.com/giantswarm/cluster-test-suites/v7/internal/helper"
	"github.com/giantswarm/cluster-test-suites/v7/internal/state"
	"github.com/giantswarm/cluster-test-suites/v7/internal/timeout"
)

var clusterComplianceReportListGVK = schema.GroupVersionKind{
	Group:   "aquasecurity.github.io",
	Version: "v1alpha1",
	Kind:    "ClusterComplianceReportList",
}

// runCompliance surfaces the Trivy Operator compliance benchmark results (CIS,
// NSA, PSS, ...) from the workload cluster in the test output. It is
// report-only: findings are logged but never fail the suite, so it provides a
// per-release snapshot of benchmark compliance without blocking releases.
// See https://github.com/giantswarm/roadmap/issues/2624.
func runCompliance(cfg *TestConfig) {
	Context("compliance reports", func() {
		var wcClient *client.Client

		BeforeEach(func() {
			if !cfg.SecurityBundleInstalled {
				Skip("security-bundle is not installed")
			}

			helper.SetResponsibleTeam(helper.TeamShield)

			Eventually(func() error {
				var err error
				wcClient, err = state.GetFramework().WC(state.GetCluster().Name)
				return err
			}).
				WithTimeout(5 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})

		It("reports Trivy Operator compliance benchmark findings", func() {
			reports, err := listComplianceReports(wcClient)
			if apimeta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
				Skip("ClusterComplianceReport CRD not found; trivy-operator is not enabled in the security-bundle")
			}
			Expect(err).NotTo(HaveOccurred())

			if len(reports) == 0 {
				Skip("no ClusterComplianceReports found; trivy-operator compliance specs are not installed")
			}

			// The compliance scanner populates report statuses on a cron
			// schedule, so give it a bounded window to produce results. If
			// nothing is populated by then (scanner disabled, or the next
			// scheduled run is hours away) we log that instead of failing:
			// this check is report-only by design.
			reportTimeout := state.GetTestTimeout(timeout.ComplianceReports, 10*time.Minute)
			logger.Log("Found %d ClusterComplianceReports, waiting up to %s for their statuses to be populated", len(reports), reportTimeout.String())

			deadline := time.Now().Add(reportTimeout)
			for {
				refreshed, err := listComplianceReports(wcClient)
				if err == nil {
					reports = refreshed
				} else {
					logger.Log("Failed to refresh ClusterComplianceReports - %v", err)
				}

				if anyReportPopulated(reports) || time.Now().After(deadline) {
					break
				}
				time.Sleep(15 * time.Second)
			}

			if !anyReportPopulated(reports) {
				logger.Log("No ClusterComplianceReport has a populated status yet; the compliance scanner may be disabled ('operator.clusterComplianceEnabled') or its next scheduled run is still pending. Skipping findings report.")
				return
			}

			for _, report := range reports {
				logComplianceReport(report)
			}
		})
	})
}

func listComplianceReports(wcClient *client.Client) ([]unstructured.Unstructured, error) {
	reportList := &unstructured.UnstructuredList{}
	reportList.SetGroupVersionKind(clusterComplianceReportListGVK)
	err := wcClient.List(state.GetContext(), reportList)
	if err != nil {
		return nil, err
	}
	return reportList.Items, nil
}

// anyReportPopulated reports whether at least one ClusterComplianceReport has
// its status summary filled in by the compliance scanner.
func anyReportPopulated(reports []unstructured.Unstructured) bool {
	for _, report := range reports {
		_, found, _ := unstructured.NestedMap(report.Object, "status", "summary")
		if found {
			return true
		}
	}
	return false
}

// logComplianceReport logs one report's pass/fail summary and, for each failed
// control, its id, severity, name and the number of failing checks. Report
// shape (reportType=summary) is documented at
// https://aquasecurity.github.io/trivy-operator/latest/docs/crds/clustercompliance-report/
func logComplianceReport(report unstructured.Unstructured) {
	name := report.GetName()
	title, _, _ := unstructured.NestedString(report.Object, "spec", "compliance", "title")

	summary, found, err := unstructured.NestedMap(report.Object, "status", "summary")
	if err != nil || !found {
		logger.Log("Compliance report '%s' (%s): status not populated yet", name, title)
		return
	}

	passCount, _, _ := unstructured.NestedInt64(summary, "passCount")
	failCount, _, _ := unstructured.NestedInt64(summary, "failCount")
	logger.Log("Compliance report '%s' (%s): %d passed, %d failed", name, title, passCount, failCount)

	if failCount == 0 {
		return
	}

	controlChecks, found, err := unstructured.NestedSlice(report.Object, "status", "summaryReport", "controlCheck")
	if err != nil || !found {
		logger.Log("Compliance report '%s': no control check details available (reportType is not 'summary'?)", name)
		return
	}

	type failedControl struct {
		id          string
		controlName string
		severity    string
		totalFail   int64
	}
	failed := []failedControl{}
	for _, check := range controlChecks {
		checkMap, ok := check.(map[string]interface{})
		if !ok {
			continue
		}
		totalFail, _, _ := unstructured.NestedInt64(checkMap, "totalFail")
		if totalFail <= 0 {
			continue
		}
		id, _, _ := unstructured.NestedString(checkMap, "id")
		controlName, _, _ := unstructured.NestedString(checkMap, "name")
		severity, _, _ := unstructured.NestedString(checkMap, "severity")
		failed = append(failed, failedControl{id: id, controlName: controlName, severity: severity, totalFail: totalFail})
	}

	sort.Slice(failed, func(i, j int) bool {
		return severityRank(failed[i].severity) < severityRank(failed[j].severity)
	})

	for _, control := range failed {
		logger.Log("  FAIL [%s] %s %s (%d failing checks)", control.severity, control.id, control.controlName, control.totalFail)
	}
}

func severityRank(severity string) int {
	ranks := map[string]int{
		"CRITICAL": 0,
		"HIGH":     1,
		"MEDIUM":   2,
		"LOW":      3,
	}
	if rank, ok := ranks[severity]; ok {
		return rank
	}
	return 4
}
