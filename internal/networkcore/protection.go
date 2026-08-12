package networkcore

import (
	"io"
)

type ProtectedTarget struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

type ProtectedTargetSet struct {
	SchemaVersion string            `json:"schema_version"`
	TargetSetID   string            `json:"target_set_id"`
	EvidenceMode  EvidenceMode      `json:"evidence_mode"`
	Targets       []ProtectedTarget `json:"targets"`
}

type GuardStatus string

const (
	GuardPass    GuardStatus = "GUARD_PASS"
	GuardBlocked GuardStatus = "GUARD_BLOCKED"
)

type GuardResult struct {
	SchemaVersion string       `json:"schema_version"`
	PlanID        string       `json:"plan_id"`
	WorkloadID    string       `json:"workload_id"`
	Status        GuardStatus  `json:"status"`
	Codes         []string     `json:"codes"`
	EvidenceMode  EvidenceMode `json:"evidence_mode"`
	ReadOnly      bool         `json:"read_only"`
}

var requiredProtectedTargetIDs = []string{
	"old-public-tcp-443-owner", "old-haproxy-config-fingerprint", "old-caddy-service-fingerprint", "old-xray-service-fingerprint", "old-client-node-config-fingerprint",
	"ssh-tcp-22", "http-tcp-80", "udp-443", "lcsp-access-loopback-18082", "lcapi-loopback-18080", "lcgpu-gateway-loopback-18081",
	"owner-console-loopback-19982", "tailnet-existing-preferences", "system-default-route", "system-global-dns", "system-firewall-rules", "system-proxy",
}

func DecodeProtectedTargetSet(reader io.Reader) (ProtectedTargetSet, error) {
	return decodeStrict(reader, "protected_target_set", ProtectedTargetSet.Validate)
}

func SyntheticProtectedTargetSet() ProtectedTargetSet {
	targets := []ProtectedTarget{
		{ID: "old-public-tcp-443-owner", Kind: "tcp_listener", Address: "0.0.0.0", Port: 443, Protocol: "tcp"},
		{ID: "ssh-tcp-22", Kind: "tcp_listener", Address: "0.0.0.0", Port: 22, Protocol: "tcp"},
		{ID: "http-tcp-80", Kind: "tcp_listener", Address: "0.0.0.0", Port: 80, Protocol: "tcp"},
		{ID: "udp-443", Kind: "udp_listener", Address: "0.0.0.0", Port: 443, Protocol: "udp"},
		{ID: "lcsp-access-loopback-18082", Kind: "tcp_listener", Address: "127.0.0.1", Port: 18082, Protocol: "tcp"},
		{ID: "lcapi-loopback-18080", Kind: "tcp_listener", Address: "127.0.0.1", Port: 18080, Protocol: "tcp"},
		{ID: "lcgpu-gateway-loopback-18081", Kind: "tcp_listener", Address: "127.0.0.1", Port: 18081, Protocol: "tcp"},
		{ID: "owner-console-loopback-19982", Kind: "tcp_listener", Address: "127.0.0.1", Port: 19982, Protocol: "tcp"},
	}
	for _, id := range requiredProtectedTargetIDs {
		found := false
		for _, target := range targets {
			if target.ID == id {
				found = true
				break
			}
		}
		if !found {
			targets = append(targets, ProtectedTarget{ID: id, Kind: "read_only_fingerprint"})
		}
	}
	return ProtectedTargetSet{SchemaVersion: SchemaVersion, TargetSetID: "protected-platform", EvidenceMode: EvidenceSynthetic, Targets: targets}
}

func (set ProtectedTargetSet) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, set.SchemaVersion)
	issues = validateID(issues, "target_set_id", set.TargetSetID)
	if set.EvidenceMode != EvidenceSynthetic {
		issues = addIssue(issues, "EVIDENCE_MODE_INVALID", "evidence_mode", "repository protected targets must be synthetic")
	}
	if len(set.Targets) == 0 || len(set.Targets) > 64 {
		issues = addIssue(issues, "PROTECTED_TARGETS_INVALID", "targets", "a bounded protected target set is required")
	}
	seen := map[string]struct{}{}
	for _, target := range set.Targets {
		issues = validateID(issues, "targets.id", target.ID)
		if target.Kind != "tcp_listener" && target.Kind != "udp_listener" && target.Kind != "read_only_fingerprint" {
			issues = addIssue(issues, "PROTECTED_TARGET_KIND_INVALID", "targets.kind", "protected target kind is invalid")
		}
		if _, ok := seen[target.ID]; ok {
			issues = addIssue(issues, "PROTECTED_TARGET_DUPLICATE", "targets", "protected targets must be unique")
		}
		seen[target.ID] = struct{}{}
	}
	for _, required := range requiredProtectedTargetIDs {
		if _, ok := seen[required]; !ok {
			issues = addIssue(issues, "PROTECTED_TARGET_REQUIRED", "targets", "permanent protected target is missing")
		}
	}
	return issues
}

func GuardCheck(plan ContinuityPlan, protected ProtectedTargetSet) GuardResult {
	codes := []string{}
	for _, issue := range plan.Validate() {
		codes = append(codes, issue.Code)
	}
	for _, issue := range protected.Validate() {
		codes = append(codes, issue.Code)
	}
	for _, target := range protected.Targets {
		if target.Protocol == plan.Endpoint.Network && target.Port == plan.Endpoint.Port && (target.Address == "0.0.0.0" || target.Address == "::" || target.Address == plan.Endpoint.Address) {
			codes = append(codes, "PROTECTED_ENDPOINT_COLLISION")
		}
	}
	codes = uniqueSorted(codes)
	status := GuardPass
	if len(codes) != 0 {
		status = GuardBlocked
	}
	return GuardResult{SchemaVersion: SchemaVersion, PlanID: plan.PlanID, WorkloadID: plan.WorkloadID, Status: status, Codes: codes, EvidenceMode: EvidenceSynthetic, ReadOnly: true}
}
