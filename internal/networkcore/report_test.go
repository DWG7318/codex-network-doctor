package networkcore

import (
	"strings"
	"testing"
)

func TestReportIsSyntheticAndNeverClaimsLiveHealth(t *testing.T) {
	report, err := BuildReport(validPlan(), verifiedReceipt())
	if err != nil {
		t.Fatal(err)
	}
	if report.EvidenceMode != EvidenceSynthetic || report.Status != HealthSimulatedPass {
		t.Fatalf("report = %+v", report)
	}
	lower := strings.ToLower(report.Summary)
	if strings.Contains(lower, "production") || strings.Contains(lower, "live health is healthy") {
		t.Fatalf("summary = %q", report.Summary)
	}
}

func TestReportMapsRollbackWithoutClaimingLiveFailure(t *testing.T) {
	receipt, err := Simulate(validPlan(), ScenarioVerificationFailed)
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildReport(validPlan(), receipt)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != HealthSimulatedRollback || report.EvidenceMode != EvidenceSynthetic {
		t.Fatalf("report = %+v", report)
	}
}

func TestReportRejectsReceiptFromDifferentPlan(t *testing.T) {
	receipt := verifiedReceipt()
	receipt.PlanID = strings.Repeat("c", 64)
	_, err := BuildReport(validPlan(), receipt)
	if err == nil || err.Error() != "RECEIPT_PLAN_MISMATCH" {
		t.Fatalf("error = %v, want RECEIPT_PLAN_MISMATCH", err)
	}
}
