package networkcore

import (
	"errors"
	"io"
	"net/netip"
	"slices"
	"strings"
)

type ContinuityPurpose string

const (
	PurposeLCSPContinuityB ContinuityPurpose = "lcsp_continuity_b"
	PurposeLCNDEgressCN    ContinuityPurpose = "lcnd_egress_cn"
)

type RouteClass string

const (
	RouteDirect       RouteClass = "direct"
	RouteChinaEgress  RouteClass = "china_egress"
	RouteGlobalEgress RouteClass = "global_egress"
)

type RouteIdentity struct {
	ID    string     `json:"id"`
	Class RouteClass `json:"class"`
}

type EndpointScope string

const (
	EndpointLoopback        EndpointScope = "loopback"
	EndpointApprovedPrivate EndpointScope = "approved_private_network"
)

type ServiceEndpoint struct {
	Ref                string        `json:"ref"`
	Network            string        `json:"network"`
	Address            string        `json:"address"`
	Port               int           `json:"port"`
	Scope              EndpointScope `json:"scope"`
	ApprovedNetworkRef string        `json:"approved_network_ref,omitempty"`
}

type ConsumerCredentialRef struct {
	ConsumerID    string `json:"consumer_id"`
	IdentityRef   string `json:"identity_ref"`
	CredentialRef string `json:"credential_ref"`
}

type ServiceAuthentication struct {
	Required       bool                    `json:"required"`
	AllowAnonymous bool                    `json:"allow_anonymous"`
	Consumers      []ConsumerCredentialRef `json:"consumers"`
}

type ServiceTimeouts struct {
	ConnectMilliseconds int `json:"connect_ms"`
	RequestMilliseconds int `json:"request_ms"`
	IdleMilliseconds    int `json:"idle_ms"`
}

type FailureMode string

const (
	FailureClosed         FailureMode = "fail_closed"
	FailureApprovedDirect FailureMode = "approved_direct"
	FailurePauseBusiness  FailureMode = "pause_business"
)

type FailurePolicy struct {
	Mode            FailureMode `json:"mode"`
	ApprovedRouteID string      `json:"approved_route_id,omitempty"`
}

type EngineCandidate struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	CandidateSHA256 string `json:"candidate_sha256"`
}

type HostNetworkImpact struct {
	DefaultRoute       bool `json:"default_route"`
	GlobalDNS          bool `json:"global_dns"`
	HostsFile          bool `json:"hosts_file"`
	Firewall           bool `json:"firewall"`
	SystemProxy        bool `json:"system_proxy"`
	TailnetPreferences bool `json:"tailnet_preferences"`
	WholeHostTUN       bool `json:"whole_host_tun"`
}

type ContinuityPlan struct {
	SchemaVersion        string                `json:"schema_version"`
	PlanID               string                `json:"plan_id"`
	Purpose              ContinuityPurpose     `json:"purpose"`
	TargetPlatform       string                `json:"target_platform"`
	HostRef              string                `json:"host_ref"`
	WorkloadID           string                `json:"workload_id"`
	Route                RouteIdentity         `json:"route"`
	Endpoint             ServiceEndpoint       `json:"endpoint"`
	Authentication       ServiceAuthentication `json:"authentication"`
	Timeouts             ServiceTimeouts       `json:"timeouts"`
	FailurePolicy        FailurePolicy         `json:"failure_policy"`
	Engine               EngineCandidate       `json:"engine"`
	ProtectedTargetsRef  string                `json:"protected_targets_ref"`
	ProbePlanRef         string                `json:"probe_plan_ref"`
	ReceiptDomain        string                `json:"receipt_domain"`
	RollbackDomain       string                `json:"rollback_domain"`
	SeparateFrom         []string              `json:"separate_from"`
	HostImpact           HostNetworkImpact     `json:"host_impact"`
	ExecutionAuthority   string                `json:"execution_authority"`
	ProductionAuthorized bool                  `json:"production_authorized"`
	EvidenceMode         EvidenceMode          `json:"evidence_mode"`
}

