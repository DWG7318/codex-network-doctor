package networkcore

import "testing"

func TestProbePlanSeparatesDirectAndApplicationEgress(t *testing.T) {
	plan := validContinuityPlan()
	probes, err := BuildDefaultProbePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if issues := probes.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
	if len(probes.Probes) != 4 || probes.Probes[0].Route.Class != RouteDirect || probes.Probes[1].Route.Class != RouteChinaEgress {
		t.Fatalf("probes = %#v", probes.Probes)
	}
}

func TestProbePlanRejectsInsufficientSamplesAndFullURL(t *testing.T) {
	probes, err := BuildDefaultProbePlan(validContinuityPlan())
	if err != nil {
		t.Fatal(err)
	}
	probes.Probes[0].Samples = 2
	probes.Probes[1].Target.Host = "https://api.example.invalid/path?token=DO_NOT_ECHO"
	issues := probes.Validate()
	if !hasIssue(issues, "PROBE_SAMPLES_INSUFFICIENT") || !hasIssue(issues, "PROBE_TARGET_INVALID") {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestProbePlanRejectsMixedWorkloadProbeClasses(t *testing.T) {
	probes, err := BuildDefaultProbePlan(validContinuityPlan())
	if err != nil {
		t.Fatal(err)
	}
	probes.Probes = append(probes.Probes, ProbeDefinition{ProbeID: "egress-continuity", Class: ProbeEgressContinuity, Route: probes.Probes[1].Route, Target: ProbeTarget{Host: "continuity.example.invalid", Port: 443, Protocol: "https"}, Samples: 3, Timeouts: probes.Probes[1].Timeouts})
	if !hasIssue(probes.Validate(), "PROBE_DOMAIN_MIXED") {
		t.Fatal("application plan accepted a continuity probe")
	}
}

func TestEvaluateProbeEvidenceReturnsInsufficientEvidenceWithoutLiveClaim(t *testing.T) {
	plan := validContinuityPlan()
	evidence := validProbeEvidence(plan)
	evidence.Results[1].Attempted = 2
	evidence.Results[1].Succeeded = 2
	report, err := EvaluateProbeEvidence(plan, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != ProbeInsufficientEvidence || report.EvidenceMode != EvidenceSynthetic {
		t.Fatalf("report = %#v", report)
	}
	if report.ProductionHealthy {
		t.Fatal("synthetic evaluation claimed production health")
	}
}

func TestEvaluateProbeEvidenceFailsClosedOnRouteIdentityDrift(t *testing.T) {
	plan := validContinuityPlan()
	evidence := validProbeEvidence(plan)
	evidence.Results[1].Route = RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}
	report, err := EvaluateProbeEvidence(plan, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != ProbeSyntheticFail || !containsString(report.Codes, "ROUTE_IDENTITY_MISMATCH") {
		t.Fatalf("report = %#v", report)
	}
}

func TestEvaluateProbeEvidenceRequiresBothDirectAndEgressLayers(t *testing.T) {
	plan := validContinuityPlan()
	evidence := validProbeEvidence(plan)
	evidence.Results = evidence.Results[1:]
	report, err := EvaluateProbeEvidence(plan, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != ProbeInsufficientEvidence || !containsString(report.Codes, "DIRECT_EGRESS_EVIDENCE_REQUIRED") {
		t.Fatalf("report = %#v", report)
	}
}

func validProbeEvidence(plan ContinuityPlan) ProbeEvidence {
	return ProbeEvidence{SchemaVersion: SchemaVersion, PlanID: plan.PlanID, WorkloadID: plan.WorkloadID, EvidenceMode: EvidenceSynthetic, Results: []ProbeResult{
		{ProbeID: "direct-application", Class: ProbeDirectApplication, Route: RouteIdentity{ID: "direct", Class: RouteDirect}, Attempted: 3, Succeeded: 3, StatusClass: "success", EndpointIdentity: "sha256:" + repeatHex("1")},
		{ProbeID: "egress-application", Class: ProbeEgressApplication, Route: plan.Route, Attempted: 3, Succeeded: 3, StatusClass: "success", EndpointIdentity: "sha256:" + repeatHex("2")},
	}}
}

func repeatHex(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return result[:64]
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
