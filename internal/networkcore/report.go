package networkcore

import "errors"

func BuildReport(plan NetworkChangePlan, receipt ExecutionReceipt) (NetworkHealthReport, error) {
	if receipt.PlanID != plan.PlanID || receipt.CandidateSHA256 != plan.CandidateSHA256 {
		return NetworkHealthReport{}, errors.New("RECEIPT_PLAN_MISMATCH")
	}
	if receipt.SchemaVersion != SchemaVersion || receipt.EvidenceMode != string(EvidenceSynthetic) {
		return NetworkHealthReport{}, errors.New("RECEIPT_INVALID")
	}

	report := NetworkHealthReport{
		SchemaVersion: SchemaVersion,
		PlanID:        plan.PlanID,
		EvidenceMode:  EvidenceSynthetic,
	}
	switch receipt.State {
	case ExecutionVerified:
		report.Status = HealthSimulatedPass
		report.Summary = "Synthetic execution reached the verified state; no live Ubuntu health claim was made."
		report.Findings = []Finding{{
			Code:        "SIMULATION_VERIFIED",
			Scope:       "simulation",
			Explanation: "All simulated guards completed.",
			Action:      "Proceed only to an approved live-read-only slice.",
		}}
	case ExecutionRolledBack:
		report.Status = HealthSimulatedRollback
		report.Summary = "Synthetic execution exercised the rollback path; no production change occurred."
		report.Findings = []Finding{{
			Code:        "SIMULATION_ROLLED_BACK",
			Scope:       "simulation",
			Explanation: "A simulated failure or timeout triggered rollback.",
			Action:      "Inspect the scenario evidence before another simulation.",
		}}
	default:
		return NetworkHealthReport{}, errors.New("RECEIPT_STATE_UNSUPPORTED")
	}
	return report, nil
}