type CandidateSummary struct {
	SchemaVersion   string            `json:"schema_version"`
	PlanID          string            `json:"plan_id"`
	Purpose         ContinuityPurpose `json:"purpose"`
	WorkloadID      string            `json:"workload_id"`
	InboundTag      string            `json:"inbound_tag"`
	Route           RouteIdentity     `json:"route"`
	EndpointRef     string            `json:"endpoint_ref"`
	EndpointScope   EndpointScope     `json:"endpoint_scope"`
	CandidateSHA256 string            `json:"candidate_sha256"`
	EvidenceMode    EvidenceMode      `json:"evidence_mode"`
}

func DecodeContinuityPlan(reader io.Reader) (ContinuityPlan, error) {
	return decodeStrict(reader, "continuity_plan", ContinuityPlan.Validate)
}

func (plan ContinuityPlan) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, plan.SchemaVersion)
	issues = validateID(issues, "plan_id", plan.PlanID)
	issues = validateID(issues, "workload_id", plan.WorkloadID)
	issues = append(issues, plan.Route.validate("route")...)
	issues = append(issues, plan.Endpoint.validate()...)
	issues = append(issues, plan.Authentication.validate()...)
	issues = append(issues, plan.Timeouts.validate()...)
	issues = append(issues, plan.FailurePolicy.validate()...)

	expectedWorkload := ""
	switch plan.Purpose {
	case PurposeLCSPContinuityB:
		expectedWorkload = "lcsp-continuity-b"
		if plan.Route.Class == RouteDirect {
			issues = addIssue(issues, "EGRESS_ROUTE_REQUIRED", "route.class", "continuity workload requires a named egress route")
		}
		if plan.Route != (RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}) {
			issues = addIssue(issues, "PURPOSE_ROUTE_MISMATCH", "route", "continuity purpose requires the stable global egress route")
		}
	case PurposeLCNDEgressCN:
		expectedWorkload = "lcnd-egress-cn"
		if plan.Route.Class != RouteChinaEgress {
			issues = addIssue(issues, "ROUTE_CLASS_MISMATCH", "route.class", "application egress route class is invalid")
		}
	default:
		issues = addIssue(issues, "PURPOSE_UNSUPPORTED", "purpose", "purpose is unsupported")
	}
	if expectedWorkload != "" && plan.WorkloadID != expectedWorkload {
		issues = addIssue(issues, "WORKLOAD_DOMAIN_MISMATCH", "workload_id", "workload does not match purpose")
	}
	domainSegment := "egress-cn"
	if plan.Purpose == PurposeLCSPContinuityB {
		domainSegment = "continuity-b"
	}
	if !strings.Contains(plan.Endpoint.Ref, "/"+domainSegment) {
		issues = addIssue(issues, "ENDPOINT_DOMAIN_MISMATCH", "endpoint.ref", "endpoint belongs to another workload domain")
	}
	for _, consumer := range plan.Authentication.Consumers {
		if !strings.Contains(consumer.CredentialRef, "/"+domainSegment+"/") {
			issues = addIssue(issues, "CREDENTIAL_DOMAIN_MISMATCH", "authentication.consumers.credential_ref", "credential belongs to another workload domain")
		}
	}
	if !strings.Contains(plan.ProbePlanRef, domainSegment) {
		issues = addIssue(issues, "PROBE_DOMAIN_MISMATCH", "probe_plan_ref", "probe plan belongs to another workload domain")
	}
	otherWorkload := "lcsp-continuity-b"
	if plan.WorkloadID == otherWorkload {
		otherWorkload = "lcnd-egress-cn"
	}
	if !slices.Contains(plan.SeparateFrom, "old-public-tcp-443") || !slices.Contains(plan.SeparateFrom, otherWorkload) {
		issues = addIssue(issues, "WORKLOAD_SEPARATION_REQUIRED", "separate_from", "protected workload separation is required")
	}
	if plan.TargetPlatform != "ubuntu-24.04-amd64" {
		issues = addIssue(issues, "TARGET_PLATFORM_UNSUPPORTED", "target_platform", "target platform is unsupported")
	}
	if !validReference(plan.HostRef, "lcsp://hosts/") {
		issues = addIssue(issues, "HOST_REF_INVALID", "host_ref", "host reference is invalid")
	}
	if plan.Engine.Name != "sing-box" || plan.Engine.Version != SingBoxVersion {
		issues = addIssue(issues, "ENGINE_UNSUPPORTED", "engine", "engine is unsupported")
	}
	if !sha256RefPattern.MatchString(plan.Engine.CandidateSHA256) {
		issues = addIssue(issues, "CANDIDATE_DIGEST_INVALID", "engine.candidate_sha256", "candidate digest is invalid")
	}
	if !validReference(plan.ProtectedTargetsRef, "lcnd://") {
		issues = addIssue(issues, "PROTECTED_TARGETS_REF_INVALID", "protected_targets_ref", "protected target reference is invalid")
	}
	if !validReference(plan.ProbePlanRef, "lcnd://") {
		issues = addIssue(issues, "PROBE_PLAN_REF_INVALID", "probe_plan_ref", "probe plan reference is invalid")
	}
	if plan.ReceiptDomain != plan.WorkloadID || plan.RollbackDomain != plan.WorkloadID {
		issues = addIssue(issues, "RECEIPT_ROLLBACK_DOMAIN_MISMATCH", "rollback_domain", "receipt and rollback must stay in one workload domain")
	}
	issues = append(issues, plan.HostImpact.validate()...)
	if plan.ExecutionAuthority != "lcsp_required" {
		issues = addIssue(issues, "EXECUTION_AUTHORITY_INVALID", "execution_authority", "LCSP execution authority is required")
	}
	if plan.ProductionAuthorized {
		issues = addIssue(issues, "PRODUCTION_AUTHORITY_FORBIDDEN", "production_authorized", "repository plans cannot authorize production")
	}
	if plan.EvidenceMode != EvidenceSynthetic {
		issues = addIssue(issues, "EVIDENCE_MODE_INVALID", "evidence_mode", "repository plans must be synthetic")
	}
	return issues
}

