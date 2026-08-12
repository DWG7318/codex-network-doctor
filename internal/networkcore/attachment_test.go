package networkcore

import (
	"strings"
	"testing"
)

func TestApplicationAttachmentSupportsProviderModelChannelAndReversibleDirectFallback(t *testing.T) {
	attachment := validApplicationAttachment()
	if issues := attachment.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestApplicationAttachmentRejectsBroadOrUnknownFallback(t *testing.T) {
	attachment := validApplicationAttachment()
	attachment.Selectors = nil
	if !hasIssue(attachment.Validate(), "SELECTOR_REQUIRED") {
		t.Fatal("empty selectors accepted")
	}
	attachment = validApplicationAttachment()
	attachment.FailurePolicy = FailurePolicy{Mode: FailureApprovedDirect, ApprovedRouteID: "unknown"}
	if !hasIssue(attachment.Validate(), "APPROVED_ROUTE_INVALID") {
		t.Fatal("unknown fallback route accepted")
	}
}

func TestApplicationAttachmentRejectsCrossEgressDomainReferences(t *testing.T) {
	attachment := validApplicationAttachment()
	attachment.Route = RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}
	issues := attachment.Validate()
	for _, code := range []string{"ENDPOINT_DOMAIN_MISMATCH", "CREDENTIAL_DOMAIN_MISMATCH", "BUSINESS_PROBE_DOMAIN_MISMATCH"} {
		if !hasIssue(issues, code) {
			t.Fatalf("Validate() did not contain %s: %#v", code, issues)
		}
	}
}

func TestApplicationAttachmentBindsConsumerIdentity(t *testing.T) {
	attachment := validApplicationAttachment()
	attachment.ConsumerIdentityRef = "lcnd-identity://consumers/other"
	if !hasIssue(attachment.Validate(), "CONSUMER_IDENTITY_MISMATCH") {
		t.Fatalf("Validate() issues = %#v", attachment.Validate())
	}
}

func TestApplicationAttachmentRejectsBroadOrUnboundedSelectors(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ApplicationSelector)
		code string
	}{
		{"provider wildcard", func(s *ApplicationSelector) { s.Provider = "*" }, "SELECTOR_PROVIDER_INVALID"},
		{"provider whitespace", func(s *ApplicationSelector) { s.Provider = " synthetic" }, "SELECTOR_PROVIDER_INVALID"},
		{"provider too long", func(s *ApplicationSelector) { s.Provider = strings.Repeat("p", 65) }, "SELECTOR_PROVIDER_INVALID"},
		{"model wildcard", func(s *ApplicationSelector) { s.Model = "*" }, "SELECTOR_MODEL_INVALID"},
		{"model too long", func(s *ApplicationSelector) { s.Model = strings.Repeat("m", 129) }, "SELECTOR_MODEL_INVALID"},
		{"channel broad", func(s *ApplicationSelector) { s.Channel = "all" }, "SELECTOR_CHANNEL_INVALID"},
		{"channel too long", func(s *ApplicationSelector) { s.Channel = strings.Repeat("c", 65) }, "SELECTOR_CHANNEL_INVALID"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attachment := validApplicationAttachment()
			test.edit(&attachment.Selectors[0])
			if !hasIssue(attachment.Validate(), test.code) {
				t.Fatalf("Validate() issues = %#v", attachment.Validate())
			}
		})
	}
}

func validApplicationAttachment() ApplicationAttachment {
	return ApplicationAttachment{
		SchemaVersion:        SchemaVersion,
		AttachmentID:         "lcapi-domestic-synthetic",
		ConsumerID:           "lcapi",
		ConsumerIdentityRef:  "lcnd-identity://consumers/lcapi",
		Selectors:            []ApplicationSelector{{Provider: "synthetic-cn-provider", Model: "synthetic-model", Channel: "chat", TargetSetRef: "lcnd-targets://lcapi/domestic-synthetic"}},
		Route:                RouteIdentity{ID: "lcnd-egress-cn", Class: RouteChinaEgress},
		EndpointRef:          "lcnd-endpoint://loopback/egress-cn",
		CredentialRef:        "lcnd-secret://lcsp/egress-cn/lcapi",
		Timeouts:             ServiceTimeouts{ConnectMilliseconds: 3000, RequestMilliseconds: 30000, IdleMilliseconds: 60000},
		FailurePolicy:        FailurePolicy{Mode: FailureApprovedDirect, ApprovedRouteID: "direct"},
		BusinessProbeRef:     "lcnd-probe://lcapi/egress-cn/domestic-synthetic",
		Rollback:             AttachmentRollback{DetachOnly: true, RestoreRouteID: "direct"},
		ProductionAuthorized: false,
	}
}
