package networkcore

import (
	"io"
	"slices"
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
	for _, required := range []string{"presence_status", "redacted_digest"} {
		if !slices.Contains(ref.AllowedOutputs, required) {
			issues = addIssue(issues, "ALLOWED_OUTPUTS_INCOMPLETE", "allowed_outputs", "safe output declarations are incomplete")
		}
	}
	for _, forbidden := range []string{"value", "url", "password", "token", "subscription", "private_key"} {
		if !slices.Contains(ref.ForbiddenOutputs, forbidden) {
			issues = addIssue(issues, "FORBIDDEN_OUTPUTS_INCOMPLETE", "forbidden_outputs", "secret output prohibitions are incomplete")
		}
	}
	return issues
}
