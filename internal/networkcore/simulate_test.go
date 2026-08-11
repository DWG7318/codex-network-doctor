package networkcore

import (
	"strings"
	"testing"
)

func TestSimulateSuccessProducesVerifiedReceipt(t *testing.T) {
	receipt, err := Simulate(validPlan(), ScenarioSuccess)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ExecutionVerified || receipt.RollbackTriggered {
		t.Fatalf("receipt = %+v", receipt)
	}
	if !receipt.RollbackArmed || receipt.EvidenceMode != string(EvidenceSynthetic) {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestSimulateVerificationFailureRollsBack(t *testing.T) {
	receipt, err := Simulate(validPlan(), ScenarioVerificationFailed)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ExecutionRolledBack || !receipt.RollbackTriggered {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestSimulateTimeoutRollsBack(t *testing.T) {
	receipt, err := Simulate(validPlan(), ScenarioTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ExecutionRolledBack || !receipt.RollbackTriggered {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestSimulateRejectsUnsupportedScenario(t *testing.T) {
	_, err := Simulate(validPlan(), Scenario("live"))
	if err == nil || err.Error() != "SCENARIO_UNSUPPORTED" {
		t.Fatalf("error = %v, want SCENARIO_UNSUPPORTED", err)
	}
}

func validPlan() NetworkChangePlan {
	return NetworkChangePlan{
		SchemaVersion:   SchemaVersion,
		PlanID:          strings.Repeat("a", 64),
		State:           PlanReady,
		ToolVersion:     "0.3.0",
		Platform:        "ubuntu-24.04",
		Architecture:    "amd64",
		ProfileID:       "overseas-host",
		NodeID:          "cn-primary",
		NodeDirection:   DirectionChinaEgress,
		DefaultOutbound: OutboundDirect,
		TunnelOutbound:  OutboundTunnel,
		Candidate: CandidateSpec{
			Engine:        "sing-box",
			EngineVersion: SingBoxVersion,
			SecretRef:     "lcnd-secret://nodes/cn-primary",
		},
		CandidateSHA256: strings.Repeat("b", 64),
		SimulationOnly:  true,
	}
}

func verifiedReceipt() ExecutionReceipt {
	receipt, err := Simulate(validPlan(), ScenarioSuccess)
	if err != nil {
		panic(err)
	}
	return receipt
}
