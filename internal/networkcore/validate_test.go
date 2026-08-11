package networkcore

import (
	"strings"
	"testing"
)

func TestDecodeNodeAcceptsValidDescriptor(t *testing.T) {
	input := `{
		"schema_version":"1.0.0",
		"node_id":"cn-primary",
		"display_name":"Synthetic China egress",
		"direction":"china_egress",
		"engine":"sing-box",
		"secret_ref":"lcnd-secret://nodes/cn-primary",
		"capabilities":{"tcp":true,"udp":true,"ipv4":true,"ipv6":false},
		"enabled":true
	}`

	node, err := DecodeNode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("DecodeNode() error = %v", err)
	}
	if node != validNode() {
		t.Fatalf("DecodeNode() = %#v, want %#v", node, validNode())
	}
}

func TestDecodeNodeRejectsUnknownField(t *testing.T) {
	_, err := DecodeNode(strings.NewReader(`{
		"schema_version":"1.0.0",
		"node_id":"cn-primary",
		"display_name":"China primary",
		"direction":"china_egress",
		"engine":"sing-box",
		"secret_ref":"lcnd-secret://nodes/cn-primary",
		"capabilities":{"tcp":true,"udp":true,"ipv4":true,"ipv6":false},
		"enabled":true,
		"password":"forbidden"
	}`))
	if err == nil || !strings.Contains(err.Error(), "UNKNOWN_FIELD") {
		t.Fatalf("error = %v, want UNKNOWN_FIELD", err)
	}
}

func TestNodeValidateRejectsEmbeddedEndpointCredential(t *testing.T) {
	node := validNode()
	node.SecretRef = "https://user:password@example.invalid/subscription"
	assertIssueCode(t, node.Validate(), "SECRET_REF_INVALID")
}

func TestNodeValidateRejectsUnsupportedDirection(t *testing.T) {
	node := validNode()
	node.Direction = Direction("sideways")
	assertIssueCode(t, node.Validate(), "DIRECTION_UNSUPPORTED")
}

func TestRouteProfileAndTargetSetRejectsEmptyTargetSet(t *testing.T) {
	targetSet := TargetSet{
		SchemaVersion: SchemaVersion,
		TargetSetID:   "domestic-services",
		Revision:      1,
		Purpose:       "Synthetic domestic service targets",
	}
	assertIssueCode(t, targetSet.Validate(), "TARGETS_REQUIRED")
}

func TestRouteProfileValidateRequiresMatchingDirection(t *testing.T) {
	profile := RouteProfile{
		SchemaVersion:        SchemaVersion,
		ProfileID:            "overseas-host",
		Name:                 ProfileGlobalDirectDomesticTunnel,
		NodeID:               "wrong-direction",
		TargetSetID:          "domestic-services",
		ProtectedTargetSetID: "management",
	}
	node := validNode()
	node.Direction = DirectionGlobalEgress
	assertIssueCode(t, ValidateProfileForNode(profile, node), "NODE_DIRECTION_MISMATCH")
}

func validNode() NodeDescriptor {
	return NodeDescriptor{
		SchemaVersion: SchemaVersion,
		NodeID:        "cn-primary",
		DisplayName:   "Synthetic China egress",
		Direction:     DirectionChinaEgress,
		Engine:        "sing-box",
		SecretRef:     "lcnd-secret://nodes/cn-primary",
		Capabilities:  Capabilities{TCP: true, UDP: true, IPv4: true},
		Enabled:       true,
	}
}

func assertIssueCode(t *testing.T, issues []ValidationIssue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issues = %#v, want code %q", issues, code)
}
