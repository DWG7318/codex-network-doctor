package networkcore

import "testing"

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
		BusinessProbeRef:     "lcnd-probe://lcapi/domestic-synthetic",
		Rollback:             AttachmentRollback{DetachOnly: true, RestoreRouteID: "direct"},
		ProductionAuthorized: false,
	}
}
