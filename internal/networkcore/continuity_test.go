package networkcore

import (
	"strings"
	"testing"
)

func TestContinuityPlanAcceptsLoopbackApplicationEgress(t *testing.T) {
	plan := validContinuityPlan()
	if issues := plan.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
	if plan.Route.Class != RouteChinaEgress || plan.Route.ID != "lcnd-egress-cn" {
		t.Fatalf("route = %#v", plan.Route)
	}
}

func TestContinuityPlanAcceptsApprovedPrivateEndpoint(t *testing.T) {
	plan := validContinuityPlan()
	plan.Endpoint = ServiceEndpoint{Ref: "lcnd-endpoint://private/egress-cn", Network: "tcp", Address: "100.64.0.20", Port: 1080, Scope: EndpointApprovedPrivate, ApprovedNetworkRef: "lcsp-network://tailnet/lcapi"}
	if issues := plan.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestRouteIdentitiesUseStableNames(t *testing.T) {
	tests := []RouteIdentity{
		{ID: "direct", Class: RouteDirect},
		{ID: "lcnd-egress-cn", Class: RouteChinaEgress},
		{ID: "lcnd-egress-global", Class: RouteGlobalEgress},
	}
	for _, route := range tests {
		if issues := route.validate("route"); len(issues) != 0 {
			t.Fatalf("route %#v issues = %#v", route, issues)
		}
	}
	if !hasIssue((RouteIdentity{ID: "unknown", Class: RouteChinaEgress}).validate("route"), "ROUTE_IDENTITY_INVALID") {
		t.Fatal("unstable route identity accepted")
	}
}

func TestContinuityPlanRejectsPublicOrHostWideNetworking(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ContinuityPlan)
		code string
	}{
		{"public address", func(p *ContinuityPlan) { p.Endpoint.Address = "198.51.100.10" }, "ENDPOINT_ADDRESS_FORBIDDEN"},
		{"protected port", func(p *ContinuityPlan) { p.Endpoint.Port = 443 }, "PROTECTED_PORT_COLLISION"},
		{"anonymous", func(p *ContinuityPlan) { p.Authentication.AllowAnonymous = true }, "ANONYMOUS_ACCESS_FORBIDDEN"},
		{"default route", func(p *ContinuityPlan) { p.HostImpact.DefaultRoute = true }, "DEFAULT_ROUTE_CHANGE_FORBIDDEN"},
		{"global dns", func(p *ContinuityPlan) { p.HostImpact.GlobalDNS = true }, "GLOBAL_DNS_CHANGE_FORBIDDEN"},
		{"firewall", func(p *ContinuityPlan) { p.HostImpact.Firewall = true }, "FIREWALL_CHANGE_FORBIDDEN"},
		{"system proxy", func(p *ContinuityPlan) { p.HostImpact.SystemProxy = true }, "SYSTEM_PROXY_CHANGE_FORBIDDEN"},
		{"tailnet", func(p *ContinuityPlan) { p.HostImpact.TailnetPreferences = true }, "TAILNET_CHANGE_FORBIDDEN"},
		{"whole host tun", func(p *ContinuityPlan) { p.HostImpact.WholeHostTUN = true }, "WHOLE_HOST_TUN_FORBIDDEN"},
		{"production", func(p *ContinuityPlan) { p.ProductionAuthorized = true }, "PRODUCTION_AUTHORITY_FORBIDDEN"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := validContinuityPlan()
			test.edit(&plan)
			if !hasIssue(plan.Validate(), test.code) {
				t.Fatalf("Validate() did not contain %s", test.code)
			}
		})
	}
}

func TestContinuityPlanKeepsWorkloadDomainsDistinct(t *testing.T) {
	plan := validContinuityPlan()
	plan.Purpose = PurposeLCSPContinuityB
	if !hasIssue(plan.Validate(), "WORKLOAD_DOMAIN_MISMATCH") {
		t.Fatal("continuity purpose accepted the application egress workload")
	}

	plan = validContinuityPlan()
	plan.SeparateFrom = []string{"old-public-tcp-443"}
	if !hasIssue(plan.Validate(), "WORKLOAD_SEPARATION_REQUIRED") {
		t.Fatal("plan accepted without the continuity workload separation marker")
	}
}

