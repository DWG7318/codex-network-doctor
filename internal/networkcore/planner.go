package networkcore

import (
	"errors"
	"net/netip"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func BuildDraft(node NodeDescriptor, targets, protected TargetSet, profile RouteProfile, toolVersion string) (PlanDraft, error) {
	if issues := ValidateProfileForNode(profile, node); len(issues) != 0 {
		return PlanDraft{}, ValidationError{Kind: "profile", Issues: issues}
	}
	if !node.Enabled {
		return PlanDraft{}, errors.New("NODE_DISABLED")
	}
	if issues := targets.Validate(); len(issues) != 0 {
		return PlanDraft{}, ValidationError{Kind: "target_set", Issues: issues}
	}
	if issues := protected.Validate(); len(issues) != 0 {
		return PlanDraft{}, ValidationError{Kind: "protected_target_set", Issues: issues}
	}
	if profile.TargetSetID != targets.TargetSetID || profile.ProtectedTargetSetID != protected.TargetSetID {
		return PlanDraft{}, errors.New("PROFILE_TARGET_REFERENCE_MISMATCH")
	}
	if strings.TrimSpace(toolVersion) == "" {
		return PlanDraft{}, errors.New("TOOL_VERSION_REQUIRED")
	}

	normalizedTargets, err := normalizeAndSortTargets(targets.Targets)
	if err != nil {
		return PlanDraft{}, err
	}
	normalizedProtected, err := normalizeAndSortTargets(protected.Targets)
	if err != nil {
		return PlanDraft{}, err
	}
	if overlaps(normalizedTargets, normalizedProtected) {
		return PlanDraft{}, errors.New("TARGET_OVERLAP")
	}

	return PlanDraft{
		SchemaVersion:      SchemaVersion,
		State:              PlanValidated,
		ToolVersion:        toolVersion,
		Platform:           "ubuntu-24.04",
		Architecture:       "amd64",
		ProfileID:          profile.ProfileID,
		NodeID:             node.NodeID,
		NodeDirection:      node.Direction,
		DefaultOutbound:    OutboundDirect,
		TunnelOutbound:     OutboundTunnel,
		Decisions:          decisions(targets.TargetSetID, normalizedTargets, OutboundTunnel),
		ProtectedDecisions: decisions(protected.TargetSetID, normalizedProtected, OutboundDirect),
		Candidate: CandidateSpec{
			Engine:        "sing-box",
			EngineVersion: SingBoxVersion,
			SecretRef:     node.SecretRef,
		},
		SimulationOnly: true,
	}, nil
}

func normalizeAndSortTargets(input []TargetRule) ([]TargetRule, error) {
	result := append([]TargetRule(nil), input...)
	for index := range result {
		rule := &result[index]
		rule.Protocol = strings.ToLower(strings.TrimSpace(rule.Protocol))
		switch rule.Kind {
		case TargetDomain, TargetDomainSuffix:
			rule.Value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(rule.Value)), ".")
		case TargetIPCIDR:
			prefix, err := netip.ParsePrefix(strings.TrimSpace(rule.Value))
			if err != nil {
				return nil, errors.New("TARGET_CIDR_INVALID")
			}
			rule.Value = prefix.Masked().String()
		default:
			return nil, errors.New("TARGET_KIND_UNSUPPORTED")
		}
		rule.Ports = append([]int(nil), rule.Ports...)
		sort.Ints(rule.Ports)
		rule.Ports = slices.Compact(rule.Ports)
	}
	sort.Slice(result, func(i, j int) bool {
		return targetSortKey(result[i]) < targetSortKey(result[j])
	})
	return result, nil
}

func decisions(targetSetID string, rules []TargetRule, outbound Outbound) []RouteDecision {
	result := make([]RouteDecision, 0, len(rules))
	for _, rule := range rules {
		result = append(result, RouteDecision{TargetSetID: targetSetID, Rule: rule, Outbound: outbound})
	}
	return result
}

func targetSortKey(rule TargetRule) string {
	var builder strings.Builder
	builder.WriteString(string(rule.Kind))
	builder.WriteByte(0)
	builder.WriteString(rule.Value)
	builder.WriteByte(0)
	builder.WriteString(rule.Protocol)
	for _, port := range rule.Ports {
		builder.WriteByte(0)
		builder.WriteString(strconv.Itoa(port))
	}
	return builder.String()
}

func overlaps(left, right []TargetRule) bool {
	for _, a := range left {
		for _, b := range right {
			if protocolsOverlap(a.Protocol, b.Protocol) && portsOverlap(a.Ports, b.Ports) && targetsOverlap(a, b) {
				return true
			}
		}
	}
	return false
}

func protocolsOverlap(left, right string) bool {
	return left == "" || right == "" || left == right
}

func portsOverlap(left, right []int) bool {
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	for _, a := range left {
		for _, b := range right {
			if a == b {
				return true
			}
		}
	}
	return false
}

func targetsOverlap(left, right TargetRule) bool {
	if left.Kind == TargetIPCIDR && right.Kind == TargetIPCIDR {
		leftPrefix, leftErr := netip.ParsePrefix(left.Value)
		rightPrefix, rightErr := netip.ParsePrefix(right.Value)
		return leftErr == nil && rightErr == nil && leftPrefix.Overlaps(rightPrefix)
	}
	if left.Kind == TargetIPCIDR || right.Kind == TargetIPCIDR {
		return false
	}

	switch {
	case left.Kind == TargetDomain && right.Kind == TargetDomain:
		return left.Value == right.Value
	case left.Kind == TargetDomainSuffix && right.Kind == TargetDomainSuffix:
		return domainMatchesSuffix(left.Value, right.Value) || domainMatchesSuffix(right.Value, left.Value)
	case left.Kind == TargetDomain && right.Kind == TargetDomainSuffix:
		return domainMatchesSuffix(left.Value, right.Value)
	case left.Kind == TargetDomainSuffix && right.Kind == TargetDomain:
		return domainMatchesSuffix(right.Value, left.Value)
	default:
		return false
	}
}

func domainMatchesSuffix(domain, suffix string) bool {
	return domain == suffix || strings.HasSuffix(domain, "."+suffix)
}
