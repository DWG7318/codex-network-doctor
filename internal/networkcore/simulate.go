package networkcore

import "errors"

func Simulate(plan NetworkChangePlan, scenario Scenario) (ExecutionReceipt, error) {
	if plan.State != PlanReady || !plan.SimulationOnly || plan.PlanID == "" || plan.CandidateSHA256 == "" {
		return ExecutionReceipt{}, errors.New("PLAN_NOT_SIMULATABLE")
	}

	receipt := baseReceipt(plan, scenario)
	switch scenario {
	case ScenarioSuccess:
		receipt.State = ExecutionVerified
		receipt.Actions = append(receipt.Actions,
			ActionResult{Action: "verify_candidate", Status: "simulated", Code: "SIMULATION_VERIFIED"},
		)
	case ScenarioVerificationFailed:
		receipt.State = ExecutionRolledBack
		receipt.RollbackTriggered = true
		receipt.Actions = append(receipt.Actions,
			ActionResult{Action: "verify_candidate", Status: "failed", Code: "SIMULATED_VERIFICATION_FAILED"},
			ActionResult{Action: "rollback_candidate", Status: "simulated", Code: "SIMULATION_ROLLED_BACK"},
		)
	case ScenarioTimeout:
		receipt.State = ExecutionRolledBack
		receipt.RollbackTriggered = true
		receipt.Actions = append(receipt.Actions,
			ActionResult{Action: "verify_candidate", Status: "timeout", Code: "SIMULATED_TIMEOUT"},
			ActionResult{Action: "rollback_candidate", Status: "simulated", Code: "SIMULATION_ROLLED_BACK"},
		)
	default:
		return ExecutionReceipt{}, errors.New("SCENARIO_UNSUPPORTED")
	}
	return receipt, nil
}

func baseReceipt(plan NetworkChangePlan, scenario Scenario) ExecutionReceipt {
	return ExecutionReceipt{
		SchemaVersion:   SchemaVersion,
		PlanID:          plan.PlanID,
		EvidenceMode:    string(EvidenceSynthetic),
		Scenario:        scenario,
		CandidateSHA256: plan.CandidateSHA256,
		RollbackArmed:   true,
		Actions: []ActionResult{
			{Action: "arm_rollback", Status: "simulated", Code: "SIMULATED_ROLLBACK_ARMED"},
			{Action: "stage_candidate", Status: "simulated", Code: "SIMULATED_CANDIDATE_STAGED"},
		},
	}
}
