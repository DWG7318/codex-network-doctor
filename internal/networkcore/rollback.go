package networkcore

import (
	"errors"
	"io"
	"slices"
)

type RollbackReceipt struct {
	SchemaVersion             string       `json:"schema_version"`
	PlanID                    string       `json:"plan_id"`
	WorkloadID                string       `json:"workload_id"`
	RollbackDomain            string       `json:"rollback_domain"`
	AttachmentID              string       `json:"attachment_id"`
	ExpectedCandidateSHA256   string       `json:"expected_candidate_sha256"`
	ObservedCandidateSHA256   string       `json:"observed_candidate_sha256"`
	RollbackArmed             bool         `json:"rollback_armed"`
	ProtectedTargetsUnchanged bool         `json:"protected_targets_unchanged"`
	OldLineHealthy            bool         `json:"old_line_healthy"`
	Triggers                  []string     `json:"triggers"`
	EvidenceMode              EvidenceMode `json:"evidence_mode"`
}

type RollbackDecisionKind string

const (
	RollbackApplyEligible RollbackDecisionKind = "APPLY_ELIGIBLE"
	RollbackRequired      RollbackDecisionKind = "ROLLBACK_REQUIRED"
	RollbackFailClosed    RollbackDecisionKind = "FAIL_CLOSED"
)

type RollbackScope struct {
	WorkloadID   string `json:"workload_id"`
	AttachmentID string `json:"attachment_id"`
}

type RollbackDecision struct {
	SchemaVersion        string               `json:"schema_version"`
	PlanID               string               `json:"plan_id"`
	Decision             RollbackDecisionKind `json:"decision"`
	Scope                RollbackScope        `json:"scope"`
	Triggers             []string             `json:"triggers"`
	EvidenceMode         EvidenceMode         `json:"evidence_mode"`
	ProductionAuthorized bool                 `json:"production_authorized"`
	Summary              string               `json:"summary"`
}

var rollbackTriggerSet = map[string]struct{}{
	"PROTECTED_TARGET_DRIFT": {}, "OLD_LINE_HEALTH_REGRESSION": {}, "PUBLIC_443_COLLISION": {}, "SSH_OR_LCSP_ACCESS_REGRESSION": {},
	"GLOBAL_ROUTE_CHANGED": {}, "GLOBAL_DNS_CHANGED": {}, "FIREWALL_CHANGED": {}, "SYSTEM_PROXY_CHANGED": {}, "TAILNET_PREFERENCE_CHANGED": {},
	"LCGPU_CONTROL_CHANGED": {}, "LCAPI_NON_OPTED_TRAFFIC_CHANGED": {}, "SECRET_VALUE_EXPOSED": {}, "CANDIDATE_DIGEST_MISMATCH": {},
	"WORKLOAD_DOMAIN_COLLISION": {}, "PROBE_TIMEOUT_QUORUM_FAILED": {}, "RESOURCE_BUDGET_EXCEEDED": {}, "ROLLBACK_NOT_ARMED": {}, "PRODUCTION_AUTHORITY_MISSING": {},
}

func DecodeRollbackReceipt(reader io.Reader) (RollbackReceipt, error) {
	return decodeStrict(reader, "rollback_receipt", RollbackReceipt.Validate)
}

func (receipt RollbackReceipt) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, receipt.SchemaVersion)
	issues = validateID(issues, "plan_id", receipt.PlanID)
	issues = validateID(issues, "workload_id", receipt.WorkloadID)
	issues = validateID(issues, "rollback_domain", receipt.RollbackDomain)
	if receipt.WorkloadID != "lcsp-continuity-b" && receipt.WorkloadID != "lcnd-egress-cn" {
		issues = addIssue(issues, "WORKLOAD_UNSUPPORTED", "workload_id", "rollback workload is unsupported")
	}
	issues = validateID(issues, "attachment_id", receipt.AttachmentID)
	if !sha256RefPattern.MatchString(receipt.ExpectedCandidateSHA256) || !sha256RefPattern.MatchString(receipt.ObservedCandidateSHA256) {
		issues = addIssue(issues, "CANDIDATE_DIGEST_INVALID", "candidate_sha256", "candidate digests are invalid")
	}
	if len(receipt.Triggers) > 32 {
		issues = addIssue(issues, "ROLLBACK_TRIGGERS_LIMIT_EXCEEDED", "triggers", "rollback triggers exceed the limit")
	}
	for _, trigger := range receipt.Triggers {
		if _, ok := rollbackTriggerSet[trigger]; !ok {
			issues = addIssue(issues, "ROLLBACK_TRIGGER_UNSUPPORTED", "triggers", "rollback trigger is unsupported")
		}
	}
	if receipt.EvidenceMode != EvidenceSynthetic {
		issues = addIssue(issues, "EVIDENCE_MODE_INVALID", "evidence_mode", "offline evaluator accepts synthetic receipts only")
	}
	return issues
}

func EvaluateRollback(receipt RollbackReceipt) (RollbackDecision, error) {
	if receipt.WorkloadID != receipt.RollbackDomain {
		return RollbackDecision{}, errors.New("WORKLOAD_DOMAIN_COLLISION")
	}
	if issues := receipt.Validate(); len(issues) != 0 {
		return RollbackDecision{}, ValidationError{Kind: "rollback_receipt", Issues: issues}
	}
	triggers := append([]string(nil), receipt.Triggers...)
	if receipt.ExpectedCandidateSHA256 != receipt.ObservedCandidateSHA256 {
		triggers = append(triggers, "CANDIDATE_DIGEST_MISMATCH")
	}
	if !receipt.ProtectedTargetsUnchanged {
		triggers = append(triggers, "PROTECTED_TARGET_DRIFT")
	}
	if !receipt.OldLineHealthy {
		triggers = append(triggers, "OLD_LINE_HEALTH_REGRESSION")
	}
	if !receipt.RollbackArmed {
		triggers = append(triggers, "ROLLBACK_NOT_ARMED")
	}
	triggers = uniqueSorted(triggers)
	decision := RollbackApplyEligible
	summary := "Synthetic receipt is apply-eligible for review only; it is not production authorization."
	if slices.Contains(triggers, "ROLLBACK_NOT_ARMED") {
		decision = RollbackFailClosed
		summary = "Synthetic receipt failed closed because rollback was not armed; no production action is authorized."
	} else if len(triggers) != 0 {
		decision = RollbackRequired
		summary = "Synthetic receipt requires rollback within the named workload and attachment only; no production action is authorized."
	}
	return RollbackDecision{SchemaVersion: SchemaVersion, PlanID: receipt.PlanID, Decision: decision, Scope: RollbackScope{WorkloadID: receipt.WorkloadID, AttachmentID: receipt.AttachmentID}, Triggers: triggers, EvidenceMode: EvidenceSynthetic, ProductionAuthorized: false, Summary: summary}, nil
}
