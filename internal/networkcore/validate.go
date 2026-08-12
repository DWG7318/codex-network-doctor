package networkcore

import (
	"net/netip"
	"regexp"
	"strings"
)

var (
	idPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
	secretRefPattern = regexp.MustCompile(`^lcnd-secret://[a-z0-9][a-z0-9/_-]{1,127}$`)
	sha256RefPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	domainPattern    = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)
)

func (node NodeDescriptor) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, node.SchemaVersion)
	issues = validateID(issues, "node_id", node.NodeID)
	if strings.TrimSpace(node.DisplayName) == "" {
		issues = addIssue(issues, "DISPLAY_NAME_REQUIRED", "display_name", "display name is required")
	}
	if node.Direction != DirectionChinaEgress && node.Direction != DirectionGlobalEgress {
		issues = addIssue(issues, "DIRECTION_UNSUPPORTED", "direction", "direction is unsupported")
	}
	if node.Engine != "sing-box" {
		issues = addIssue(issues, "ENGINE_UNSUPPORTED", "engine", "engine must be sing-box")
	}
	if !secretRefPattern.MatchString(node.SecretRef) {
		issues = addIssue(issues, "SECRET_REF_INVALID", "secret_ref", "secret_ref must be an LCnd secret reference")
	}
	return issues
}

func (targetSet TargetSet) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, targetSet.SchemaVersion)
	issues = validateID(issues, "target_set_id", targetSet.TargetSetID)
	if targetSet.Revision < 1 {
		issues = addIssue(issues, "REVISION_INVALID", "revision", "revision must be at least 1")
	}
	if strings.TrimSpace(targetSet.Purpose) == "" {
		issues = addIssue(issues, "PURPOSE_REQUIRED", "purpose", "purpose is required")
	}
	if len(targetSet.Targets) == 0 {
		issues = addIssue(issues, "TARGETS_REQUIRED", "targets", "at least one target is required")
	}
	if len(targetSet.Targets) > 512 {
		issues = addIssue(issues, "TARGETS_LIMIT_EXCEEDED", "targets", "no more than 512 targets are allowed")
	}
	for index, target := range targetSet.Targets {
		issues = append(issues, target.validate(index)...)
	}
	return issues
}

func (profile RouteProfile) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, profile.SchemaVersion)
	issues = validateID(issues, "profile_id", profile.ProfileID)
	issues = validateID(issues, "node_id", profile.NodeID)
	issues = validateID(issues, "target_set_id", profile.TargetSetID)
	issues = validateID(issues, "protected_target_set_id", profile.ProtectedTargetSetID)
	if profile.Name != ProfileDomesticDirectGlobalTunnel && profile.Name != ProfileGlobalDirectDomesticTunnel {
		issues = addIssue(issues, "PROFILE_NAME_UNSUPPORTED", "name", "route profile name is unsupported")
	}
	return issues
}

func ValidateProfileForNode(profile RouteProfile, node NodeDescriptor) []ValidationIssue {
	issues := profile.Validate()
	issues = append(issues, node.Validate()...)
	if profile.NodeID != node.NodeID {
		issues = addIssue(issues, "NODE_ID_MISMATCH", "node_id", "profile node_id does not match the node descriptor")
	}

	expected := Direction("")
	switch profile.Name {
	case ProfileGlobalDirectDomesticTunnel:
		expected = DirectionChinaEgress
	case ProfileDomesticDirectGlobalTunnel:
		expected = DirectionGlobalEgress
	}
	if expected != "" && node.Direction != expected {
		issues = addIssue(issues, "NODE_DIRECTION_MISMATCH", "direction", "node direction does not match the route profile")
	}
	return issues
}

func (target TargetRule) validate(index int) []ValidationIssue {
	var issues []ValidationIssue
	field := "targets"
	switch target.Kind {
	case TargetDomain, TargetDomainSuffix:
		if !isLowerASCIIDomain(target.Value) {
			issues = addIssue(issues, "TARGET_DOMAIN_INVALID", field, "target domain must be a lower-case ASCII DNS name")
		}
	case TargetIPCIDR:
		prefix, err := netip.ParsePrefix(target.Value)
		if err != nil || prefix != prefix.Masked() {
			issues = addIssue(issues, "TARGET_CIDR_INVALID", field, "target CIDR must be valid and masked")
		}
	default:
		issues = addIssue(issues, "TARGET_KIND_UNSUPPORTED", field, "target kind is unsupported")
	}
	if target.Protocol != "" && target.Protocol != "tcp" && target.Protocol != "udp" {
		issues = addIssue(issues, "TARGET_PROTOCOL_UNSUPPORTED", field, "target protocol is unsupported")
	}
	seenPorts := make(map[int]struct{}, len(target.Ports))
	for _, port := range target.Ports {
		if port < 1 || port > 65535 {
			issues = addIssue(issues, "TARGET_PORT_INVALID", field, "target port must be between 1 and 65535")
		}
		if _, exists := seenPorts[port]; exists {
			issues = addIssue(issues, "TARGET_PORT_DUPLICATE", field, "target ports must be unique")
		}
		seenPorts[port] = struct{}{}
	}
	_ = index
	return issues
}

func validateSchemaVersion(issues []ValidationIssue, version string) []ValidationIssue {
	if version != SchemaVersion {
		return addIssue(issues, "SCHEMA_VERSION_UNSUPPORTED", "schema_version", "schema version is unsupported")
	}
	return issues
}

func validateID(issues []ValidationIssue, field, value string) []ValidationIssue {
	if !idPattern.MatchString(value) {
		return addIssue(issues, "ID_INVALID", field, "identifier format is invalid")
	}
	return issues
}

func isLowerASCIIDomain(value string) bool {
	return len(value) <= 253 && value == strings.ToLower(value) && domainPattern.MatchString(value)
}

func addIssue(issues []ValidationIssue, code, field, message string) []ValidationIssue {
	return append(issues, ValidationIssue{Code: code, Field: field, Message: message})
}
