package singbox

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dwg7318/codex-network-doctor/internal/networkcore"
)

const testOnlyMaterial = "TEST_ONLY_NOT_A_SECRET"

func TestRenderWritesProtectedCandidateAndRedactedPlan(t *testing.T) {
	draft := validDraft(t)
	material := NodeMaterial{
		SchemaVersion: networkcore.SchemaVersion,
		SecretRef:     draft.Candidate.SecretRef,
		Outbound:      json.RawMessage(`{"type":"shadowsocks","server":"192.0.2.10","server_port":443,"method":"2022-blake3-aes-128-gcm","password":"TEST_ONLY_NOT_A_SECRET"}`),
	}
	candidatePath := filepath.Join(t.TempDir(), "sing-box.json")
	plan, summary, err := RenderToFile(draft, material, candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	candidate, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(candidate, []byte(testOnlyMaterial)) {
		t.Fatal("candidate missing material")
	}
	planJSON, _, err := networkcore.CanonicalJSON(plan)
	if err != nil {
		t.Fatal(err)
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(planJSON, []byte(testOnlyMaterial)) || bytes.Contains(summaryJSON, []byte(testOnlyMaterial)) {
		t.Fatal("secret leaked")
	}
	if plan.State != networkcore.PlanReady || plan.PlanID == "" || summary.PlanID != plan.PlanID {
		t.Fatalf("plan = %#v, summary = %#v", plan, summary)
	}
}

func TestRenderCandidateMatchesRedactedGoldenSkeleton(t *testing.T) {
	draft := validDraft(t)
	material := NodeMaterial{
		SchemaVersion: networkcore.SchemaVersion,
		SecretRef:     draft.Candidate.SecretRef,
		Outbound:      json.RawMessage(`{"type":"shadowsocks","server":"192.0.2.10","server_port":443,"method":"2022-blake3-aes-128-gcm","password":"TEST_ONLY_NOT_A_SECRET"}`),
	}
	candidatePath := filepath.Join(t.TempDir(), "sing-box.json")
	if _, _, err := RenderToFile(draft, material, candidatePath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	var actual map[string]any
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatal(err)
	}
	outbounds, ok := actual["outbounds"].([]any)
	if !ok || len(outbounds) != 2 {
		t.Fatalf("outbounds = %#v", actual["outbounds"])
	}
	outbounds[1] = map[string]any{"type": "REDACTED", "tag": "lcnd-tunnel"}

	var expected any
	if err := json.Unmarshal([]byte(`{
		"log":{"level":"warn"},
		"inbounds":[{"type":"tun","tag":"lcnd-tun","interface_name":"lcnd0","address":["172.19.0.1/30"],"auto_route":true,"strict_route":true}],
		"outbounds":[{"type":"direct","tag":"direct"},{"type":"REDACTED","tag":"lcnd-tunnel"}],
		"route":{"rules":[
			{"action":"route","ip_cidr":["10.0.0.0/8"],"outbound":"direct"},
			{"action":"route","domain":["api.example.cn"],"network":"tcp","outbound":"lcnd-tunnel","port":[443]}
		],"final":"direct","auto_detect_interface":true}
	}`), &expected); err != nil {
		t.Fatal(err)
	}
	actualJSON, _, err := networkcore.CanonicalJSON(actual)
	if err != nil {
		t.Fatal(err)
	}
	expectedJSON, _, err := networkcore.CanonicalJSON(expected)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Fatalf("redacted candidate differs\ngot:  %s\nwant: %s", actualJSON, expectedJSON)
	}
}

func TestRenderRejectsExistingCandidateWithoutOverwrite(t *testing.T) {
	draft := validDraft(t)
	material := NodeMaterial{
		SchemaVersion: networkcore.SchemaVersion,
		SecretRef:     draft.Candidate.SecretRef,
		Outbound:      json.RawMessage(`{"type":"shadowsocks","server":"192.0.2.10","server_port":443,"password":"TEST_ONLY_NOT_A_SECRET"}`),
	}
	candidatePath := filepath.Join(t.TempDir(), "sing-box.json")
	if err := os.WriteFile(candidatePath, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := RenderToFile(draft, material, candidatePath)
	if err == nil || !strings.Contains(err.Error(), "CANDIDATE_CREATE_FAILED") {
		t.Fatalf("error = %v, want CANDIDATE_CREATE_FAILED", err)
	}
	data, readErr := os.ReadFile(candidatePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "preserve" {
		t.Fatalf("existing candidate was modified: %q", data)
	}
}

func TestRenderErrorDoesNotEchoCandidatePath(t *testing.T) {
	draft := validDraft(t)
	material := NodeMaterial{
		SchemaVersion: networkcore.SchemaVersion,
		SecretRef:     draft.Candidate.SecretRef,
		Outbound:      json.RawMessage(`{"type":"shadowsocks","server":"192.0.2.10","server_port":443,"password":"TEST_ONLY_NOT_A_SECRET"}`),
	}
	candidatePath := filepath.Join(t.TempDir(), testOnlyMaterial+".json")
	if err := os.WriteFile(candidatePath, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := RenderToFile(draft, material, candidatePath)
	if err == nil {
		t.Fatal("RenderToFile() error = nil")
	}
	if strings.Contains(err.Error(), testOnlyMaterial) {
		t.Fatalf("error leaked candidate path: %v", err)
	}
}

func TestDecodeMaterialRejectsUnknownField(t *testing.T) {
	_, err := DecodeMaterial(strings.NewReader(`{
		"schema_version":"1.0.0",
		"secret_ref":"lcnd-secret://nodes/cn-primary",
		"outbound":{"type":"shadowsocks"},
		"unexpected":true
	}`))
	if err == nil || err.Error() != "MATERIAL_DECODE_FAILED" {
		t.Fatalf("error = %v, want MATERIAL_DECODE_FAILED", err)
	}
}

func TestDecodeMaterialRejectsInputOverLimit(t *testing.T) {
	input := `{"schema_version":"1.0.0","secret_ref":"lcnd-secret://nodes/cn-primary","outbound":{"type":"shadowsocks"}}` + strings.Repeat(" ", maxMaterialBytes)
	_, err := DecodeMaterial(strings.NewReader(input))
	if err == nil || err.Error() != "MATERIAL_INPUT_TOO_LARGE" {
		t.Fatalf("error = %v, want MATERIAL_INPUT_TOO_LARGE", err)
	}
}

func validDraft(t *testing.T) networkcore.PlanDraft {
	t.Helper()
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
	return draft
}
