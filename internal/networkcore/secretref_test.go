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