func BuildCandidateSummary(plan ContinuityPlan, observedDigest string) (CandidateSummary, error) {
	if issues := plan.Validate(); len(issues) != 0 {
		return CandidateSummary{}, ValidationError{Kind: "continuity_plan", Issues: issues}
	}
	if observedDigest != plan.Engine.CandidateSHA256 {
		return CandidateSummary{}, errors.New("CANDIDATE_DIGEST_MISMATCH")
	}
	return CandidateSummary{SchemaVersion: SchemaVersion, PlanID: plan.PlanID, Purpose: plan.Purpose, WorkloadID: plan.WorkloadID, InboundTag: plan.WorkloadID + "-in", Route: plan.Route, EndpointRef: plan.Endpoint.Ref, EndpointScope: plan.Endpoint.Scope, CandidateSHA256: plan.Engine.CandidateSHA256, EvidenceMode: EvidenceSynthetic}, nil
}

func (route RouteIdentity) validate(field string) []ValidationIssue {
	var issues []ValidationIssue
	issues = validateID(issues, field+".id", route.ID)
	if route.Class != RouteDirect && route.Class != RouteChinaEgress && route.Class != RouteGlobalEgress {
		issues = addIssue(issues, "ROUTE_CLASS_UNSUPPORTED", field+".class", "route class is unsupported")
	}
	if route.Class == RouteDirect && route.ID != "direct" {
		issues = addIssue(issues, "DIRECT_ROUTE_ID_INVALID", field+".id", "direct route id must be direct")
	}
	expectedIDs := map[RouteClass]string{RouteDirect: "direct", RouteChinaEgress: "lcnd-egress-cn", RouteGlobalEgress: "lcnd-egress-global"}
	if expected, ok := expectedIDs[route.Class]; ok && route.ID != expected {
		issues = addIssue(issues, "ROUTE_IDENTITY_INVALID", field+".id", "route identity is not the stable name for its class")
	}
	return issues
}

