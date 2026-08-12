package networkcore

import (
	"errors"
	"io"
	"slices"
)

type ProbeClass string

const (
	ProbeDirectControl     ProbeClass = "direct_control_probe"
	ProbeDirectApplication ProbeClass = "direct_application_probe"
	ProbeEgressContinuity  ProbeClass = "egress_continuity_probe"
	ProbeEgressApplication ProbeClass = "egress_application_probe"
	ProbeProtectedTarget   ProbeClass = "protected_target_probe"
	ProbeRollback          ProbeClass = "rollback_probe"
	ProbeSecretLeak        ProbeClass = "secret_leak_probe"
)

type ProbeTarget struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ProbeDefinition struct {
	ProbeID  string          `json:"probe_id"`
	Class    ProbeClass      `json:"class"`
	Route    RouteIdentity   `json:"route"`
	Target   ProbeTarget     `json:"target"`
	Samples  int             `json:"samples"`
	Timeouts ServiceTimeouts `json:"timeouts"`
}

type ProbePlan struct {
	SchemaVersion string            `json:"schema_version"`
	PlanID        string            `json:"plan_id"`
	WorkloadID    string            `json:"workload_id"`
	Purpose       ContinuityPurpose `json:"purpose"`
	EvidenceMode  EvidenceMode      `json:"evidence_mode"`
	Probes        []ProbeDefinition `json:"probes"`
}

type ProbeResult struct {
	ProbeID          string        `json:"probe_id"`
	Class            ProbeClass    `json:"class"`
	Route            RouteIdentity `json:"route"`
	Attempted        int           `json:"attempted"`
	Succeeded        int           `json:"succeeded"`
	StatusClass      string        `json:"status_class"`
	EndpointIdentity string        `json:"endpoint_identity"`
}

type ProbeEvidence struct {
	SchemaVersion string        `json:"schema_version"`
	PlanID        string        `json:"plan_id"`
	WorkloadID    string        `json:"workload_id"`
	EvidenceMode  EvidenceMode  `json:"evidence_mode"`
	Results       []ProbeResult `json:"results"`
}

type ProbeEvaluationStatus string

const (
	ProbeSyntheticPass        ProbeEvaluationStatus = "SYNTHETIC_PASS"
	ProbeSyntheticFail        ProbeEvaluationStatus = "SYNTHETIC_FAIL"
	ProbeInsufficientEvidence ProbeEvaluationStatus = "INSUFFICIENT_EVIDENCE"
)

type ProbeEvaluation struct {
	SchemaVersion     string                `json:"schema_version"`
	PlanID            string                `json:"plan_id"`
	WorkloadID        string                `json:"workload_id"`
	EvidenceMode      EvidenceMode          `json:"evidence_mode"`
	Status            ProbeEvaluationStatus `json:"status"`
	Codes             []string              `json:"codes"`
	Summary           string                `json:"summary"`
	ProductionHealthy bool                  `json:"production_healthy"`
}

func DecodeProbePlan(reader io.Reader) (ProbePlan, error) {
	return decodeStrict(reader, "probe_plan", ProbePlan.Validate)
}

func DecodeProbeEvidence(reader io.Reader) (ProbeEvidence, error) {
	return decodeStrict(reader, "probe_evidence", ProbeEvidence.Validate)
}

