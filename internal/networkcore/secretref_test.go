package networkcore

import (
	"strings"
	"testing"
)

func TestSecretReferenceAllowsReferenceOnly(t *testing.T) {
	ref := SecretReference{SchemaVersion: SchemaVersion, Ref: "lcnd-secret://lcsp/egress-cn/provider-node", Owner: "LCSP", Consumer: "lcnd-egress-cn", MaterialClass: "engine_runtime_config", RotationPolicy: "owner_managed", AllowedOutputs: []string{"presence_status", "redacted_digest"}, ForbiddenOutputs: []string{"value", "url", "password", "token", "subscription", "private_key"}}
	if issues := ref.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestSecretReferenceSafeOutputSetOrderIsIrrelevant(t *testing.T) {
	ref := validSecretReference()
	ref.AllowedOutputs = []string{"redacted_digest", "presence_status"}
	ref.ForbiddenOutputs = []string{"private_key", "subscription", "token", "password", "url", "value"}
	if issues := ref.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestDecodeSecretReferenceRejectsSecretMaterialAndUnknownFields(t *testing.T) {
	for _, input := range []string{
		`{"schema_version":"1.0.0","ref":"ss://example.invalid/material"}`,
		`{"schema_version":"1.0.0","ref":"lcnd-secret://lcsp/egress-cn/provider-node","token":"DO_NOT_ECHO"}`,
	} {
		_, err := DecodeSecretReference(strings.NewReader(input))
		if err == nil || strings.Contains(err.Error(), "DO_NOT_ECHO") {
			t.Fatalf("error = %v", err)
		}
	}
}

func TestSecretReferenceRequiresExactSafeOutputSets(t *testing.T) {
	tests := []struct {
		name string
		edit func(*SecretReference)
		code string
	}{
		{"allowed extra", func(r *SecretReference) { r.AllowedOutputs = append(r.AllowedOutputs, "value") }, "ALLOWED_OUTPUTS_INVALID"},
		{"allowed duplicate", func(r *SecretReference) {
			r.AllowedOutputs = []string{"presence_status", "presence_status", "redacted_digest"}
		}, "ALLOWED_OUTPUTS_INVALID"},
		{"forbidden extra", func(r *SecretReference) { r.ForbiddenOutputs = append(r.ForbiddenOutputs, "raw_config") }, "FORBIDDEN_OUTPUTS_INVALID"},
		{"forbidden duplicate", func(r *SecretReference) { r.ForbiddenOutputs = append(r.ForbiddenOutputs, "value") }, "FORBIDDEN_OUTPUTS_INVALID"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ref := validSecretReference()
			test.edit(&ref)
			if !hasIssue(ref.Validate(), test.code) {
				t.Fatalf("Validate() issues = %#v", ref.Validate())
			}
		})
	}
}

func TestSecretReferenceRejectsCrossWorkloadDomain(t *testing.T) {
	tests := []struct {
		consumer string
		ref      string
	}{
		{"lcnd-egress-cn", "lcnd-secret://lcsp/continuity-b/provider-node"},
		{"lcsp-continuity-b", "lcnd-secret://lcsp/egress-cn/provider-node"},
	}
	for _, test := range tests {
		ref := validSecretReference()
		ref.Consumer = test.consumer
		ref.Ref = test.ref
		if !hasIssue(ref.Validate(), "SECRET_DOMAIN_MISMATCH") {
			t.Fatalf("consumer=%s issues=%#v", test.consumer, ref.Validate())
		}
	}
}

func validSecretReference() SecretReference {
	return SecretReference{SchemaVersion: SchemaVersion, Ref: "lcnd-secret://lcsp/egress-cn/provider-node", Owner: "LCSP", Consumer: "lcnd-egress-cn", MaterialClass: "engine_runtime_config", RotationPolicy: "owner_managed", AllowedOutputs: []string{"presence_status", "redacted_digest"}, ForbiddenOutputs: []string{"value", "url", "password", "token", "subscription", "private_key"}}
}