func (endpoint ServiceEndpoint) validate() []ValidationIssue {
	var issues []ValidationIssue
	if !validReference(endpoint.Ref, "lcnd-endpoint://") {
		issues = addIssue(issues, "ENDPOINT_REF_INVALID", "endpoint.ref", "endpoint reference is invalid")
	}
	if endpoint.Network != "tcp" {
		issues = addIssue(issues, "ENDPOINT_NETWORK_UNSUPPORTED", "endpoint.network", "only application TCP endpoints are supported")
	}
	address, err := netip.ParseAddr(endpoint.Address)
	if err != nil {
		issues = addIssue(issues, "ENDPOINT_ADDRESS_INVALID", "endpoint.address", "endpoint address is invalid")
	} else {
		switch endpoint.Scope {
		case EndpointLoopback:
			if !address.IsLoopback() || endpoint.ApprovedNetworkRef != "" {
				issues = addIssue(issues, "ENDPOINT_ADDRESS_FORBIDDEN", "endpoint.address", "loopback scope requires a loopback address")
			}
		case EndpointApprovedPrivate:
			if (!address.IsPrivate() && !carrierGradeNAT.Contains(address)) || !validReference(endpoint.ApprovedNetworkRef, "lcsp-network://") {
				issues = addIssue(issues, "ENDPOINT_ADDRESS_FORBIDDEN", "endpoint.address", "private endpoints require an approved private network")
			}
		default:
			issues = addIssue(issues, "ENDPOINT_SCOPE_UNSUPPORTED", "endpoint.scope", "endpoint scope is unsupported")
		}
	}
	if endpoint.Port < 1 || endpoint.Port > 65535 {
		issues = addIssue(issues, "ENDPOINT_PORT_INVALID", "endpoint.port", "endpoint port is invalid")
	}
	if endpoint.Port == 22 || endpoint.Port == 80 || endpoint.Port == 443 {
		issues = addIssue(issues, "PROTECTED_PORT_COLLISION", "endpoint.port", "endpoint collides with a protected port")
	}
	return issues
}

func (auth ServiceAuthentication) validate() []ValidationIssue {
	var issues []ValidationIssue
	if !auth.Required {
		issues = addIssue(issues, "AUTHENTICATION_REQUIRED", "authentication.required", "authentication is required")
	}
	if auth.AllowAnonymous {
		issues = addIssue(issues, "ANONYMOUS_ACCESS_FORBIDDEN", "authentication.allow_anonymous", "anonymous access is forbidden")
	}
	if len(auth.Consumers) == 0 || len(auth.Consumers) > 64 {
		issues = addIssue(issues, "CONSUMER_ALLOWLIST_INVALID", "authentication.consumers", "a bounded consumer allowlist is required")
	}
	seen := map[string]struct{}{}
	for _, consumer := range auth.Consumers {
		issues = validateID(issues, "authentication.consumers.consumer_id", consumer.ConsumerID)
		if !validReference(consumer.IdentityRef, "lcnd-identity://consumers/") {
			issues = addIssue(issues, "CONSUMER_IDENTITY_REF_INVALID", "authentication.consumers.identity_ref", "consumer identity reference is invalid")
		} else if consumer.IdentityRef != "lcnd-identity://consumers/"+consumer.ConsumerID {
			issues = addIssue(issues, "CONSUMER_IDENTITY_MISMATCH", "authentication.consumers.identity_ref", "consumer identity does not match consumer_id")
		}
		if !secretRefPattern.MatchString(consumer.CredentialRef) {
			issues = addIssue(issues, "CREDENTIAL_REF_INVALID", "authentication.consumers.credential_ref", "credential reference is invalid")
		}
		if _, ok := seen[consumer.ConsumerID]; ok {
			issues = addIssue(issues, "CONSUMER_DUPLICATE", "authentication.consumers", "consumer identities must be unique")
		}
		seen[consumer.ConsumerID] = struct{}{}
	}
	return issues
}