func BuildDefaultProbePlan(plan ContinuityPlan) (ProbePlan, error) {
	if issues := plan.Validate(); len(issues) != 0 {
		return ProbePlan{}, ValidationError{Kind: "continuity_plan", Issues: issues}
	}
	directClass, egressClass := ProbeDirectControl, ProbeEgressContinuity
	if plan.Purpose == PurposeLCNDEgressCN {
		directClass, egressClass = ProbeDirectApplication, ProbeEgressApplication
	}
	timeouts := plan.Timeouts
	return ProbePlan{SchemaVersion: SchemaVersion, PlanID: plan.PlanID, WorkloadID: plan.WorkloadID, Purpose: plan.Purpose, EvidenceMode: EvidenceSynthetic, Probes: []ProbeDefinition{
		{ProbeID: probeID(directClass), Class: directClass, Route: RouteIdentity{ID: "direct", Class: RouteDirect}, Target: ProbeTarget{Host: "direct.example.invalid", Port: 443, Protocol: "https"}, Samples: 3, Timeouts: timeouts},
		{ProbeID: probeID(egressClass), Class: egressClass, Route: plan.Route, Target: ProbeTarget{Host: "egress.example.invalid", Port: 443, Protocol: "https"}, Samples: 3, Timeouts: timeouts},
		{ProbeID: "protected-target", Class: ProbeProtectedTarget, Route: RouteIdentity{ID: "direct", Class: RouteDirect}, Target: ProbeTarget{Host: "protected.example.invalid", Port: 443, Protocol: "https"}, Samples: 3, Timeouts: timeouts},
		{ProbeID: "secret-leak", Class: ProbeSecretLeak, Route: plan.Route, Target: ProbeTarget{Host: "redaction.example.invalid", Port: 443, Protocol: "https"}, Samples: 3, Timeouts: timeouts},
	}}, nil
}

func (plan ProbePlan) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, plan.SchemaVersion)
	issues = validateID(issues, "plan_id", plan.PlanID)
	issues = validateID(issues, "workload_id", plan.WorkloadID)
	if plan.EvidenceMode != EvidenceSynthetic {
		issues = addIssue(issues, "EVIDENCE_MODE_INVALID", "evidence_mode", "repository probes must be synthetic")
	}
	if len(plan.Probes) == 0 || len(plan.Probes) > 32 {
		issues = addIssue(issues, "PROBES_INVALID", "probes", "a bounded probe set is required")
	}
	requiredDirect, requiredEgress := ProbeDirectControl, ProbeEgressContinuity
	if plan.Purpose == PurposeLCNDEgressCN {
		requiredDirect, requiredEgress = ProbeDirectApplication, ProbeEgressApplication
		if plan.WorkloadID != "lcnd-egress-cn" {
			issues = addIssue(issues, "PURPOSE_WORKLOAD_MISMATCH", "workload_id", "application purpose requires the application egress workload")
		}
	} else if plan.Purpose == PurposeLCSPContinuityB {
		if plan.WorkloadID != "lcsp-continuity-b" {
			issues = addIssue(issues, "PURPOSE_WORKLOAD_MISMATCH", "workload_id", "continuity purpose requires the continuity workload")
		}
	} else if plan.Purpose != PurposeLCSPContinuityB {
		issues = addIssue(issues, "PURPOSE_UNSUPPORTED", "purpose", "purpose is unsupported")
	}
	foundDirect, foundEgress := false, false
	for _, probe := range plan.Probes {
		issues = validateID(issues, "probes.probe_id", probe.ProbeID)
		issues = append(issues, probe.Route.validate("probes.route")...)
		if probe.Class == requiredDirect {
			foundDirect = true
			if probe.Route.Class != RouteDirect {
				issues = addIssue(issues, "DIRECT_PROBE_ROUTE_INVALID", "probes.route", "direct probe must use direct")
			}
		}
		if probe.Class == requiredEgress {
			foundEgress = true
			expectedRoute := RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}
			if plan.Purpose == PurposeLCNDEgressCN {
				expectedRoute = RouteIdentity{ID: "lcnd-egress-cn", Class: RouteChinaEgress}
			}
			if probe.Route != expectedRoute {
				issues = addIssue(issues, "EGRESS_PROBE_ROUTE_INVALID", "probes.route", "egress probe must use named egress")
			}
		}
		if !validProbeClass(probe.Class) {
			issues = addIssue(issues, "PROBE_CLASS_UNSUPPORTED", "probes.class", "probe class is unsupported")
		}
		if (plan.Purpose == PurposeLCNDEgressCN && (probe.Class == ProbeDirectControl || probe.Class == ProbeEgressContinuity)) || (plan.Purpose == PurposeLCSPContinuityB && (probe.Class == ProbeDirectApplication || probe.Class == ProbeEgressApplication)) {
			issues = addIssue(issues, "PROBE_DOMAIN_MIXED", "probes.class", "probe class belongs to another workload domain")
		}
		if expectedRoute, routeBound := expectedProbeRoute(plan.WorkloadID, probe.Class); routeBound && probe.Route != expectedRoute {
			issues = addIssue(issues, "PROBE_ROUTE_DOMAIN_MISMATCH", "probes.route", "probe route does not match its workload domain")
		}
		if !isLowerASCIIDomain(probe.Target.Host) || probe.Target.Port < 1 || probe.Target.Port > 65535 || !slices.Contains([]string{"tcp", "tls", "https"}, probe.Target.Protocol) {
			issues = addIssue(issues, "PROBE_TARGET_INVALID", "probes.target", "probe target must be a bounded host, port, and protocol")
		}
		if probe.Samples < 3 || probe.Samples > 10 {
			issues = addIssue(issues, "PROBE_SAMPLES_INSUFFICIENT", "probes.samples", "comparative probes require three to ten samples")
		}
		issues = append(issues, probe.Timeouts.validate()...)
	}
	if !foundDirect || !foundEgress {
		issues = addIssue(issues, "DIRECT_EGRESS_PROBES_REQUIRED", "probes", "direct and egress probes must remain separate")
	}
	return issues
}

