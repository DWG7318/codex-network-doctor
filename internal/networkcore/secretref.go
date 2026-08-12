package networkcore

import (
	"io"
)

type SecretReference struct {
	SchemaVersion    string   `json:"schema_version"`
	Ref              string   `json:"ref"`
	Owner            string   `json:"owner"`
	Consumer         string   `json:"consumer"`
	MaterialClass    string   `json:"material_class"`
	RotationPolicy   string   `json:"rotation_policy"`
	AllowedOutputs   []string `json:"allowed_outputs"`
	ForbiddenOutputs []string `json:"forbidden_outputs"`
}

func DecodeSecretReference(reader io.Reader) (SecretReference, error) {
	return decodeStrict(reader, "secret_ref", SecretReference.Validate)
}

func (ref SecretReference) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, ref.SchemaVersion)
	if !secretRefPattern.MatchString(ref.Ref) {
		issues = addIssue(issues, "SECRET_REF_INVALID", "ref", "secret reference is invalid")
	}
	if ref.Owner != "LCSP" {
		issues = addIssue(issues, "SECRET_OWNER_INVALID", "owner", "LCSP must own runtime secrets")
	}
	issues = validateID(issues, "consumer", ref.Consumer)
	if ref.MaterialClass != "engine_runtime_config" && ref.MaterialClass != "consumer_credential" {
		issues = addIssue(issues, "MATERIAL_CLASS_UNSUPPORTED", "material_class", "material class is unsupported")
	}
	if ref.RotationPolicy != "owner_managed" {
		issues = addIssue(issues, "ROTATION_POLICY_INVALID", "rotation_policy", "rotation must be owner managed")
	}
	if !exactStringSet(ref.AllowedOutputs, []string{"presence_status", "redacted_digest"}) {
		issues = addIssue(issues, "ALLOWED_OUTPUTS_INVALID", "allowed_outputs", "allowed outputs must be the exact safe set")
	}
	if !exactStringSet(ref.ForbiddenOutputs, []string{"value", "url", "password", "token", "subscription", "private_key"}) {
		issues = addIssue(issues, "FORBIDDEN_OUTPUTS_INVALID", "forbidden_outputs", "forbidden outputs must be the exact closed set")
	}
	domains := map[string]string{
		"lcsp-continuity-b":  "continuity-b",
		"lcnd-egress-cn":     "egress-cn",
		"lcnd-egress-global": "egress-global",
	}
	if domain, ok := domains[ref.Consumer]; ok && !referenceHasSegment(ref.Ref, domain) {
		issues = addIssue(issues, "SECRET_DOMAIN_MISMATCH", "ref", "secret reference belongs to another workload domain")
	}
	return issues
}

func exactStringSet(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	counts := make(map[string]int, len(expected))
	for _, value := range expected {
		counts[value]++
	}
	for _, value := range actual {
		if counts[value] != 1 {
			return false
		}
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