func TestContinuityPlanRejectsCrossDomainEndpointSecretAndProbeReferences(t *testing.T) {
	plan := validContinuityPlan()
	plan.Endpoint.Ref = "lcnd-endpoint://loopback/continuity-b"
	plan.Authentication.Consumers[0].CredentialRef = "lcnd-secret://lcsp/continuity-b/client"
	plan.ProbePlanRef = "lcnd://samples/network/continuity-b-probes.synthetic.json"
	issues := plan.Validate()
	for _, code := range []string{"ENDPOINT_DOMAIN_MISMATCH", "CREDENTIAL_DOMAIN_MISMATCH", "PROBE_DOMAIN_MISMATCH"} {
		if !hasIssue(issues, code) {
			t.Fatalf("Validate() did not contain %s: %#v", code, issues)
		}
	}
}

func TestContinuityAndApplicationSummariesUseDifferentInboundTags(t *testing.T) {
	application := validContinuityPlan()
	applicationSummary, err := BuildCandidateSummary(application, application.Engine.CandidateSHA256)
	if err != nil {
		t.Fatal(err)
	}
	continuity := validContinuityPlan()
	continuity.PlanID = "lcnd-plan-20260812-continuity-b-synthetic"
	continuity.Purpose = PurposeLCSPContinuityB
	continuity.WorkloadID = "lcsp-continuity-b"
	continuity.Route = RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}
	continuity.Endpoint.Ref = "lcnd-endpoint://loopback/continuity-b"
	continuity.Authentication.Consumers = []ConsumerCredentialRef{{ConsumerID: "lcsp-control", IdentityRef: "lcnd-identity://consumers/lcsp-control", CredentialRef: "lcnd-secret://lcsp/continuity-b/client"}}
	continuity.ProbePlanRef = "lcnd://samples/network/continuity-b-probes.synthetic.json"
	continuity.ReceiptDomain = continuity.WorkloadID
	continuity.RollbackDomain = continuity.WorkloadID
	continuity.SeparateFrom = []string{"old-public-tcp-443", "lcnd-egress-cn"}
	continuitySummary, err := BuildCandidateSummary(continuity, continuity.Engine.CandidateSHA256)
	if err != nil {
		t.Fatal(err)
	}
	if applicationSummary.InboundTag == continuitySummary.InboundTag {
		t.Fatalf("inbound tags collide: %q", applicationSummary.InboundTag)
	}
}

func TestContinuityPurposeRequiresNamedEgressRoute(t *testing.T) {
	plan := validContinuityPlan()
	plan.Purpose = PurposeLCSPContinuityB
	plan.WorkloadID = "lcsp-continuity-b"
	plan.Route = RouteIdentity{ID: "direct", Class: RouteDirect}
	plan.Endpoint.Ref = "lcnd-endpoint://loopback/continuity-b"
	plan.Authentication.Consumers = []ConsumerCredentialRef{{ConsumerID: "lcsp-control", IdentityRef: "lcnd-identity://consumers/lcsp-control", CredentialRef: "lcnd-secret://lcsp/continuity-b/client"}}
	plan.ProbePlanRef = "lcnd://samples/network/continuity-b-probes.synthetic.json"
	plan.ReceiptDomain = plan.WorkloadID
	plan.RollbackDomain = plan.WorkloadID
	plan.SeparateFrom = []string{"old-public-tcp-443", "lcnd-egress-cn"}
	if !hasIssue(plan.Validate(), "EGRESS_ROUTE_REQUIRED") {
		t.Fatal("continuity workload accepted the direct route as its egress")
	}
}

func TestContinuityPlanBindsConsumerIdentity(t *testing.T) {
	plan := validContinuityPlan()
	plan.Authentication.Consumers[0].IdentityRef = "lcnd-identity://consumers/other"
	if !hasIssue(plan.Validate(), "CONSUMER_IDENTITY_MISMATCH") {
		t.Fatalf("Validate() issues = %#v", plan.Validate())
	}
}

