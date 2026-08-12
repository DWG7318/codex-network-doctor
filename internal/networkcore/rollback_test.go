package networkcore

import (
	"errors"
	"testing"
)

func TestRollbackEvaluationIsScopedAndDeterministic(t *testing.T) {
	receipt := validRollbackReceipt()
	decision, err := EvaluateRollback(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != RollbackApplyEligible || decision.Scope.WorkloadID != "lcnd-egress-cn" || decision.Scope.AttachmentID != "lcapi-domestic-synthetic" {
		t.Fatalf("decision = %#v", decision)
	}
	if decision.ProductionAuthorized {
		t.Fatal("synthetic decision authorized production")
	}
}

func TestRollbackEvaluationFailsClosedWithoutArmedRollback(t *testing.T) {
	receipt := validRollbackReceipt()
	receipt.RollbackArmed = false
	decision, err := EvaluateRollback(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != RollbackFailClosed || !containsString(decision.Triggers, "ROLLBACK_NOT_ARMED") {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestRollbackEvaluationRequiresRollbackOnDigestOrProtectedTargetDrift(t *testing.T) {
	receipt := validRollbackReceipt()
	receipt.ObservedCandidateSHA256 = "sha256:" + repeatHex("b")
	receipt.ProtectedTargetsUnchanged = false
	decision, err := EvaluateRollback(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != RollbackRequired || !containsString(decision.Triggers, "CANDIDATE_DIGEST_MISMATCH") || !containsString(decision.Triggers, "PROTECTED_TARGET_DRIFT") {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestRollbackEvaluationRejectsCrossDomainReceipt(t *testing.T) {
	receipt := validRollbackReceipt()
	receipt.RollbackDomain = "lcsp-continuity-b"
	_, err := EvaluateRollback(receipt)
	if err == nil || err.Error() != "WORKLOAD_DOMAIN_COLLISION" {
		t.Fatalf("error = %v", err)
	}
}

func TestRollbackEvaluationRejectsUnknownWorkloadDomain(t *testing.T) {
	receipt := validRollbackReceipt()
	receipt.WorkloadID = "unknown-workload"
	receipt.RollbackDomain = "unknown-workload"
	_, err := EvaluateRollback(receipt)
	var validationErr ValidationError
	if err == nil || !errors.As(err, &validationErr) || !hasIssue(validationErr.Issues, "WORKLOAD_UNSUPPORTED") {
		t.Fatalf("error = %v", err)
	}
}

func validRollbackReceipt() RollbackReceipt {
	digest := "sha256:" + repeatHex("a")
	return RollbackReceipt{
		SchemaVersion:             SchemaVersion,
		PlanID:                    "lcnd-plan-20260812-egress-cn-synthetic",
		WorkloadID:                "lcnd-egress-cn",
		RollbackDomain:            "lcnd-egress-cn",
		AttachmentID:              "lcapi-domestic-synthetic",
		ExpectedCandidateSHA256:   digest,
		ObservedCandidateSHA256:   digest,
		RollbackArmed:             true,
		ProtectedTargetsUnchanged: true,
		OldLineHealthy:            true,
		Triggers:                  []string{},
		EvidenceMode:              EvidenceSynthetic,
	}
}
