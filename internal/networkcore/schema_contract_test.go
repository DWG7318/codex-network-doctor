package networkcore

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var networkSamplePaths = []string{
	"samples/network/node-china-egress.synthetic.json",
	"samples/network/node-global-egress.synthetic.json",
	"samples/network/targets-domestic.synthetic.json",
	"samples/network/targets-global.synthetic.json",
	"samples/network/protected-management.synthetic.json",
	"samples/network/profile-overseas-host.synthetic.json",
	"samples/network/profile-domestic-host.synthetic.json",
	"samples/network/continuity-b.synthetic.json",
	"samples/network/egress-cn.synthetic.json",
	"samples/network/egress-cn-probes.synthetic.json",
	"samples/network/continuity-b-probes.synthetic.json",
	"samples/network/protected-platform.synthetic.json",
	"samples/network/secret-ref-egress-cn.synthetic.json",
	"samples/network/lcapi-egress-attachment.synthetic.json",
	"samples/network/lcapi-global-egress-attachment.synthetic.json",
	"samples/network/rollback-receipt.synthetic.json",
}

func TestPublicNetworkSamplesMatchStrictContracts(t *testing.T) {
	root := networkRepositoryRoot(t)
	tests := []struct {
		path   string
		decode func(io.Reader) error
	}{
		{"samples/network/node-china-egress.synthetic.json", func(reader io.Reader) error { _, err := DecodeNode(reader); return err }},
		{"samples/network/node-global-egress.synthetic.json", func(reader io.Reader) error { _, err := DecodeNode(reader); return err }},
		{"samples/network/targets-domestic.synthetic.json", func(reader io.Reader) error { _, err := DecodeTargetSet(reader); return err }},
		{"samples/network/targets-global.synthetic.json", func(reader io.Reader) error { _, err := DecodeTargetSet(reader); return err }},
		{"samples/network/protected-management.synthetic.json", func(reader io.Reader) error { _, err := DecodeTargetSet(reader); return err }},
		{"samples/network/profile-overseas-host.synthetic.json", func(reader io.Reader) error { _, err := DecodeRouteProfile(reader); return err }},
		{"samples/network/profile-domestic-host.synthetic.json", func(reader io.Reader) error { _, err := DecodeRouteProfile(reader); return err }},
		{"samples/network/continuity-b.synthetic.json", func(reader io.Reader) error { _, err := DecodeContinuityPlan(reader); return err }},
		{"samples/network/egress-cn.synthetic.json", func(reader io.Reader) error { _, err := DecodeContinuityPlan(reader); return err }},
		{"samples/network/egress-cn-probes.synthetic.json", func(reader io.Reader) error { _, err := DecodeProbePlan(reader); return err }},
		{"samples/network/continuity-b-probes.synthetic.json", func(reader io.Reader) error { _, err := DecodeProbePlan(reader); return err }},
		{"samples/network/protected-platform.synthetic.json", func(reader io.Reader) error { _, err := DecodeProtectedTargetSet(reader); return err }},
		{"samples/network/secret-ref-egress-cn.synthetic.json", func(reader io.Reader) error { _, err := DecodeSecretReference(reader); return err }},
		{"samples/network/lcapi-egress-attachment.synthetic.json", func(reader io.Reader) error { _, err := DecodeApplicationAttachment(reader); return err }},
		{"samples/network/lcapi-global-egress-attachment.synthetic.json", func(reader io.Reader) error { _, err := DecodeApplicationAttachment(reader); return err }},
		{"samples/network/rollback-receipt.synthetic.json", func(reader io.Reader) error { _, err := DecodeRollbackReceipt(reader); return err }},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			file, err := os.Open(filepath.Join(root, filepath.FromSlash(test.path)))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if err := test.decode(file); err != nil {
				t.Fatalf("strict decode failed: %v", err)
			}
		})
	}
}

func TestPublicNetworkSamplesContainNoSecretMaterial(t *testing.T) {
	root := networkRepositoryRoot(t)
	for _, path := range networkSamplePaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(data))
		for _, forbidden := range []string{`"password":`, `"private_key":`, "subscription_url", "access_token", "bearer ", "ss://", `"token":`} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contains %q", path, forbidden)
			}
		}
	}
}

func TestPrivateEgressSchemasAreStrict(t *testing.T) {
	root := networkRepositoryRoot(t)
	paths := []string{
		"schemas/network-continuity-plan.schema.json",
		"schemas/network-secret-ref.schema.json",
		"schemas/network-probe-plan.schema.json",
		"schemas/network-protected-targets.schema.json",
		"schemas/network-application-attachment.schema.json",
		"schemas/network-rollback-receipt.schema.json",
	}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, `"additionalProperties": false`) || !strings.Contains(text, `"$schema": "https://json-schema.org/draft/2020-12/schema"`) {
			t.Fatalf("%s is not a strict draft 2020-12 schema", path)
		}
	}
}

func TestPrivateEgressSchemasBindRouteTuplesAndAttachmentDomains(t *testing.T) {
	root := networkRepositoryRoot(t)
	paths := []string{
		"schemas/network-continuity-plan.schema.json",
		"schemas/network-application-attachment.schema.json",
		"schemas/network-candidate-summary.schema.json",
		"schemas/network-probe-plan.schema.json",
		"schemas/network-probe-evidence.schema.json",
	}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		if !schemaHasRouteOneOf(document) {
			t.Fatalf("%s does not bind route id/class with oneOf", path)
		}
	}
	attachment, err := os.ReadFile(filepath.Join(root, "schemas", "network-application-attachment.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(attachment, []byte(`"oneOf"`)) || !bytes.Contains(attachment, []byte(`egress-cn`)) || !bytes.Contains(attachment, []byte(`egress-global`)) {
		t.Fatal("application attachment schema does not bind egress domains")
	}
	continuity, err := os.ReadFile(filepath.Join(root, "schemas", "network-continuity-plan.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{`"probe_plan_ref": {"pattern": "/egress-cn(/|$)"}`, `"probe_plan_ref": {"pattern": "/continuity-b(/|$)"}`} {
		if !bytes.Contains(continuity, []byte(pattern)) {
			t.Fatalf("continuity schema lacks exact probe domain pattern %s", pattern)
		}
	}
}

func schemaHasRouteOneOf(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		if oneOf, ok := typed["oneOf"].([]any); ok && len(oneOf) >= 2 {
			encoded, _ := json.Marshal(typed)
			if bytes.Contains(encoded, []byte(`"id"`)) && bytes.Contains(encoded, []byte(`"class"`)) {
				return true
			}
		}
		for _, child := range typed {
			if schemaHasRouteOneOf(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if schemaHasRouteOneOf(child) {
				return true
			}
		}
	}
	return false
}

func networkRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