func TestContinuityPurposeBindsStableRouteDomain(t *testing.T) {
	plan := validContinuityPlan()
	plan.Purpose = PurposeLCSPContinuityB
	plan.WorkloadID = "lcsp-continuity-b"
	plan.Route = RouteIdentity{ID: "lcnd-egress-cn", Class: RouteChinaEgress}
	plan.Endpoint.Ref = "lcnd-endpoint://loopback/continuity-b"
	plan.Authentication.Consumers = []ConsumerCredentialRef{{ConsumerID: "lcsp-control", IdentityRef: "lcnd-identity://consumers/lcsp-control", CredentialRef: "lcnd-secret://lcsp/continuity-b/client"}}
	plan.ProbePlanRef = "lcnd://samples/network/continuity-b-probes.synthetic.json"
	plan.ReceiptDomain = plan.WorkloadID
	plan.RollbackDomain = plan.WorkloadID
	plan.SeparateFrom = []string{"old-public-tcp-443", "lcnd-egress-cn"}
	if !hasIssue(plan.Validate(), "PURPOSE_ROUTE_MISMATCH") {
		t.Fatalf("Validate() issues = %#v", plan.Validate())
	}
}

func TestBuildCandidateSummaryRejectsDigestMismatch(t *testing.T) {
	plan := validContinuityPlan()
	_, err := BuildCandidateSummary(plan, "sha256:"+strings.Repeat("b", 64))
	if err == nil || err.Error() != "CANDIDATE_DIGEST_MISMATCH" {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeContinuityPlanRejectsUnknownField(t *testing.T) {
	_, err := DecodeContinuityPlan(strings.NewReader(`{"schema_version":"1.0.0","unexpected":"DO_NOT_ECHO"}`))
	if err == nil || strings.Contains(err.Error(), "DO_NOT_ECHO") {
		t.Fatalf("error = %v", err)
	}
}

func validContinuityPlan() ContinuityPlan {
	digest := "sha256:" + strings.Repeat("a", 64)
	return ContinuityPlan{
		SchemaVersion:        SchemaVersion,
		PlanID:               "lcnd-plan-20260812-egress-cn-synthetic",
		Purpose:              PurposeLCNDEgressCN,
		TargetPlatform:       "ubuntu-24.04-amd64",
		HostRef:              "lcsp://hosts/owner-approved-primary",
		WorkloadID:           "lcnd-egress-cn",
		Route:                RouteIdentity{ID: "lcnd-egress-cn", Class: RouteChinaEgress},
		Endpoint:             ServiceEndpoint{Ref: "lcnd-endpoint://loopback/egress-cn", Network: "tcp", Address: "127.0.0.1", Port: 1080, Scope: EndpointLoopback},
		Authentication:       ServiceAuthentication{Required: true, AllowAnonymous: false, Consumers: []ConsumerCredentialRef{{ConsumerID: "lcapi", IdentityRef: "lcnd-identity://consumers/lcapi", CredentialRef: "lcnd-secret://lcsp/egress-cn/lcapi"}}},
		Timeouts:             ServiceTimeouts{ConnectMilliseconds: 3000, RequestMilliseconds: 30000, IdleMilliseconds: 60000},
		FailurePolicy:        FailurePolicy{Mode: FailureApprovedDirect, ApprovedRouteID: "direct"},
		Engine:               EngineCandidate{Name: "sing-box", Version: SingBoxVersion, CandidateSHA256: digest},
		ProtectedTargetsRef:  "lcnd://samples/network/protected-platform.synthetic.json",
		ProbePlanRef:         "lcnd://samples/network/egress-cn-probes.synthetic.json",
		ReceiptDomain:        "lcnd-egress-cn",
		RollbackDomain:       "lcnd-egress-cn",
		SeparateFrom:         []string{"old-public-tcp-443", "lcsp-continuity-b"},
		HostImpact:           HostNetworkImpact{},
		ExecutionAuthority:   "lcsp_required",
		ProductionAuthorized: false,
		EvidenceMode:         EvidenceSynthetic,
	}
}

func hasIssue(issues []ValidationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
