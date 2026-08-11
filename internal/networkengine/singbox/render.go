package singbox

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/dwg7318/codex-network-doctor/internal/networkcore"
)

func RenderToFile(draft networkcore.PlanDraft, material NodeMaterial, path string) (networkcore.NetworkChangePlan, RenderSummary, error) {
	if draft.State != networkcore.PlanValidated || !draft.SimulationOnly {
		return networkcore.NetworkChangePlan{}, RenderSummary{}, errors.New("DRAFT_STATE_INVALID")
	}
	if material.SchemaVersion != networkcore.SchemaVersion || material.SecretRef != draft.Candidate.SecretRef {
		return networkcore.NetworkChangePlan{}, RenderSummary{}, errors.New("MATERIAL_REFERENCE_MISMATCH")
	}
	outbound, err := decodeOutbound(material.Outbound)
	if err != nil {
		return networkcore.NetworkChangePlan{}, RenderSummary{}, err
	}
	outbound["tag"] = string(networkcore.OutboundTunnel)
	config := buildConfig(draft, outbound)
	candidate, _, err := networkcore.CanonicalJSON(config)
	if err != nil {
		return networkcore.NetworkChangePlan{}, RenderSummary{}, err
	}
	if err := writeProtected(path, candidate); err != nil {
		return networkcore.NetworkChangePlan{}, RenderSummary{}, err
	}

	candidateSHA := sha256Hex(candidate)
	plan := finalizePlan(draft, candidateSHA)
	planID, err := planDigest(plan)
	if err != nil {
		_ = os.Remove(path)
		return networkcore.NetworkChangePlan{}, RenderSummary{}, err
	}
	plan.PlanID = planID
	return plan, RenderSummary{
		SchemaVersion:   networkcore.SchemaVersion,
		PlanID:          plan.PlanID,
		CandidateSHA256: candidateSHA,
		CandidateBytes:  len(candidate),
	}, nil
}

func decodeOutbound(raw json.RawMessage) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var outbound map[string]any
	if err := decoder.Decode(&outbound); err != nil {
		return nil, errors.New("OUTBOUND_DECODE_FAILED")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, errors.New("OUTBOUND_TRAILING_DATA")
	}
	typeName, ok := outbound["type"].(string)
	if !ok || typeName == "" || typeName == "direct" || typeName == "block" {
		return nil, errors.New("OUTBOUND_TYPE_INVALID")
	}
	delete(outbound, "tag")
	return outbound, nil
}

func buildConfig(draft networkcore.PlanDraft, tunnel map[string]any) map[string]any {
	rules := make([]map[string]any, 0, len(draft.ProtectedDecisions)+len(draft.Decisions))
	for _, decision := range draft.ProtectedDecisions {
		rules = append(rules, ruleForDecision(decision))
	}
	for _, decision := range draft.Decisions {
		rules = append(rules, ruleForDecision(decision))
	}
	return map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{map[string]any{
			"type":           "tun",
			"tag":            "lcnd-tun",
			"interface_name": "lcnd0",
			"address":        []string{"172.19.0.1/30"},
			"auto_route":     true,
			"strict_route":   true,
		}},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}, tunnel},
		"route": map[string]any{
			"rules":                 rules,
			"final":                 "direct",
			"auto_detect_interface": true,
		},
	}
}

func ruleForDecision(decision networkcore.RouteDecision) map[string]any {
	rule := map[string]any{"action": "route", "outbound": string(decision.Outbound)}
	switch decision.Rule.Kind {
	case networkcore.TargetDomain:
		rule["domain"] = []string{decision.Rule.Value}
	case networkcore.TargetDomainSuffix:
		rule["domain_suffix"] = []string{decision.Rule.Value}
	case networkcore.TargetIPCIDR:
		rule["ip_cidr"] = []string{decision.Rule.Value}
	}
	if decision.Rule.Protocol != "" {
		rule["network"] = decision.Rule.Protocol
	}
	if len(decision.Rule.Ports) != 0 {
		rule["port"] = append([]int(nil), decision.Rule.Ports...)
	}
	return rule
}

func writeProtected(path string, data []byte) error {
	if path == "" || path == "-" {
		return errors.New("CANDIDATE_PATH_INVALID")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("CANDIDATE_CREATE_FAILED")
	}
	written := false
	defer func() {
		if !written {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return errors.New("CANDIDATE_WRITE_FAILED")
	}
	if err := file.Close(); err != nil {
		return errors.New("CANDIDATE_CLOSE_FAILED")
	}
	written = true
	return nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func finalizePlan(draft networkcore.PlanDraft, candidateSHA string) networkcore.NetworkChangePlan {
	return networkcore.NetworkChangePlan{
		SchemaVersion:      draft.SchemaVersion,
		State:              networkcore.PlanReady,
		ToolVersion:        draft.ToolVersion,
		Platform:           draft.Platform,
		Architecture:       draft.Architecture,
		ProfileID:          draft.ProfileID,
		NodeID:             draft.NodeID,
		NodeDirection:      draft.NodeDirection,
		DefaultOutbound:    draft.DefaultOutbound,
		TunnelOutbound:     draft.TunnelOutbound,
		Decisions:          append([]networkcore.RouteDecision(nil), draft.Decisions...),
		ProtectedDecisions: append([]networkcore.RouteDecision(nil), draft.ProtectedDecisions...),
		Candidate:          draft.Candidate,
		CandidateSHA256:    candidateSHA,
		SimulationOnly:     true,
	}
}

func planDigest(plan networkcore.NetworkChangePlan) (string, error) {
	plan.PlanID = ""
	_, digest, err := networkcore.CanonicalJSON(plan)
	if err != nil {
		return "", err
	}
	return digest, nil
}
