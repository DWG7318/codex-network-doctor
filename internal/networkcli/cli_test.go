package networkcli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dwg7318/codex-network-doctor/internal/networkcore"
)

const cliTestMaterial = "TEST_ONLY_NOT_A_SECRET"

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{"version"}, &stdout, &stderr, "0.3.0")
	if exit != 0 || stdout.String() != "cnd-network 0.3.0\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{"unknown"}, &stdout, &stderr, "0.3.0")
	if exit != 2 || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
}

func TestRunValidateRejectsUnknownFieldWithoutEchoingValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node.json")
	data := `{"schema_version":"1.0.0","node_id":"cn-primary","display_name":"Synthetic","direction":"china_egress","engine":"sing-box","secret_ref":"lcnd-secret://nodes/cn-primary","capabilities":{"tcp":true,"udp":true,"ipv4":true,"ipv6":false},"enabled":true,"password":"DO_NOT_ECHO_THIS_VALUE"}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{"validate", "--kind", "node", "--input", path, "--format", "json"}, &stdout, &stderr, "0.3.0")
	if exit != 2 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "DO_NOT_ECHO_THIS_VALUE") || strings.Contains(stderr.String(), "DO_NOT_ECHO_THIS_VALUE") {
		t.Fatal("invalid input value leaked to process output")
	}
}

func TestRunRenderDoesNotPrintMaterial(t *testing.T) {
	paths := writeSyntheticInputs(t)
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{
		"render", "--draft", paths.Draft, "--material", paths.Material,
		"--candidate", paths.Candidate, "--plan", paths.Plan, "--format", "json",
	}, &stdout, &stderr, "0.3.0")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
	}
	if strings.Contains(stdout.String(), cliTestMaterial) || strings.Contains(stderr.String(), cliTestMaterial) {
		t.Fatal("secret leaked to process output")
	}
	plan, err := os.ReadFile(paths.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(plan, []byte(cliTestMaterial)) {
		t.Fatal("secret leaked to finalized plan")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(paths.Candidate)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("candidate mode = %o", info.Mode().Perm())
		}
	}
}

func TestRunRenderPreflightsAllOutputs(t *testing.T) {
	paths := writeSyntheticInputs(t)
	if err := os.WriteFile(paths.Plan, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{
		"render", "--draft", paths.Draft, "--material", paths.Material,
		"--candidate", paths.Candidate, "--plan", paths.Plan,
	}, &stdout, &stderr, "0.3.0")
	if exit != 1 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(paths.Candidate); !os.IsNotExist(err) {
		t.Fatalf("candidate should not be created, stat error = %v", err)
	}
	data, err := os.ReadFile(paths.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "preserve" {
		t.Fatalf("existing plan was modified: %q", data)
	}
}

func TestRunSimulateRejectsOversizedPlan(t *testing.T) {
	root := t.TempDir()
	planPath := filepath.Join(root, "plan.json")
	outputPath := filepath.Join(root, "receipt.json")
	plan := networkcore.NetworkChangePlan{
		SchemaVersion: networkcore.SchemaVersion,
		PlanID:        strings.Repeat("a", 64), State: networkcore.PlanReady,
		ToolVersion: "0.3.0", Platform: "ubuntu-24.04", Architecture: "amd64",
		ProfileID: "overseas-host", NodeID: "cn-primary", NodeDirection: networkcore.DirectionChinaEgress,
		DefaultOutbound: networkcore.OutboundDirect, TunnelOutbound: networkcore.OutboundTunnel,
		Candidate:       networkcore.CandidateSpec{Engine: "sing-box", EngineVersion: networkcore.SingBoxVersion, SecretRef: "lcnd-secret://nodes/cn-primary"},
		CandidateSHA256: strings.Repeat("b", 64), SimulationOnly: true,
	}
	data, _, err := networkcore.CanonicalJSON(plan)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, bytes.Repeat([]byte(" "), maxInputBytes)...)
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{
		"simulate", "--plan", planPath, "--scenario", "success", "--output", outputPath,
	}, &stdout, &stderr, "0.3.0")
	if exit != 2 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("receipt should not be created, stat error = %v", err)
	}
}

func TestRunSyntheticWorkflowIsDeterministicAndSynthetic(t *testing.T) {
	paths := writeContractInputs(t)
	draftA := filepath.Join(paths.Root, "draft-a.json")
	draftB := filepath.Join(paths.Root, "draft-b.json")
	for _, output := range []string{draftA, draftB} {
		var stdout, stderr bytes.Buffer
		exit := Run(context.Background(), []string{
			"plan", "--node", paths.Node, "--targets", paths.Targets,
			"--protected", paths.Protected, "--profile", paths.Profile,
			"--output", output, "--format", "json",
		}, &stdout, &stderr, "0.3.0")
		if exit != 0 {
			t.Fatalf("plan exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
		}
	}
	first, err := os.ReadFile(draftA)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(draftB)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("plan output is not deterministic")
	}

	renderPaths := writeSyntheticInputsFromDraft(t, paths.Root, draftA)
	var renderOut, renderErr bytes.Buffer
	if exit := Run(context.Background(), []string{
		"render", "--draft", renderPaths.Draft, "--material", renderPaths.Material,
		"--candidate", renderPaths.Candidate, "--plan", renderPaths.Plan, "--format", "json",
	}, &renderOut, &renderErr, "0.3.0"); exit != 0 {
		t.Fatalf("render exit=%d stdout=%q stderr=%q", exit, renderOut.String(), renderErr.String())
	}

	receiptPath := filepath.Join(paths.Root, "receipt.json")
	var simulateOut, simulateErr bytes.Buffer
	if exit := Run(context.Background(), []string{
		"simulate", "--plan", renderPaths.Plan, "--scenario", "success", "--output", receiptPath, "--format", "json",
	}, &simulateOut, &simulateErr, "0.3.0"); exit != 0 {
		t.Fatalf("simulate exit=%d stdout=%q stderr=%q", exit, simulateOut.String(), simulateErr.String())
	}

	reportPath := filepath.Join(paths.Root, "report.json")
	var reportOut, reportErr bytes.Buffer
	if exit := Run(context.Background(), []string{
		"report", "--plan", renderPaths.Plan, "--receipt", receiptPath, "--output", reportPath, "--format", "json",
	}, &reportOut, &reportErr, "0.3.0"); exit != 0 {
		t.Fatalf("report exit=%d stdout=%q stderr=%q", exit, reportOut.String(), reportErr.String())
	}
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report networkcore.NetworkHealthReport
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatal(err)
	}
	if report.EvidenceMode != networkcore.EvidenceSynthetic || report.Status != networkcore.HealthSimulatedPass {
		t.Fatalf("report = %#v", report)
	}
}

type syntheticPaths struct {
	Draft     string
	Material  string
	Candidate string
	Plan      string
}

func writeSyntheticInputs(t *testing.T) syntheticPaths {
	t.Helper()
	root := t.TempDir()
	draft, err := networkcore.BuildDraft(
		networkcore.NodeDescriptor{SchemaVersion: networkcore.SchemaVersion, NodeID: "cn-primary", DisplayName: "Synthetic", Direction: networkcore.DirectionChinaEgress, Engine: "sing-box", SecretRef: "lcnd-secret://nodes/cn-primary", Capabilities: networkcore.Capabilities{TCP: true, IPv4: true}, Enabled: true},
		networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "domestic-services", Revision: 1, Purpose: "synthetic", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetDomain, Value: "api.example.cn", Protocol: "tcp", Ports: []int{443}}}},
		networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "management", Revision: 1, Purpose: "protected", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetIPCIDR, Value: "10.0.0.0/8"}}},
		networkcore.RouteProfile{SchemaVersion: networkcore.SchemaVersion, ProfileID: "overseas-host", Name: networkcore.ProfileGlobalDirectDomesticTunnel, NodeID: "cn-primary", TargetSetID: "domestic-services", ProtectedTargetSetID: "management"},
		"0.3.0",
	)
	if err != nil {
		t.Fatal(err)
	}
	draftData, _, err := networkcore.CanonicalJSON(draft)
	if err != nil {
		t.Fatal(err)
	}
	draftPath := filepath.Join(root, "draft.json")
	if err := os.WriteFile(draftPath, draftData, 0o600); err != nil {
		t.Fatal(err)
	}
	return writeSyntheticInputsFromDraft(t, root, draftPath)
}

func writeSyntheticInputsFromDraft(t *testing.T, root, draftPath string) syntheticPaths {
	t.Helper()
	materialPath := filepath.Join(root, "material.json")
	material := `{"schema_version":"1.0.0","secret_ref":"lcnd-secret://nodes/cn-primary","outbound":{"type":"shadowsocks","server":"192.0.2.10","server_port":443,"method":"2022-blake3-aes-128-gcm","password":"TEST_ONLY_NOT_A_SECRET"}}`
	if err := os.WriteFile(materialPath, []byte(material), 0o600); err != nil {
		t.Fatal(err)
	}
	return syntheticPaths{
		Draft:     draftPath,
		Material:  materialPath,
		Candidate: filepath.Join(root, "candidate.json"),
		Plan:      filepath.Join(root, "plan.json"),
	}
}

type contractPaths struct {
	Root      string
	Node      string
	Targets   string
	Protected string
	Profile   string
}

func writeContractInputs(t *testing.T) contractPaths {
	t.Helper()
	root := t.TempDir()
	paths := contractPaths{
		Root:      root,
		Node:      filepath.Join(root, "node.json"),
		Targets:   filepath.Join(root, "targets.json"),
		Protected: filepath.Join(root, "protected.json"),
		Profile:   filepath.Join(root, "profile.json"),
	}
	writeCanonical(t, paths.Node, networkcore.NodeDescriptor{SchemaVersion: networkcore.SchemaVersion, NodeID: "cn-primary", DisplayName: "Synthetic", Direction: networkcore.DirectionChinaEgress, Engine: "sing-box", SecretRef: "lcnd-secret://nodes/cn-primary", Capabilities: networkcore.Capabilities{TCP: true, UDP: true, IPv4: true}, Enabled: true})
	writeCanonical(t, paths.Targets, networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "domestic-services", Revision: 1, Purpose: "synthetic", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetDomain, Value: "api.example.cn", Protocol: "tcp", Ports: []int{443}}}})
	writeCanonical(t, paths.Protected, networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "management", Revision: 1, Purpose: "protected", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetIPCIDR, Value: "10.0.0.0/8"}}})
	writeCanonical(t, paths.Profile, networkcore.RouteProfile{SchemaVersion: networkcore.SchemaVersion, ProfileID: "overseas-host", Name: networkcore.ProfileGlobalDirectDomesticTunnel, NodeID: "cn-primary", TargetSetID: "domestic-services", ProtectedTargetSetID: "management"})
	return paths
}

func writeCanonical(t *testing.T, path string, value any) {
	t.Helper()
	data, _, err := networkcore.CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