func (evidence ProbeEvidence) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, evidence.SchemaVersion)
	issues = validateID(issues, "plan_id", evidence.PlanID)
	issues = validateID(issues, "workload_id", evidence.WorkloadID)
	if evidence.EvidenceMode != EvidenceSynthetic {
		issues = addIssue(issues, "EVIDENCE_MODE_INVALID", "evidence_mode", "offline evaluator accepts synthetic evidence only")
	}
	if evidence.WorkloadID != "lcsp-continuity-b" && evidence.WorkloadID != "lcnd-egress-cn" {
		issues = addIssue(issues, "WORKLOAD_UNSUPPORTED", "workload_id", "probe evidence workload is unsupported")
	}
	if len(evidence.Results) == 0 || len(evidence.Results) > 32 {
		issues = addIssue(issues, "PROBE_RESULTS_INVALID", "results", "a bounded result set is required")
	}
	for _, result := range evidence.Results {
		issues = validateID(issues, "results.probe_id", result.ProbeID)
		issues = append(issues, result.Route.validate("results.route")...)
		if !validProbeClass(result.Class) || result.Attempted < 0 || result.Attempted > 10 || result.Succeeded < 0 || result.Succeeded > result.Attempted {
			issues = addIssue(issues, "PROBE_RESULT_INVALID", "results", "probe result is invalid")
		}
		if result.StatusClass != "success" && result.StatusClass != "failure" && result.StatusClass != "timeout" {
			issues = addIssue(issues, "PROBE_STATUS_INVALID", "results.status_class", "probe status is invalid")
		}
		if !sha256RefPattern.MatchString(result.EndpointIdentity) {
			issues = addIssue(issues, "ENDPOINT_IDENTITY_INVALID", "results.endpoint_identity", "endpoint identity must be redacted")
		}
		expectedRoute, routeBound := expectedProbeRoute(evidence.WorkloadID, result.Class)
		if routeBound && result.Route != expectedRoute {
			issues = addIssue(issues, "EGRESS_PROBE_ROUTE_INVALID", "results.route", "probe route does not match its workload domain")
		}
		if !probeClassAllowedForWorkload(evidence.WorkloadID, result.Class) {
			issues = addIssue(issues, "PROBE_DOMAIN_MIXED", "results.class", "probe class belongs to another workload domain")
		}
	}
	return issues
}

func expectedProbeRoute(workloadID string, class ProbeClass) (RouteIdentity, bool) {
	switch class {
	case ProbeDirectControl, ProbeDirectApplication, ProbeProtectedTarget, ProbeRollback:
		return RouteIdentity{ID: "direct", Class: RouteDirect}, true
	case ProbeEgressApplication:
		return RouteIdentity{ID: "lcnd-egress-cn", Class: RouteChinaEgress}, workloadID == "lcnd-egress-cn"
	case ProbeEgressContinuity:
		return RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}, workloadID == "lcsp-continuity-b"
	case ProbeSecretLeak:
		if workloadID == "lcnd-egress-cn" {
			return RouteIdentity{ID: "lcnd-egress-cn", Class: RouteChinaEgress}, true
		}
		if workloadID == "lcsp-continuity-b" {
			return RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}, true
		}
	}
	return RouteIdentity{}, false
}

