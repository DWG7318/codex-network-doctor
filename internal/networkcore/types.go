package networkcore

const SchemaVersion = "1.0.0"
const SingBoxVersion = "1.13.12"

type Direction string

const (
	DirectionChinaEgress  Direction = "china_egress"
	DirectionGlobalEgress Direction = "global_egress"
)

type RouteProfileName string

const (
	ProfileDomesticDirectGlobalTunnel RouteProfileName = "domestic_direct_global_tunnel"
	ProfileGlobalDirectDomesticTunnel RouteProfileName = "global_direct_domestic_tunnel"
)

type PlanState string

const (
	PlanValidated PlanState = "VALIDATED"
	PlanReady     PlanState = "PLAN_READY"
)

type Scenario string

const (
	ScenarioSuccess            Scenario = "success"
	ScenarioVerificationFailed Scenario = "verification_failed"
	ScenarioTimeout            Scenario = "timeout"
)

type Capabilities struct {
	TCP  bool `json:"tcp"`
	UDP  bool `json:"udp"`
	IPv4 bool `json:"ipv4"`
	IPv6 bool `json:"ipv6"`
}

type NodeDescriptor struct {
	SchemaVersion string       `json:"schema_version"`
	NodeID        string       `json:"node_id"`
	DisplayName   string       `json:"display_name"`
	Direction     Direction    `json:"direction"`
	Engine        string       `json:"engine"`
	SecretRef     string       `json:"secret_ref"`
	Capabilities  Capabilities `json:"capabilities"`
	Enabled       bool         `json:"enabled"`
	ValidUntil    string       `json:"valid_until,omitempty"`
}

type TargetKind string

const (
	TargetDomain       TargetKind = "domain"
	TargetDomainSuffix TargetKind = "domain_suffix"
	TargetIPCIDR       TargetKind = "ip_cidr"
)

type TargetRule struct {
	Kind     TargetKind `json:"kind"`
	Value    string     `json:"value"`
	Protocol string     `json:"protocol,omitempty"`
	Ports    []int      `json:"ports,omitempty"`
}

type TargetSet struct {
	SchemaVersion string       `json:"schema_version"`
	TargetSetID   string       `json:"target_set_id"`
	Revision      int          `json:"revision"`
	Purpose       string       `json:"purpose"`
	Targets       []TargetRule `json:"targets"`
}

type RouteProfile struct {
	SchemaVersion        string           `json:"schema_version"`
	ProfileID            string           `json:"profile_id"`
	Name                 RouteProfileName `json:"name"`
	NodeID               string           `json:"node_id"`
	TargetSetID          string           `json:"target_set_id"`
	ProtectedTargetSetID string           `json:"protected_target_set_id"`
}

type ValidationIssue struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationReport struct {
	SchemaVersion string            `json:"schema_version"`
	Kind          string            `json:"kind"`
	Valid         bool              `json:"valid"`
	Issues        []ValidationIssue `json:"issues"`
}