func (timeouts ServiceTimeouts) validate() []ValidationIssue {
	var issues []ValidationIssue
	if timeouts.ConnectMilliseconds < 100 || timeouts.ConnectMilliseconds > 60000 {
		issues = addIssue(issues, "CONNECT_TIMEOUT_INVALID", "timeouts.connect_ms", "connect timeout is outside the bounded range")
	}
	if timeouts.RequestMilliseconds < timeouts.ConnectMilliseconds || timeouts.RequestMilliseconds > 120000 {
		issues = addIssue(issues, "REQUEST_TIMEOUT_INVALID", "timeouts.request_ms", "request timeout is outside the bounded range")
	}
	if timeouts.IdleMilliseconds < timeouts.RequestMilliseconds || timeouts.IdleMilliseconds > 300000 {
		issues = addIssue(issues, "IDLE_TIMEOUT_INVALID", "timeouts.idle_ms", "idle timeout is outside the bounded range")
	}
	return issues
}

func (policy FailurePolicy) validate() []ValidationIssue {
	var issues []ValidationIssue
	switch policy.Mode {
	case FailureApprovedDirect:
		if policy.ApprovedRouteID != "direct" {
			issues = addIssue(issues, "APPROVED_ROUTE_INVALID", "failure_policy.approved_route_id", "only the named direct route can be approved")
		}
	case FailureClosed, FailurePauseBusiness:
		if policy.ApprovedRouteID != "" {
			issues = addIssue(issues, "FAILURE_ROUTE_FORBIDDEN", "failure_policy.approved_route_id", "this failure policy cannot fall back")
		}
	default:
		issues = addIssue(issues, "FAILURE_POLICY_UNSUPPORTED", "failure_policy.mode", "failure policy is unsupported")
	}
	return issues
}

func (impact HostNetworkImpact) validate() []ValidationIssue {
	var issues []ValidationIssue
	checks := []struct {
		changed     bool
		code, field string
	}{
		{impact.DefaultRoute, "DEFAULT_ROUTE_CHANGE_FORBIDDEN", "host_impact.default_route"},
		{impact.GlobalDNS, "GLOBAL_DNS_CHANGE_FORBIDDEN", "host_impact.global_dns"},
		{impact.HostsFile, "HOSTS_CHANGE_FORBIDDEN", "host_impact.hosts_file"},
		{impact.Firewall, "FIREWALL_CHANGE_FORBIDDEN", "host_impact.firewall"},
		{impact.SystemProxy, "SYSTEM_PROXY_CHANGE_FORBIDDEN", "host_impact.system_proxy"},
		{impact.TailnetPreferences, "TAILNET_CHANGE_FORBIDDEN", "host_impact.tailnet_preferences"},
		{impact.WholeHostTUN, "WHOLE_HOST_TUN_FORBIDDEN", "host_impact.whole_host_tun"},
	}
	for _, check := range checks {
		if check.changed {
			issues = addIssue(issues, check.code, check.field, "host-wide network changes are forbidden")
		}
	}
	return issues
}

func validReference(value, prefix string) bool {
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix) && len(value) <= 256 && !strings.ContainsAny(value, " \t\r\n?#")
}

func egressDomainForRoute(route RouteIdentity) (string, bool) {
	switch route {
	case RouteIdentity{ID: "lcnd-egress-cn", Class: RouteChinaEgress}:
		return "egress-cn", true
	case RouteIdentity{ID: "lcnd-egress-global", Class: RouteGlobalEgress}:
		return "egress-global", true
	default:
		return "", false
	}
}

func referenceHasSegment(reference, segment string) bool {
	if segment == "" {
		return false
	}
	for _, part := range strings.Split(reference, "/") {
		if part == segment {
			return true
		}
	}
	return false
}

var carrierGradeNAT = netip.MustParsePrefix("100.64.0.0/10")