func probeClassAllowedForWorkload(workloadID string, class ProbeClass) bool {
	switch workloadID {
	case "lcnd-egress-cn":
		return class != ProbeDirectControl && class != ProbeEgressContinuity
	case "lcsp-continuity-b":
		return class != ProbeDirectApplication && class != ProbeEgressApplication
	default:
		return false
	}
}

func EvaluateProbeEvidence(plan ContinuityPlan, evidence ProbeEvidence) (ProbeEvaluation, error) {
	if issues := plan.Validate(); len(issues) != 0 {
		return ProbeEvaluation{}, ValidationError{Kind: "continuity_plan", Issues: issues}
	}
	if issues := evidence.Validate(); len(issues) != 0 {
		return ProbeEvaluation{}, ValidationError{Kind: "probe_evidence", Issues: issues}
	}
	if evidence.PlanID != plan.PlanID || evidence.WorkloadID != plan.WorkloadID {
		return ProbeEvaluation{}, errors.New("PROBE_PLAN_MISMATCH")
	}
	codes := []string{}
	insufficient := false
	requiredDirect, requiredEgress := ProbeDirectControl, ProbeEgressContinuity
	if plan.Purpose == PurposeLCNDEgressCN {
		requiredDirect, requiredEgress = ProbeDirectApplication, ProbeEgressApplication
	}
	foundDirect, foundEgress := false, false
	for _, result := range evidence.Results {
		if result.Class == requiredDirect {
			foundDirect = true
		}
		if result.Class == requiredEgress {
			foundEgress = true
		}
		expected := RouteIdentity{ID: "direct", Class: RouteDirect}
		if result.Class == ProbeEgressApplication || result.Class == ProbeEgressContinuity || result.Class == ProbeSecretLeak {
			expected = plan.Route
		}
		if result.Route != expected {
			codes = append(codes, "ROUTE_IDENTITY_MISMATCH")
		}
		if result.Attempted < 3 {
			insufficient = true
			codes = append(codes, "INSUFFICIENT_EVIDENCE")
		} else if result.Succeeded < 3 || result.StatusClass != "success" {
			codes = append(codes, "PROBE_QUORUM_FAILED")
		}
	}
	if !foundDirect || !foundEgress {
		insufficient = true
		codes = append(codes, "DIRECT_EGRESS_EVIDENCE_REQUIRED", "INSUFFICIENT_EVIDENCE")
	}
	codes = uniqueSorted(codes)
	status := ProbeSyntheticPass
	summary := "Synthetic offline probe evidence satisfied the declared contract; no live or production health claim was made."
	if containsAny(codes, "ROUTE_IDENTITY_MISMATCH", "PROBE_QUORUM_FAILED") {
		status = ProbeSyntheticFail
		summary = "Synthetic offline probe evidence failed closed; no live or production health claim was made."
	} else if insufficient {
		status = ProbeInsufficientEvidence
		summary = "Synthetic offline probe evidence was insufficient; no live or production health claim was made."
	}
	return ProbeEvaluation{SchemaVersion: SchemaVersion, PlanID: plan.PlanID, WorkloadID: plan.WorkloadID, EvidenceMode: EvidenceSynthetic, Status: status, Codes: codes, Summary: summary, ProductionHealthy: false}, nil
}

func probeID(class ProbeClass) string {
	return string(class[:len(class)-len("_probe")])
}

func validProbeClass(class ProbeClass) bool {
	return slices.Contains([]ProbeClass{ProbeDirectControl, ProbeDirectApplication, ProbeEgressContinuity, ProbeEgressApplication, ProbeProtectedTarget, ProbeRollback, ProbeSecretLeak}, class)
}

func uniqueSorted(values []string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}

func containsAny(values []string, expected ...string) bool {
	for _, value := range values {
		if slices.Contains(expected, value) {
			return true
		}
	}
	return false
}
