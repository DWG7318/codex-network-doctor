package networkcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dwg7318/codex-network-doctor/internal/networkcore"
)

func TestContinuityValidateAndCandidateSummaryAreDeterministic(t *testing.T) {
	root := t.TempDir()
	planPath := filepath.Join(root, "plan.json")
	writeCanonical(t, planPath, cliContinuityPlan())

	var stdout, stderr bytes.Buffer
	if exit := Run(context.Background(), []string{"continuity", "validate", "--input", planPath, "--format", "json"}, &stdout, &stderr, "0.4.0"); exit != 0 {
		t.Fatalf("validate exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}

	outputs := []string{filepath.Join(root, "summary-a.json"), filepath.Join(root, "summary-b.json")}
	for _, output := range outputs {
		stdout.Reset()
		stderr.Reset()
		if exit := Run(context.Background(), []string{"continuity", "candidate-summary", "--input", planPath, "--output", output, "--format", "json"}, &stdout, &stderr, "0.4.0"); exit != 0 {
			t.Fatalf("candidate-summary exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
		}
	}
	first, _ := os.ReadFile(outputs[0])
	second, _ := os.ReadFile(outputs[1])
	if !bytes.Equal(first, second) {
		t.Fatal("candidate summaries are not deterministic")
	}
	if bytes.Contains(first, []byte("password")) || bytes.Contains(first, []byte("token")) {
		t.Fatal("candidate summary contains a secret field")
	}

	stdout.Reset()
	stderr.Reset()
	if exit := Run(context.Background(), []string{"continuity", "candidate-summary", "--input", planPath, "--output", outputs[0]}, &stdout, &stderr, "0.4.0"); exit != 1 {
		t.Fatalf("overwrite exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
}

func TestContinuityOfflineWorkflowProducesSyntheticScopedArtifacts(t *testing.T) {
	root := t.TempDir()
	plan := cliContinuityPlan()
	planPath := filepath.Join(root, "plan.json")
	writeCanonical(t, planPath, plan)

	probePlanPath := filepath.Join(root, "probes.json")
	runContinuityCommand(t, []string{"continuity", "probe-plan", "--input", planPath, "--output", probePlanPath})

	evidence := networkcore.ProbeEvidence{SchemaVersion: networkcore.SchemaVersion, PlanID: plan.PlanID, WorkloadID: plan.WorkloadID, EvidenceMode: networkcore.EvidenceSynthetic, Results: []networkcore.ProbeResult{
		{ProbeID: "direct-application", Class: networkcore.ProbeDirectApplication, Route: networkcore.RouteIdentity{ID: "direct", Class: networkcore.RouteDirect}, Attempted: 2, Succeeded: 2, StatusClass: "success", EndpointIdentity: "sha256:" + strings.Repeat("1", 64)},
		{ProbeID: "egress-application", Class: networkcore.ProbeEgressApplication, Route: plan.Route, Attempted: 2, Succeeded: 2, StatusClass: "success", EndpointIdentity: "sha256:" + strings.Repeat("2", 64)},
	}}
	evidencePath := filepath.Join(root, "evidence.json")
	writeCanonical(t, evidencePath, evidence)
	evaluationPath := filepath.Join(root, "evaluation.json")
	runContinuityCommand(t, []string{"continuity", "evaluate", "--plan", planPath, "--evidence", evidencePath, "--output", evaluationPath})
	evaluationData, _ := os.ReadFile(evaluationPath)
	if !bytes.Contains(evaluationData, []byte("INSUFFICIENT_EVIDENCE")) || !bytes.Contains(evaluationData, []byte("synthetic")) {
		t.Fatalf("evaluation = %s", evaluationData)
	}

	protectedPath := filepath.Join(root, "protected.json")
	writeCanonical(t, protectedPath, networkcore.SyntheticProtectedTargetSet())
	guardPath := filepath.Join(root, "guard.json")
	runContinuityCommand(t, []string{"continuity", "guard-check", "--input", planPath, "--protected", protectedPath, "--output", guardPath})

	receiptPath := filepath.Join(root, "receipt.json")
	digest := plan.Engine.CandidateSHA256
	writeCanonical(t, receiptPath, networkcore.RollbackReceipt{SchemaVersion: networkcore.SchemaVersion, PlanID: plan.PlanID, WorkloadID: plan.WorkloadID, RollbackDomain: plan.RollbackDomain, AttachmentID: "lcapi-domestic-synthetic", ExpectedCandidateSHA256: digest, ObservedCandidateSHA256: digest, RollbackArmed: true, ProtectedTargetsUnchanged: true, OldLineHealthy: true, Triggers: []string{}, EvidenceMode: networkcore.EvidenceSynthetic})
	decisionPath := filepath.Join(root, "decision.json")
	runContinuityCommand(t, []string{"continuity", "rollback-evaluate", "--receipt", receiptPath, "--output", decisionPath})
	decisionData, _ := os.ReadFile(decisionPath)
	if !bytes.Contains(decisionData, []byte("APPLY_ELIGIBLE")) || bytes.Contains(decisionData, []byte(`"production_authorized":true`)) {
		t.Fatalf("decision = %s", decisionData)
	}

	attachmentPath := filepath.Join(root, "attachment-input.json")
	writeCanonical(t, attachmentPath, cliApplicationAttachment())
	attachmentOutput := filepath.Join(root, "attachment.json")
	runContinuityCommand(t, []string{"continuity", "attachment-render", "--input", attachmentPath, "--output", attachmentOutput})
}

func TestContinuityCLIRejectsMissingAndInvalidInputsWithStableExitCodes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{"continuity", "validate", "--input", filepath.Join(t.TempDir(), "missing.json")}, &stdout, &stderr, "0.4.0")
	if exit != 1 {
		t.Fatalf("missing exit=%d stderr=%q", exit, stderr.String())
	}

	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"1.0.0","token":"DO_NOT_ECHO"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	exit = Run(context.Background(), []string{"continuity", "validate", "--input", path}, &stdout, &stderr, "0.4.0")
	if exit != 2 || strings.Contains(stdout.String()+stderr.String(), "DO_NOT_ECHO") {
		t.Fatalf("invalid exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
}

func runContinuityCommand(t *testing.T, args []string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if exit := Run(context.Background(), args, &stdout, &stderr, "0.4.0"); exit != 0 {
		t.Fatalf("Run(%v) exit=%d stdout=%q stderr=%q", args, exit, stdout.String(), stderr.String())
	}
}

func cliContinuityPlan() networkcore.ContinuityPlan {
	digest := "sha256:" + strings.Repeat("a", 64)
	return networkcore.ContinuityPlan{SchemaVersion: networkcore.SchemaVersion, PlanID: "lcnd-plan-20260812-egress-cn-synthetic", Purpose: networkcore.PurposeLCNDEgressCN, TargetPlatform: "ubuntu-24.04-amd64", HostRef: "lcsp://hosts/owner-approved-primary", WorkloadID: "lcnd-egress-cn", Route: networkcore.RouteIdentity{ID: "lcnd-egress-cn", Class: networkcore.RouteChinaEgress}, Endpoint: networkcore.ServiceEndpoint{Ref: "lcnd-endpoint://loopback/egress-cn", Network: "tcp", Address: "127.0.0.1", Port: 1080, Scope: networkcore.EndpointLoopback}, Authentication: networkcore.ServiceAuthentication{Required: true, Consumers: []networkcore.ConsumerCredentialRef{{ConsumerID: "lcapi", IdentityRef: "lcnd-identity://consumers/lcapi", CredentialRef: "lcnd-secret://lcsp/egress-cn/lcapi"}}}, Timeouts: networkcore.ServiceTimeouts{ConnectMilliseconds: 3000, RequestMilliseconds: 30000, IdleMilliseconds: 60000}, FailurePolicy: networkcore.FailurePolicy{Mode: networkcore.FailureApprovedDirect, ApprovedRouteID: "direct"}, Engine: networkcore.EngineCandidate{Name: "sing-box", Version: networkcore.SingBoxVersion, CandidateSHA256: digest}, ProtectedTargetsRef: "lcnd://samples/network/protected-platform.synthetic.json", ProbePlanRef: "lcnd://samples/network/egress-cn-probes.synthetic.json", ReceiptDomain: "lcnd-egress-cn", RollbackDomain: "lcnd-egress-cn", SeparateFrom: []string{"old-public-tcp-443", "lcsp-continuity-b"}, HostImpact: networkcore.HostNetworkImpact{}, ExecutionAuthority: "lcsp_required", ProductionAuthorized: false, EvidenceMode: networkcore.EvidenceSynthetic}
}

func cliApplicationAttachment() networkcore.ApplicationAttachment {
	return networkcore.ApplicationAttachment{SchemaVersion: networkcore.SchemaVersion, AttachmentID: "lcapi-domestic-synthetic", ConsumerID: "lcapi", ConsumerIdentityRef: "lcnd-identity://consumers/lcapi", Selectors: []networkcore.ApplicationSelector{{Provider: "synthetic-cn-provider", Model: "synthetic-model", Channel: "chat", TargetSetRef: "lcnd-targets://lcapi/domestic-synthetic"}}, Route: networkcore.RouteIdentity{ID: "lcnd-egress-cn", Class: networkcore.RouteChinaEgress}, EndpointRef: "lcnd-endpoint://loopback/egress-cn", CredentialRef: "lcnd-secret://lcsp/egress-cn/lcapi", Timeouts: networkcore.ServiceTimeouts{ConnectMilliseconds: 3000, RequestMilliseconds: 30000, IdleMilliseconds: 60000}, FailurePolicy: networkcore.FailurePolicy{Mode: networkcore.FailureApprovedDirect, ApprovedRouteID: "direct"}, BusinessProbeRef: "lcnd-probe://lcapi/domestic-synthetic", Rollback: networkcore.AttachmentRollback{DetachOnly: true, RestoreRouteID: "direct"}, ProductionAuthorized: false}
}
