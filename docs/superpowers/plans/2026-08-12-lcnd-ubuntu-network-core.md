# LCnd Ubuntu Network Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a simulation-only `cnd-network` CLI that validates synthetic node/target contracts, plans both cross-region directions, renders a protected sing-box candidate, and produces deterministic simulated execution evidence without changing any host network state.

**Architecture:** Keep the existing passive `cnd.exe` and Windows repair binaries unchanged. Add an isolated `internal/networkcore` domain, a sing-box 1.13.12 adapter under `internal/networkengine/singbox`, and a thin `internal/networkcli`/`cmd/cnd-network` entry point. LCSP remains a deferred infrastructure adapter; this plan does not modify LCSP or call sudo/systemd/network commands.

**Tech Stack:** Go 1.26 standard library, JSON Schema draft 2020-12 documents, deterministic JSON/SHA-256, PowerShell 5.1-compatible build scripts, sing-box stable target 1.13.12 (configuration generation only; no download or installation).

**Approved design:** `docs/superpowers/specs/2026-08-12-lcnd-ubuntu-network-core-design.md`

**Official sing-box references:**

- `https://github.com/SagerNet/sing-box/releases/tag/v1.13.12`
- `https://sing-box.sagernet.org/configuration/route/rule/`
- `https://sing-box.sagernet.org/configuration/inbound/tun/`
- `https://sing-box.sagernet.org/deprecated/`

---

## Scope and file map

### Files to create

```text
cmd/cnd-network/main.go                         process entry point only
cmd/cnd-network/main_test.go                    entry/version behavior
internal/networkcore/types.go                   public domain types and enums
internal/networkcore/decode.go                  strict single-document JSON decoding
internal/networkcore/validate.go                fail-closed domain validation
internal/networkcore/canonical.go               deterministic JSON and SHA-256
internal/networkcore/planner.go                 normalized bidirectional route planning
internal/networkcore/simulate.go                simulation state machine
internal/networkcore/report.go                  synthetic health report construction
internal/networkcore/*_test.go                  focused TDD coverage
internal/networkengine/singbox/types.go          protected node material and config types
internal/networkengine/singbox/render.go         1.13.12 candidate rendering
internal/networkengine/singbox/render_test.go    render, digest, mode, and redaction tests
internal/networkcli/cli.go                       validate/plan/render/simulate/report commands
internal/networkcli/cli_test.go                  command, exit code, and leak tests
schemas/network-node.schema.json                 public node descriptor contract
schemas/network-target-set.schema.json           public target-set contract
schemas/network-route-profile.schema.json        public route-profile contract
schemas/network-change-plan.schema.json          finalized plan contract
schemas/network-execution-receipt.schema.json    simulation/future execution receipt
schemas/network-health-report.schema.json        synthetic/future live report contract
samples/network/node-china-egress.synthetic.json public non-secret sample
samples/network/node-global-egress.synthetic.json public non-secret sample
samples/network/targets-domestic.synthetic.json  public domestic target sample
samples/network/targets-global.synthetic.json    public global target sample
samples/network/protected-management.synthetic.json public management-path sample
samples/network/profile-overseas-host.synthetic.json overseas-host profile
samples/network/profile-domestic-host.synthetic.json domestic-host profile
scripts/build-linux.ps1                          Ubuntu amd64 cross-build without CGO
docs/RFC/0003-ubuntu-network-core.md              durable product/CLI contract
```

### Files to modify

```text
internal/projectassets/assets_test.go            require and parse new public assets
scripts/build.ps1                                build cnd-network.exe without changing existing outputs
README.md                                        document simulation-only Ubuntu CLI
docs/MASTER_SPEC.md                              add server network-core boundary
docs/SDD.md                                      add isolated network-core architecture
docs/TEST_HANDBOOK.md                            add focused and cross-build checks
docs/BACKLOG.md                                  record deferred LCSP/live-node slices
CHANGELOG.md                                     prepare v0.3.0 entry after acceptance
VERSION                                          0.2.0 -> 0.3.0 after all gates pass
```

### Explicitly untouched

```text
internal/tunrepair/**
internal/proxyrepair/**
internal/speedtest/**
cmd/cnd/**
cmd/cnd-repair/**
any LCSP repository or production server
```

## Stable contracts used by every task

Use these exact names throughout the implementation:

```go
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
```

The CLI command surface is also fixed for this slice:

```text
cnd-network version
cnd-network validate --kind node|target-set|route-profile --input FILE [--format text|json]
cnd-network plan --node FILE --targets FILE --protected FILE --profile FILE --output DRAFT.json [--format text|json]
cnd-network render --draft DRAFT.json --material SECRET.json --candidate CANDIDATE.json --plan PLAN.json [--format text|json]
cnd-network simulate --plan PLAN.json --scenario success|verification_failed|timeout --output RECEIPT.json [--format text|json]
cnd-network report --plan PLAN.json --receipt RECEIPT.json --output REPORT.json [--format text|json]
```

Exit codes:

```text
0 success
1 operational/read/write/render failure
2 invalid arguments, contract rejection, or invalid state transition
```

---

### Task 1: Strict network domain contracts

**Files:**

- Create: `internal/networkcore/types.go`
- Create: `internal/networkcore/decode.go`
- Create: `internal/networkcore/validate.go`
- Test: `internal/networkcore/validate_test.go`

- [ ] **Step 1: Write failing strict-decoding and validation tests**

Create table-driven tests that cover a valid node, an unknown JSON field, an embedded credential, an unsupported direction, an empty target set, and a profile/node direction mismatch.

```go
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
	issues := node.Validate()
	assertIssueCode(t, issues, "SECRET_REF_INVALID")
}

func TestRouteProfileValidateRequiresMatchingDirection(t *testing.T) {
	profile := RouteProfile{
		SchemaVersion: SchemaVersion,
		ProfileID: "overseas-host",
		Name: ProfileGlobalDirectDomesticTunnel,
		NodeID: "wrong-direction",
		TargetSetID: "domestic-services",
		ProtectedTargetSetID: "management",
	}
	node := validNode()
	node.Direction = DirectionGlobalEgress
	issues := ValidateProfileForNode(profile, node)
	assertIssueCode(t, issues, "NODE_DIRECTION_MISMATCH")
}

func validNode() NodeDescriptor {
	return NodeDescriptor{
		SchemaVersion: SchemaVersion,
		NodeID: "cn-primary",
		DisplayName: "Synthetic China egress",
		Direction: DirectionChinaEgress,
		Engine: "sing-box",
		SecretRef: "lcnd-secret://nodes/cn-primary",
		Capabilities: Capabilities{TCP: true, UDP: true, IPv4: true},
		Enabled: true,
	}
}

func assertIssueCode(t *testing.T, issues []ValidationIssue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code { return }
	}
	t.Fatalf("issues = %#v, want code %q", issues, code)
}
```

- [ ] **Step 2: Run the focused test and confirm RED**

Run:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkcore -run 'Test(DecodeNode|NodeValidate|RouteProfile)' -count=1
```

Expected: FAIL because `DecodeNode`, `NodeDescriptor`, and validation methods do not exist.

- [ ] **Step 3: Add the exact domain types**

Implement these structures in `types.go` with the shown JSON names. Do not use `map[string]any` in the domain layer.

```go
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
```

- [ ] **Step 4: Implement strict decoding and fail-closed validation**

`decode.go` must call `DisallowUnknownFields`, reject a second JSON value, and return safe error codes without echoing input values.

```go
func decodeStrict[T any](reader io.Reader, kind string, validate func(T) []ValidationIssue) (T, error) {
	var value T
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("%s_DECODE_FAILED: %w", strings.ToUpper(kind), sanitizeJSONError(err))
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return value, fmt.Errorf("%s_TRAILING_DATA", strings.ToUpper(kind))
	}
	if issues := validate(value); len(issues) != 0 {
		return value, ValidationError{Kind: kind, Issues: issues}
	}
	return value, nil
}

func DecodeNode(reader io.Reader) (NodeDescriptor, error) {
	return decodeStrict(reader, "node", NodeDescriptor.Validate)
}

func DecodeTargetSet(reader io.Reader) (TargetSet, error) {
	return decodeStrict(reader, "target_set", TargetSet.Validate)
}

func DecodeRouteProfile(reader io.Reader) (RouteProfile, error) {
	return decodeStrict(reader, "route_profile", RouteProfile.Validate)
}

type ValidationError struct {
	Kind   string
	Issues []ValidationIssue
}

func (err ValidationError) Error() string {
	codes := make([]string, 0, len(err.Issues))
	for _, issue := range err.Issues { codes = append(codes, issue.Code) }
	return strings.ToUpper(err.Kind) + "_INVALID: " + strings.Join(codes, ",")
}

func sanitizeJSONError(err error) error {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) { return fmt.Errorf("invalid JSON near byte %d", syntax.Offset) }
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) { return fmt.Errorf("invalid type for field %s", typeError.Field) }
	if strings.HasPrefix(err.Error(), "json: unknown field ") { return errors.New("UNKNOWN_FIELD") }
	return errors.New("INVALID_JSON")
}
```

Validation requirements:

```text
IDs: ^[a-z0-9][a-z0-9._-]{1,63}$
secret_ref: ^lcnd-secret://[a-z0-9][a-z0-9/_-]{1,127}$
engine: exactly sing-box
direction: china_egress or global_egress
target protocol: empty, tcp, or udp
target ports: 1..65535, sorted and unique after normalization
domain values: lower-case ASCII DNS names without URL schemes or credentials
CIDR values: net/netip.ParsePrefix succeeds and is masked
profile/name direction: global_direct_domestic_tunnel -> china_egress;
                        domestic_direct_global_tunnel -> global_egress
```

- [ ] **Step 5: Run focused tests and confirm GREEN**

Run:

```powershell
& $go test ./internal/networkcore -run 'Test(DecodeNode|NodeValidate|RouteProfile)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 1**

```powershell
git add -- internal/networkcore/types.go internal/networkcore/decode.go internal/networkcore/validate.go internal/networkcore/validate_test.go
git commit -m "feat(network): add strict network contracts"
```

---

### Task 2: Public schemas and synthetic samples

**Files:**

- Create: `schemas/network-node.schema.json`
- Create: `schemas/network-target-set.schema.json`
- Create: `schemas/network-route-profile.schema.json`
- Create: `schemas/network-change-plan.schema.json`
- Create: `schemas/network-execution-receipt.schema.json`
- Create: `schemas/network-health-report.schema.json`
- Create: `samples/network/*.synthetic.json`
- Modify: `internal/projectassets/assets_test.go`
- Test: `internal/networkcore/schema_contract_test.go`

- [ ] **Step 1: Extend asset tests first**

Add the six schema paths and seven sample paths to `TestRequiredEngineeringAssetsExist`. Extend `TestJSONSchemasAreValidJSON` with all six new schemas and assert each has `additionalProperties == false`.

```go
for _, relativePath := range []string{
	"schemas/network-node.schema.json",
	"schemas/network-target-set.schema.json",
	"schemas/network-route-profile.schema.json",
	"schemas/network-change-plan.schema.json",
	"schemas/network-execution-receipt.schema.json",
	"schemas/network-health-report.schema.json",
} {
	t.Run(relativePath, func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
		if err != nil { t.Fatal(err) }
		var schema map[string]any
		if err := json.Unmarshal(data, &schema); err != nil { t.Fatal(err) }
		if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" { t.Fatalf("$schema = %v", schema["$schema"]) }
		if schema["additionalProperties"] != false { t.Fatalf("additionalProperties = %v", schema["additionalProperties"]) }
	})
}
```

- [ ] **Step 2: Run asset tests and confirm RED**

Run:

```powershell
& $go test ./internal/projectassets -run 'Test(RequiredEngineeringAssetsExist|JSONSchemasAreValidJSON)' -count=1
```

Expected: FAIL with missing `schemas/network-*.schema.json`.

- [ ] **Step 3: Create strict JSON Schema contracts**

Every schema uses draft 2020-12, `type: object`, `additionalProperties: false`, explicit `required`, bounded string/array sizes, and the same enums/patterns as Go validation. Use this exact top-level pattern and replace `title`, `$id`, `required`, and `properties` for each contract.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.invalid/cnd/schemas/network-node.schema.json",
  "title": "LCnd Network Node Descriptor",
  "type": "object",
  "additionalProperties": false,
  "required": ["schema_version", "node_id", "display_name", "direction", "engine", "secret_ref", "capabilities", "enabled"],
  "properties": {
    "schema_version": {"const": "1.0.0"},
    "node_id": {"type": "string", "pattern": "^[a-z0-9][a-z0-9._-]{1,63}$"},
    "display_name": {"type": "string", "minLength": 1, "maxLength": 128},
    "direction": {"enum": ["china_egress", "global_egress"]},
    "engine": {"const": "sing-box"},
    "secret_ref": {"type": "string", "pattern": "^lcnd-secret://[a-z0-9][a-z0-9/_-]{1,127}$"},
    "capabilities": {
      "type": "object",
      "additionalProperties": false,
      "required": ["tcp", "udp", "ipv4", "ipv6"],
      "properties": {
        "tcp": {"type": "boolean"},
        "udp": {"type": "boolean"},
        "ipv4": {"type": "boolean"},
        "ipv6": {"type": "boolean"}
      }
    },
    "enabled": {"type": "boolean"},
    "valid_until": {"type": "string", "format": "date-time"}
  }
}
```

Use these exact top-level requirements for the remaining schemas; nested objects also set `additionalProperties: false`:

| Schema | Required fields | Exact constraints |
| --- | --- | --- |
| `network-target-set` | `schema_version,target_set_id,revision,purpose,targets` | revision integer >= 1; 1..512 targets; each target requires `kind,value`; kind enum `domain,domain_suffix,ip_cidr`; protocol enum `tcp,udp`; ports unique integers 1..65535 |
| `network-route-profile` | `schema_version,profile_id,name,node_id,target_set_id,protected_target_set_id` | name enum contains exactly the two approved profiles; all IDs use the shared ID pattern |
| `network-change-plan` | `schema_version,plan_id,state,tool_version,platform,architecture,profile_id,node_id,node_direction,default_outbound,tunnel_outbound,decisions,protected_decisions,candidate,candidate_sha256,simulation_only` | state const `PLAN_READY`; platform const `ubuntu-24.04`; architecture const `amd64`; outbound enums `direct,lcnd-tunnel`; candidate requires `engine,engine_version,secret_ref`; SHA fields use `^[0-9a-f]{64}$`; simulation const true |
| `network-execution-receipt` | `schema_version,plan_id,evidence_mode,scenario,state,candidate_sha256,rollback_armed,rollback_triggered,actions` | evidence const `synthetic`; scenario enum contains the three approved scenarios; state enum `VERIFIED,ROLLED_BACK`; actions require `action,status,code` |
| `network-health-report` | `schema_version,plan_id,evidence_mode,status,summary,findings` | evidence const `synthetic`; status enum `SIMULATED_PASS,SIMULATED_ROLLBACK`; findings contain only `code,scope,explanation,action` |

All schemas use `schema_version: {"const":"1.0.0"}`. No public schema permits `password`, `token`, `credential`, raw outbound configuration, or arbitrary command fields.

- [ ] **Step 4: Create only synthetic public samples**

Use documentation ranges and domains. The China node sample is exactly:

```json
{
  "schema_version": "1.0.0",
  "node_id": "china-primary",
  "display_name": "Synthetic China egress",
  "direction": "china_egress",
  "engine": "sing-box",
  "secret_ref": "lcnd-secret://nodes/china-primary",
  "capabilities": {"tcp": true, "udp": true, "ipv4": true, "ipv6": false},
  "enabled": true
}
```

Use `api.example.cn`, `models.example.cn`, `api.example.com`, and RFC 5737 address ranges in target samples. The `.synthetic.json` filenames and `purpose` text identify them as synthetic; do not add fields outside the strict contracts.

Create the remaining samples with these exact references:

| File | ID/profile | Direction or targets |
| --- | --- | --- |
| `node-global-egress.synthetic.json` | `global-primary` | `global_egress`, secret ref `lcnd-secret://nodes/global-primary` |
| `targets-domestic.synthetic.json` | `domestic-services`, revision 1 | domains `api.example.cn`, `models.example.cn` |
| `targets-global.synthetic.json` | `global-services`, revision 1 | domain `api.example.com` and CIDR `203.0.113.0/24` |
| `protected-management.synthetic.json` | `management`, revision 1 | CIDRs `127.0.0.0/8`, `10.0.0.0/8`, `192.168.0.0/16` |
| `profile-overseas-host.synthetic.json` | `overseas-host` | `global_direct_domestic_tunnel`, node `china-primary`, targets `domestic-services`, protected `management` |
| `profile-domestic-host.synthetic.json` | `domestic-host` | `domestic_direct_global_tunnel`, node `global-primary`, targets `global-services`, protected `management` |

- [ ] **Step 5: Add contract/fixture parity tests**

`schema_contract_test.go` must decode every sample through the matching strict Go decoder and scan public fixtures for forbidden keys.

```go
func TestPublicNetworkSamplesContainNoSecretMaterial(t *testing.T) {
	root := repositoryRoot(t)
	for _, path := range networkSamplePaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil { t.Fatal(err) }
		lower := strings.ToLower(string(data))
		for _, forbidden := range []string{"password", "private_key", "subscription_url", "access_token", "bearer "} {
			if strings.Contains(lower, forbidden) { t.Fatalf("%s contains %q", path, forbidden) }
		}
	}
}
```

- [ ] **Step 6: Run and commit Task 2**

Run:

```powershell
& $go test ./internal/projectassets ./internal/networkcore -count=1
```

Expected: PASS.

Commit:

```powershell
git add -- schemas/network-*.schema.json samples/network internal/projectassets/assets_test.go internal/networkcore/schema_contract_test.go
git commit -m "feat(network): publish network contract schemas"
```

---

### Task 3: Deterministic bidirectional planner

**Files:**

- Create: `internal/networkcore/canonical.go`
- Create: `internal/networkcore/planner.go`
- Test: `internal/networkcore/planner_test.go`

- [ ] **Step 1: Write failing direction, protection, overlap, and determinism tests**

```go
func TestBuildDraftRoutesDomesticTargetsThroughChinaEgress(t *testing.T) {
	draft, err := BuildDraft(chinaNode(), domesticTargets(), managementTargets(), overseasHostProfile(), "0.3.0")
	if err != nil { t.Fatal(err) }
	if draft.DefaultOutbound != OutboundDirect { t.Fatalf("default = %q", draft.DefaultOutbound) }
	if draft.TunnelOutbound != OutboundTunnel { t.Fatalf("tunnel = %q", draft.TunnelOutbound) }
	if draft.NodeDirection != DirectionChinaEgress { t.Fatalf("direction = %q", draft.NodeDirection) }
	assertAllDecisions(t, draft.Decisions, "domestic-services", OutboundTunnel)
	assertAllDecisions(t, draft.ProtectedDecisions, "management", OutboundDirect)
}

func TestBuildDraftIsDeterministicAcrossInputOrder(t *testing.T) {
	firstTargets := domesticTargets()
	secondTargets := domesticTargets()
	slices.Reverse(secondTargets.Targets)
	first, err := BuildDraft(chinaNode(), firstTargets, managementTargets(), overseasHostProfile(), "0.3.0")
	if err != nil { t.Fatal(err) }
	second, err := BuildDraft(chinaNode(), secondTargets, managementTargets(), overseasHostProfile(), "0.3.0")
	if err != nil { t.Fatal(err) }
	firstJSON, firstSHA, _ := CanonicalJSON(first)
	secondJSON, secondSHA, _ := CanonicalJSON(second)
	if !bytes.Equal(firstJSON, secondJSON) || firstSHA != secondSHA { t.Fatalf("drafts differ") }
}

func TestBuildDraftRejectsProtectedTargetOverlap(t *testing.T) {
	targets := domesticTargets()
	protected := managementTargets()
	protected.Targets = append(protected.Targets, targets.Targets[0])
	_, err := BuildDraft(chinaNode(), targets, protected, overseasHostProfile(), "0.3.0")
	if err == nil || !strings.Contains(err.Error(), "TARGET_OVERLAP") { t.Fatalf("error = %v", err) }
}

func chinaNode() NodeDescriptor { return validNode() }

func domesticTargets() TargetSet {
	return TargetSet{SchemaVersion: SchemaVersion, TargetSetID: "domestic-services", Revision: 1, Purpose: "synthetic domestic APIs", Targets: []TargetRule{
		{Kind: TargetDomain, Value: "api.example.cn", Protocol: "tcp", Ports: []int{443}},
		{Kind: TargetDomainSuffix, Value: "example.cn", Protocol: "tcp", Ports: []int{443}},
	}}
}

func managementTargets() TargetSet {
	return TargetSet{SchemaVersion: SchemaVersion, TargetSetID: "management", Revision: 1, Purpose: "protected management", Targets: []TargetRule{
		{Kind: TargetIPCIDR, Value: "10.0.0.0/8"},
		{Kind: TargetIPCIDR, Value: "127.0.0.0/8"},
	}}
}

func overseasHostProfile() RouteProfile {
	return RouteProfile{SchemaVersion: SchemaVersion, ProfileID: "overseas-host", Name: ProfileGlobalDirectDomesticTunnel, NodeID: "cn-primary", TargetSetID: "domestic-services", ProtectedTargetSetID: "management"}
}

func assertAllDecisions(t *testing.T, got []RouteDecision, targetSetID string, outbound Outbound) {
	t.Helper()
	if len(got) == 0 { t.Fatal("no decisions") }
	for _, decision := range got {
		if decision.TargetSetID != targetSetID || decision.Outbound != outbound { t.Fatalf("decision = %#v", decision) }
	}
}
```

- [ ] **Step 2: Run planner tests and confirm RED**

```powershell
& $go test ./internal/networkcore -run 'TestBuildDraft' -count=1
```

Expected: FAIL because `BuildDraft` and canonical helpers do not exist.

- [ ] **Step 3: Add draft and decision types**

```go
type Outbound string
const (
	OutboundDirect Outbound = "direct"
	OutboundTunnel Outbound = "lcnd-tunnel"
)

type RouteDecision struct {
	TargetSetID string     `json:"target_set_id"`
	Rule        TargetRule `json:"rule"`
	Outbound    Outbound   `json:"outbound"`
}

type CandidateSpec struct {
	Engine        string `json:"engine"`
	EngineVersion string `json:"engine_version"`
	SecretRef     string `json:"secret_ref"`
}

type PlanDraft struct {
	SchemaVersion      string          `json:"schema_version"`
	State              PlanState       `json:"state"`
	ToolVersion        string          `json:"tool_version"`
	Platform           string          `json:"platform"`
	Architecture       string          `json:"architecture"`
	ProfileID          string          `json:"profile_id"`
	NodeID             string          `json:"node_id"`
	NodeDirection      Direction       `json:"node_direction"`
	DefaultOutbound    Outbound        `json:"default_outbound"`
	TunnelOutbound     Outbound        `json:"tunnel_outbound"`
	Decisions          []RouteDecision `json:"decisions"`
	ProtectedDecisions []RouteDecision `json:"protected_decisions"`
	Candidate          CandidateSpec   `json:"candidate"`
	SimulationOnly     bool            `json:"simulation_only"`
}
```

- [ ] **Step 4: Implement canonical JSON and planning**

```go
func CanonicalJSON(value any) ([]byte, string, error) {
	data, err := json.Marshal(value)
	if err != nil { return nil, "", fmt.Errorf("CANONICAL_JSON_FAILED: %w", err) }
	data = append(data, '\n')
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), nil
}

func BuildDraft(node NodeDescriptor, targets, protected TargetSet, profile RouteProfile, toolVersion string) (PlanDraft, error) {
	if issues := ValidateProfileForNode(profile, node); len(issues) != 0 { return PlanDraft{}, ValidationError{Kind: "profile", Issues: issues} }
	if profile.TargetSetID != targets.TargetSetID || profile.ProtectedTargetSetID != protected.TargetSetID { return PlanDraft{}, errors.New("PROFILE_TARGET_REFERENCE_MISMATCH") }
	normalizedTargets, err := normalizeAndSortTargets(targets.Targets)
	if err != nil { return PlanDraft{}, err }
	normalizedProtected, err := normalizeAndSortTargets(protected.Targets)
	if err != nil { return PlanDraft{}, err }
	if overlaps(normalizedTargets, normalizedProtected) { return PlanDraft{}, errors.New("TARGET_OVERLAP") }
	return PlanDraft{
		SchemaVersion: SchemaVersion, State: PlanValidated, ToolVersion: toolVersion,
		Platform: "ubuntu-24.04", Architecture: "amd64", ProfileID: profile.ProfileID,
		NodeID: node.NodeID, NodeDirection: node.Direction,
		DefaultOutbound: OutboundDirect, TunnelOutbound: OutboundTunnel,
		Decisions: decisions(targets.TargetSetID, normalizedTargets, OutboundTunnel),
		ProtectedDecisions: decisions(protected.TargetSetID, normalizedProtected, OutboundDirect),
		Candidate: CandidateSpec{Engine: "sing-box", EngineVersion: SingBoxVersion, SecretRef: node.SecretRef},
		SimulationOnly: true,
	}, nil
}

func normalizeAndSortTargets(input []TargetRule) ([]TargetRule, error) {
	result := append([]TargetRule(nil), input...)
	for index := range result {
		rule := &result[index]
		rule.Protocol = strings.ToLower(strings.TrimSpace(rule.Protocol))
		switch rule.Kind {
		case TargetDomain, TargetDomainSuffix:
			rule.Value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(rule.Value)), ".")
		case TargetIPCIDR:
			prefix, err := netip.ParsePrefix(strings.TrimSpace(rule.Value))
			if err != nil { return nil, errors.New("TARGET_CIDR_INVALID") }
			rule.Value = prefix.Masked().String()
		default:
			return nil, errors.New("TARGET_KIND_UNSUPPORTED")
		}
		sort.Ints(rule.Ports)
		rule.Ports = slices.Compact(rule.Ports)
	}
	sort.Slice(result, func(i, j int) bool { return targetSortKey(result[i]) < targetSortKey(result[j]) })
	return result, nil
}

func overlaps(left, right []TargetRule) bool {
	for _, a := range left {
		for _, b := range right {
			if protocolsOverlap(a.Protocol, b.Protocol) && portsOverlap(a.Ports, b.Ports) && targetsOverlap(a, b) { return true }
		}
	}
	return false
}

func targetsOverlap(a, b TargetRule) bool {
	if a.Kind == TargetIPCIDR && b.Kind == TargetIPCIDR {
		left, leftErr := netip.ParsePrefix(a.Value)
		right, rightErr := netip.ParsePrefix(b.Value)
		return leftErr == nil && rightErr == nil && left.Overlaps(right)
	}
	if a.Kind == TargetIPCIDR || b.Kind == TargetIPCIDR { return false }
	left, right := strings.TrimPrefix(a.Value, "."), strings.TrimPrefix(b.Value, ".")
	return left == right || strings.HasSuffix(left, "."+right) || strings.HasSuffix(right, "."+left)
}
```

`overlaps` must handle exact domain equality, domain/domain-suffix containment, equal normalized ports/protocols, and `netip.Prefix.Overlaps` for CIDR. It must never perform DNS or network I/O.

- [ ] **Step 5: Run planner tests and the whole package**

```powershell
& $go test ./internal/networkcore -run 'TestBuildDraft' -count=1
& $go test ./internal/networkcore -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 3**

```powershell
git add -- internal/networkcore/canonical.go internal/networkcore/planner.go internal/networkcore/planner_test.go internal/networkcore/types.go
git commit -m "feat(network): plan deterministic bidirectional routes"
```

---

### Task 4: Protected sing-box 1.13.12 renderer

**Files:**

- Create: `internal/networkengine/singbox/types.go`
- Create: `internal/networkengine/singbox/render.go`
- Test: `internal/networkengine/singbox/render_test.go`

- [ ] **Step 1: Write failing protected-render tests**

The test creates material in memory, writes only to `t.TempDir`, and asserts the secret appears only in the protected candidate—not in the finalized plan or render summary.

```go
func TestRenderWritesProtectedCandidateAndRedactedPlan(t *testing.T) {
	draft := validDraft(t)
	material := NodeMaterial{
		SchemaVersion: networkcore.SchemaVersion,
		SecretRef: draft.Candidate.SecretRef,
		Outbound: json.RawMessage(`{"type":"shadowsocks","server":"192.0.2.10","server_port":443,"method":"2022-blake3-aes-128-gcm","password":"TEST_ONLY_NOT_A_SECRET"}`),
	}
	candidatePath := filepath.Join(t.TempDir(), "sing-box.json")
	plan, summary, err := RenderToFile(draft, material, candidatePath)
	if err != nil { t.Fatal(err) }
	info, err := os.Stat(candidatePath)
	if err != nil { t.Fatal(err) }
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 { t.Fatalf("mode = %o", info.Mode().Perm()) }
	candidate, _ := os.ReadFile(candidatePath)
	if !bytes.Contains(candidate, []byte("TEST_ONLY_NOT_A_SECRET")) { t.Fatal("candidate missing material") }
	planJSON, _, _ := networkcore.CanonicalJSON(plan)
	summaryJSON, _ := json.Marshal(summary)
	if bytes.Contains(planJSON, []byte("TEST_ONLY_NOT_A_SECRET")) || bytes.Contains(summaryJSON, []byte("TEST_ONLY_NOT_A_SECRET")) { t.Fatal("secret leaked") }
}

func validDraft(t *testing.T) networkcore.PlanDraft {
	t.Helper()
	draft, err := networkcore.BuildDraft(
		networkcore.NodeDescriptor{SchemaVersion: networkcore.SchemaVersion, NodeID: "cn-primary", DisplayName: "Synthetic", Direction: networkcore.DirectionChinaEgress, Engine: "sing-box", SecretRef: "lcnd-secret://nodes/cn-primary", Capabilities: networkcore.Capabilities{TCP: true, IPv4: true}, Enabled: true},
		networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "domestic-services", Revision: 1, Purpose: "synthetic", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetDomain, Value: "api.example.cn", Protocol: "tcp", Ports: []int{443}}}},
		networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "management", Revision: 1, Purpose: "protected", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetIPCIDR, Value: "10.0.0.0/8"}}},
		networkcore.RouteProfile{SchemaVersion: networkcore.SchemaVersion, ProfileID: "overseas-host", Name: networkcore.ProfileGlobalDirectDomesticTunnel, NodeID: "cn-primary", TargetSetID: "domestic-services", ProtectedTargetSetID: "management"},
		"0.3.0",
	)
	if err != nil { t.Fatal(err) }
	return draft
}
```

- [ ] **Step 2: Run renderer tests and confirm RED**

```powershell
& $go test ./internal/networkengine/singbox -count=1
```

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Add protected material and finalized plan types**

```go
type NodeMaterial struct {
	SchemaVersion string          `json:"schema_version"`
	SecretRef     string          `json:"secret_ref"`
	Outbound      json.RawMessage `json:"outbound"`
}

func DecodeMaterial(reader io.Reader) (NodeMaterial, error) {
	var material NodeMaterial
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&material); err != nil { return material, errors.New("MATERIAL_DECODE_FAILED") }
	if material.SchemaVersion != networkcore.SchemaVersion || material.SecretRef == "" || len(material.Outbound) == 0 { return material, errors.New("MATERIAL_INVALID") }
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF { return material, errors.New("MATERIAL_TRAILING_DATA") }
	return material, nil
}

type RenderSummary struct {
	SchemaVersion   string `json:"schema_version"`
	PlanID          string `json:"plan_id"`
	CandidateSHA256 string `json:"candidate_sha256"`
	CandidateBytes  int    `json:"candidate_bytes"`
}
```

Add to `networkcore/types.go`:

```go
type NetworkChangePlan struct {
	SchemaVersion      string          `json:"schema_version"`
	PlanID             string          `json:"plan_id"`
	State              PlanState       `json:"state"`
	ToolVersion        string          `json:"tool_version"`
	Platform           string          `json:"platform"`
	Architecture       string          `json:"architecture"`
	ProfileID          string          `json:"profile_id"`
	NodeID             string          `json:"node_id"`
	NodeDirection      Direction       `json:"node_direction"`
	DefaultOutbound    Outbound        `json:"default_outbound"`
	TunnelOutbound     Outbound        `json:"tunnel_outbound"`
	Decisions          []RouteDecision `json:"decisions"`
	ProtectedDecisions []RouteDecision `json:"protected_decisions"`
	Candidate          CandidateSpec   `json:"candidate"`
	CandidateSHA256    string          `json:"candidate_sha256"`
	SimulationOnly     bool            `json:"simulation_only"`
}
```

- [ ] **Step 4: Implement deterministic 1.13.12 rendering**

Rules must use `action: route`, never removed `geoip`/`geosite` fields. Protected rules come first and route to `direct`; selected targets route to `lcnd-tunnel`; final is `direct`.

```go
func RenderToFile(draft networkcore.PlanDraft, material NodeMaterial, path string) (networkcore.NetworkChangePlan, RenderSummary, error) {
	if draft.State != networkcore.PlanValidated || !draft.SimulationOnly { return networkcore.NetworkChangePlan{}, RenderSummary{}, errors.New("DRAFT_STATE_INVALID") }
	if material.SchemaVersion != networkcore.SchemaVersion || material.SecretRef != draft.Candidate.SecretRef { return networkcore.NetworkChangePlan{}, RenderSummary{}, errors.New("MATERIAL_REFERENCE_MISMATCH") }
	outbound, err := decodeOutbound(material.Outbound)
	if err != nil { return networkcore.NetworkChangePlan{}, RenderSummary{}, err }
	outbound["tag"] = string(networkcore.OutboundTunnel)
	config := buildConfig(draft, outbound)
	candidate, _, err := networkcore.CanonicalJSON(config)
	if err != nil { return networkcore.NetworkChangePlan{}, RenderSummary{}, err }
	if err := writeProtected(path, candidate); err != nil { return networkcore.NetworkChangePlan{}, RenderSummary{}, err }
	candidateSHA := sha256Hex(candidate)
	plan := finalizePlan(draft, candidateSHA)
	plan.PlanID = planDigest(plan)
	return plan, RenderSummary{SchemaVersion: networkcore.SchemaVersion, PlanID: plan.PlanID, CandidateSHA256: candidateSHA, CandidateBytes: len(candidate)}, nil
}

func decodeOutbound(raw json.RawMessage) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var outbound map[string]any
	if err := decoder.Decode(&outbound); err != nil { return nil, errors.New("OUTBOUND_DECODE_FAILED") }
	typeName, ok := outbound["type"].(string)
	if !ok || typeName == "" || typeName == "direct" || typeName == "block" { return nil, errors.New("OUTBOUND_TYPE_INVALID") }
	delete(outbound, "tag")
	return outbound, nil
}

func buildConfig(draft networkcore.PlanDraft, tunnel map[string]any) map[string]any {
	rules := make([]map[string]any, 0, len(draft.ProtectedDecisions)+len(draft.Decisions))
	for _, decision := range draft.ProtectedDecisions { rules = append(rules, ruleForDecision(decision)) }
	for _, decision := range draft.Decisions { rules = append(rules, ruleForDecision(decision)) }
	return map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{map[string]any{"type": "tun", "tag": "lcnd-tun", "interface_name": "lcnd0", "address": []string{"172.19.0.1/30"}, "auto_route": true, "strict_route": true}},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}, tunnel},
		"route": map[string]any{"rules": rules, "final": "direct", "auto_detect_interface": true},
	}
}

func ruleForDecision(decision networkcore.RouteDecision) map[string]any {
	rule := map[string]any{"action": "route", "outbound": string(decision.Outbound)}
	switch decision.Rule.Kind {
	case networkcore.TargetDomain: rule["domain"] = []string{decision.Rule.Value}
	case networkcore.TargetDomainSuffix: rule["domain_suffix"] = []string{decision.Rule.Value}
	case networkcore.TargetIPCIDR: rule["ip_cidr"] = []string{decision.Rule.Value}
	}
	if decision.Rule.Protocol != "" { rule["network"] = decision.Rule.Protocol }
	if len(decision.Rule.Ports) != 0 { rule["port"] = append([]int(nil), decision.Rule.Ports...) }
	return rule
}

func writeProtected(path string, data []byte) error {
	if path == "" || path == "-" { return errors.New("CANDIDATE_PATH_INVALID") }
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil { return fmt.Errorf("CANDIDATE_CREATE_FAILED: %w", err) }
	if _, err := file.Write(data); err != nil { _ = file.Close(); return fmt.Errorf("CANDIDATE_WRITE_FAILED: %w", err) }
	return file.Close()
}

func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func finalizePlan(draft networkcore.PlanDraft, candidateSHA string) networkcore.NetworkChangePlan {
	return networkcore.NetworkChangePlan{SchemaVersion: draft.SchemaVersion, State: networkcore.PlanReady, ToolVersion: draft.ToolVersion, Platform: draft.Platform, Architecture: draft.Architecture, ProfileID: draft.ProfileID, NodeID: draft.NodeID, NodeDirection: draft.NodeDirection, DefaultOutbound: draft.DefaultOutbound, TunnelOutbound: draft.TunnelOutbound, Decisions: append([]networkcore.RouteDecision(nil), draft.Decisions...), ProtectedDecisions: append([]networkcore.RouteDecision(nil), draft.ProtectedDecisions...), Candidate: draft.Candidate, CandidateSHA256: candidateSHA, SimulationOnly: true}
}

func planDigest(plan networkcore.NetworkChangePlan) string {
	copy := plan
	copy.PlanID = ""
	_, digest, err := networkcore.CanonicalJSON(copy)
	if err != nil { panic(err) }
	return digest
}
```

The generated config shape is:

```json
{
  "log": {"level": "warn"},
  "inbounds": [{"type": "tun", "tag": "lcnd-tun", "interface_name": "lcnd0", "address": ["172.19.0.1/30"], "auto_route": true, "strict_route": true}],
  "outbounds": [
    {"type": "direct", "tag": "direct"},
    {"type": "<from protected material>", "tag": "lcnd-tunnel"}
  ],
  "route": {"rules": [], "final": "direct", "auto_detect_interface": true}
}
```

`writeProtected` must use `os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)` and reject stdout (`path == "-"`) and existing files.

- [ ] **Step 5: Add a redacted golden skeleton assertion**

Normalize the candidate by replacing the outbound object with `{ "type": "REDACTED", "tag": "lcnd-tunnel" }` before comparing it with an inline expected JSON value. Do not commit a secret-bearing candidate fixture.

- [ ] **Step 6: Run and commit Task 4**

```powershell
& $go test ./internal/networkengine/singbox ./internal/networkcore -count=1
git add -- internal/networkengine/singbox internal/networkcore/types.go schemas/network-change-plan.schema.json
git commit -m "feat(network): render protected sing-box candidates"
```

---

### Task 5: Simulation workflow and synthetic health report

**Files:**

- Create: `internal/networkcore/simulate.go`
- Create: `internal/networkcore/report.go`
- Test: `internal/networkcore/simulate_test.go`
- Test: `internal/networkcore/report_test.go`

- [ ] **Step 1: Write failing state-transition tests**

```go
func TestSimulateSuccessProducesVerifiedReceipt(t *testing.T) {
	receipt, err := Simulate(validPlan(), ScenarioSuccess)
	if err != nil { t.Fatal(err) }
	if receipt.State != ExecutionVerified || receipt.RollbackTriggered { t.Fatalf("receipt = %+v", receipt) }
}

func TestSimulateVerificationFailureRollsBack(t *testing.T) {
	receipt, err := Simulate(validPlan(), ScenarioVerificationFailed)
	if err != nil { t.Fatal(err) }
	if receipt.State != ExecutionRolledBack || !receipt.RollbackTriggered { t.Fatalf("receipt = %+v", receipt) }
}

func TestReportIsSyntheticAndNeverClaimsLiveHealth(t *testing.T) {
	report, err := BuildReport(validPlan(), verifiedReceipt())
	if err != nil { t.Fatal(err) }
	if report.EvidenceMode != EvidenceSynthetic || report.Status != HealthSimulatedPass { t.Fatalf("report = %+v", report) }
	if strings.Contains(strings.ToLower(report.Summary), "production") { t.Fatalf("summary = %q", report.Summary) }
}

func validPlan() NetworkChangePlan {
	return NetworkChangePlan{SchemaVersion: SchemaVersion, PlanID: strings.Repeat("a", 64), State: PlanReady, ToolVersion: "0.3.0", Platform: "ubuntu-24.04", Architecture: "amd64", ProfileID: "overseas-host", NodeID: "cn-primary", NodeDirection: DirectionChinaEgress, DefaultOutbound: OutboundDirect, TunnelOutbound: OutboundTunnel, Candidate: CandidateSpec{Engine: "sing-box", EngineVersion: SingBoxVersion, SecretRef: "lcnd-secret://nodes/cn-primary"}, CandidateSHA256: strings.Repeat("b", 64), SimulationOnly: true}
}

func verifiedReceipt() ExecutionReceipt {
	receipt, err := Simulate(validPlan(), ScenarioSuccess)
	if err != nil { panic(err) }
	return receipt
}
```

- [ ] **Step 2: Run tests and confirm RED**

```powershell
& $go test ./internal/networkcore -run 'Test(Simulate|Report)' -count=1
```

- [ ] **Step 3: Implement receipt/report types and transitions**

```go
type ExecutionState string
const (
	ExecutionVerified   ExecutionState = "VERIFIED"
	ExecutionRolledBack ExecutionState = "ROLLED_BACK"
)

type ActionResult struct {
	Action string `json:"action"`
	Status string `json:"status"`
	Code   string `json:"code"`
}

type ExecutionReceipt struct {
	SchemaVersion     string         `json:"schema_version"`
	PlanID            string         `json:"plan_id"`
	EvidenceMode      string         `json:"evidence_mode"`
	Scenario          Scenario       `json:"scenario"`
	State             ExecutionState `json:"state"`
	CandidateSHA256   string         `json:"candidate_sha256"`
	RollbackArmed     bool           `json:"rollback_armed"`
	RollbackTriggered bool           `json:"rollback_triggered"`
	Actions           []ActionResult `json:"actions"`
}

type EvidenceMode string
const EvidenceSynthetic EvidenceMode = "synthetic"

type HealthStatus string
const (
	HealthSimulatedPass     HealthStatus = "SIMULATED_PASS"
	HealthSimulatedRollback HealthStatus = "SIMULATED_ROLLBACK"
)

type Finding struct {
	Code        string `json:"code"`
	Scope       string `json:"scope"`
	Explanation string `json:"explanation"`
	Action      string `json:"action"`
}

type NetworkHealthReport struct {
	SchemaVersion string       `json:"schema_version"`
	PlanID        string       `json:"plan_id"`
	EvidenceMode  EvidenceMode `json:"evidence_mode"`
	Status        HealthStatus `json:"status"`
	Summary       string       `json:"summary"`
	Findings      []Finding    `json:"findings"`
}

func Simulate(plan NetworkChangePlan, scenario Scenario) (ExecutionReceipt, error) {
	if plan.State != PlanReady || !plan.SimulationOnly || plan.PlanID == "" || plan.CandidateSHA256 == "" { return ExecutionReceipt{}, errors.New("PLAN_NOT_SIMULATABLE") }
	receipt := baseReceipt(plan, scenario)
	switch scenario {
	case ScenarioSuccess:
		receipt.State = ExecutionVerified
	case ScenarioVerificationFailed, ScenarioTimeout:
		receipt.State = ExecutionRolledBack
		receipt.RollbackTriggered = true
	default:
		return ExecutionReceipt{}, errors.New("SCENARIO_UNSUPPORTED")
	}
	return receipt, nil
}
```

`BuildReport` requires matching `plan_id` and candidate digest, always sets `evidence_mode: synthetic`, and maps scenarios to `SIMULATED_PASS` or `SIMULATED_ROLLBACK` rather than `HEALTHY`/`UNHEALTHY`.

```go
func BuildReport(plan NetworkChangePlan, receipt ExecutionReceipt) (NetworkHealthReport, error) {
	if receipt.PlanID != plan.PlanID || receipt.CandidateSHA256 != plan.CandidateSHA256 { return NetworkHealthReport{}, errors.New("RECEIPT_PLAN_MISMATCH") }
	report := NetworkHealthReport{SchemaVersion: SchemaVersion, PlanID: plan.PlanID, EvidenceMode: EvidenceSynthetic}
	switch receipt.State {
	case ExecutionVerified:
		report.Status = HealthSimulatedPass
		report.Summary = "Synthetic execution reached the verified state; no live Ubuntu health claim was made."
		report.Findings = []Finding{{Code: "SIMULATION_VERIFIED", Scope: "simulation", Explanation: "All simulated guards completed.", Action: "Proceed only to an approved live-read-only slice."}}
	case ExecutionRolledBack:
		report.Status = HealthSimulatedRollback
		report.Summary = "Synthetic execution exercised the rollback path; no production change occurred."
		report.Findings = []Finding{{Code: "SIMULATION_ROLLED_BACK", Scope: "simulation", Explanation: "A simulated failure or timeout triggered rollback.", Action: "Inspect the scenario evidence before another simulation."}}
	default:
		return NetworkHealthReport{}, errors.New("RECEIPT_STATE_UNSUPPORTED")
	}
	return report, nil
}
```

- [ ] **Step 4: Run tests and update receipt/report schemas**

```powershell
& $go test ./internal/networkcore -run 'Test(Simulate|Report)' -count=1
& $go test ./internal/projectassets -run TestJSONSchemasAreValidJSON -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 5**

```powershell
git add -- internal/networkcore/simulate.go internal/networkcore/simulate_test.go internal/networkcore/report.go internal/networkcore/report_test.go internal/networkcore/types.go schemas/network-execution-receipt.schema.json schemas/network-health-report.schema.json
git commit -m "feat(network): simulate guarded network workflows"
```

---

### Task 6: `cnd-network` CLI

**Files:**

- Create: `internal/networkcli/cli.go`
- Create: `internal/networkcli/cli_test.go`
- Create: `cmd/cnd-network/main.go`
- Create: `cmd/cnd-network/main_test.go`

- [ ] **Step 1: Write CLI tests before implementation**

Cover version, unknown command, strict validation, deterministic plan output, protected rendering, simulation, report generation, refusal to overwrite, and secret non-disclosure.

```go
func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{"version"}, &stdout, &stderr, "0.3.0")
	if exit != 0 || stdout.String() != "cnd-network 0.3.0\n" { t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String()) }
}

func TestRunRenderDoesNotPrintMaterial(t *testing.T) {
	paths := writeSyntheticInputs(t)
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{
		"render", "--draft", paths.Draft, "--material", paths.Material,
		"--candidate", paths.Candidate, "--plan", paths.Plan, "--format", "json",
	}, &stdout, &stderr, "0.3.0")
	if exit != 0 { t.Fatalf("exit=%d stderr=%q", exit, stderr.String()) }
	if strings.Contains(stdout.String(), "TEST_ONLY_NOT_A_SECRET") || strings.Contains(stderr.String(), "TEST_ONLY_NOT_A_SECRET") { t.Fatal("secret leaked to process output") }
}

type syntheticPaths struct { Draft, Material, Candidate, Plan string }

func writeSyntheticInputs(t *testing.T) syntheticPaths {
	t.Helper()
	root := t.TempDir()
	draft, err := networkcore.BuildDraft(
		networkcore.NodeDescriptor{SchemaVersion: networkcore.SchemaVersion, NodeID: "cn-primary", DisplayName: "Synthetic", Direction: networkcore.DirectionChinaEgress, Engine: "sing-box", SecretRef: "lcnd-secret://nodes/cn-primary", Capabilities: networkcore.Capabilities{TCP: true, IPv4: true}, Enabled: true},
		networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "domestic-services", Revision: 1, Purpose: "synthetic", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetDomain, Value: "api.example.cn", Protocol: "tcp", Ports: []int{443}}}},
		networkcore.TargetSet{SchemaVersion: networkcore.SchemaVersion, TargetSetID: "management", Revision: 1, Purpose: "protected", Targets: []networkcore.TargetRule{{Kind: networkcore.TargetIPCIDR, Value: "10.0.0.0/8"}}},
		networkcore.RouteProfile{SchemaVersion: networkcore.SchemaVersion, ProfileID: "overseas-host", Name: networkcore.ProfileGlobalDirectDomesticTunnel, NodeID: "cn-primary", TargetSetID: "domestic-services", ProtectedTargetSetID: "management"},
		"0.3.0",
	)
	if err != nil { t.Fatal(err) }
	draftData, _, _ := networkcore.CanonicalJSON(draft)
	draftPath := filepath.Join(root, "draft.json")
	materialPath := filepath.Join(root, "material.json")
	if err := os.WriteFile(draftPath, draftData, 0o600); err != nil { t.Fatal(err) }
	material := `{"schema_version":"1.0.0","secret_ref":"lcnd-secret://nodes/cn-primary","outbound":{"type":"shadowsocks","server":"192.0.2.10","server_port":443,"method":"2022-blake3-aes-128-gcm","password":"TEST_ONLY_NOT_A_SECRET"}}`
	if err := os.WriteFile(materialPath, []byte(material), 0o600); err != nil { t.Fatal(err) }
	return syntheticPaths{Draft: draftPath, Material: materialPath, Candidate: filepath.Join(root, "candidate.json"), Plan: filepath.Join(root, "plan.json")}
}
```

- [ ] **Step 2: Run CLI tests and confirm RED**

```powershell
& $go test ./internal/networkcli ./cmd/cnd-network -count=1
```

Expected: FAIL because packages do not exist.

- [ ] **Step 3: Implement the CLI dispatcher and safe file helpers**

```go
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 { printUsage(stderr); return 2 }
	switch args[0] {
	case "version", "--version", "-version":
		_, _ = fmt.Fprintf(stdout, "cnd-network %s\n", version)
		return 0
	case "validate": return runValidate(args[1:], stdout, stderr)
	case "plan": return runPlan(args[1:], stdout, stderr, version)
	case "render": return runRender(args[1:], stdout, stderr)
	case "simulate": return runSimulate(args[1:], stdout, stderr)
	case "report": return runReport(args[1:], stdout, stderr)
	case "help", "--help", "-h": printUsage(stdout); return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}
```

All input reads use a 1 MiB limit. Public output files use `O_EXCL` and mode `0644`; secret material input is read-only; the candidate uses the renderer's `0600` writer. Error messages contain file roles (`node`, `draft`, `material`) but never file contents.

- [ ] **Step 4: Implement each command with exact state gates**

```text
validate: decode one selected public contract and print ValidationReport
plan:     decode node/targets/protected/profile, BuildDraft, write canonical draft
render:   decode draft and NodeMaterial, RenderToFile, write canonical finalized plan
simulate: decode finalized plan, Simulate, write canonical receipt
report:   decode plan and receipt, BuildReport, write canonical report
```

`--format` defaults to `text`. JSON mode prints only a redacted summary object; no command prints candidate bytes or node material.

- [ ] **Step 5: Add the process entry point**

```go
package main

import (
	"context"
	"os"

	"github.com/dwg7318/codex-network-doctor/internal/networkcli"
)

var version = "0.1.0-dev"

func main() {
	os.Exit(networkcli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, version))
}
```

- [ ] **Step 6: Run CLI tests and a manual synthetic chain**

```powershell
& $go test ./internal/networkcli ./cmd/cnd-network -count=1
& $go run ./cmd/cnd-network version
```

Expected:

```text
cnd-network 0.1.0-dev
```

Run the full chain with only files under `.codex/.tmp/network-smoke`; verify candidate output is not printed and all public artifacts are JSON-decodable. Delete the temporary directory through PowerShell after resolving and confirming it is inside `D:\LCnd\.codex\.tmp`.

- [ ] **Step 7: Commit Task 6**

```powershell
git add -- internal/networkcli cmd/cnd-network
git commit -m "feat(network): add simulation-only network CLI"
```

---

### Task 7: Build integration, documentation, and version preparation

**Files:**

- Modify: `scripts/build.ps1`
- Create: `scripts/build-linux.ps1`
- Modify: `internal/projectassets/assets_test.go`
- Create: `docs/RFC/0003-ubuntu-network-core.md`
- Modify: `README.md`
- Modify: `docs/MASTER_SPEC.md`
- Modify: `docs/SDD.md`
- Modify: `docs/TEST_HANDBOOK.md`
- Modify: `docs/BACKLOG.md`
- Modify after all tests pass: `VERSION`
- Modify after all tests pass: `CHANGELOG.md`

- [ ] **Step 1: Write build-asset tests first**

Extend `TestBuildScriptConfiguresWindowsRaceToolchain` or add a focused test requiring:

```go
for _, required := range []string{"cmd\\cnd-network", "cnd-network.exe"} {
	if !strings.Contains(script, required) { t.Errorf("build script missing %q", required) }
}
linuxScript := readAsset(t, "scripts/build-linux.ps1")
for _, required := range []string{"GOOS", "linux", "GOARCH", "amd64", "CGO_ENABLED", "cmd\\cnd-network"} {
	if !strings.Contains(linuxScript, required) { t.Errorf("linux build script missing %q", required) }
}

func readAsset(t *testing.T, relativePath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(relativePath)))
	if err != nil { t.Fatal(err) }
	return string(data)
}
```

- [ ] **Step 2: Run project asset tests and confirm RED**

```powershell
& $go test ./internal/projectassets -run 'Test(Build|RequiredEngineeringAssets)' -count=1
```

- [ ] **Step 3: Extend the Windows build without changing existing names**

After building existing executables in `scripts/build.ps1`, add:

```powershell
$networkExecutable = Join-Path $OutputDirectory 'cnd-network.exe'
& $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $networkExecutable .\cmd\cnd-network
if ($LASTEXITCODE -ne 0) {
    throw 'go build cnd-network failed'
}
```

Append `$networkExecutable` to final output. Do not rename or remove the four existing Windows executables.

- [ ] **Step 4: Add a side-effect-free Ubuntu cross-build script**

`scripts/build-linux.ps1` must save and restore environment variables in `finally`:

```powershell
[CmdletBinding()]
param([string]$Version, [string]$OutputDirectory)

$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$go = Join-Path $repositoryRoot '.tools\go\bin\go.exe'
if (-not (Test-Path -LiteralPath $go)) { $go = (Get-Command go -ErrorAction Stop).Source }
if ([string]::IsNullOrWhiteSpace($Version)) { $Version = (Get-Content -Raw -LiteralPath (Join-Path $repositoryRoot 'VERSION')).Trim() }
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) { $OutputDirectory = Join-Path $repositoryRoot 'dist\linux-amd64' }
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$oldGOOS, $oldGOARCH, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
try {
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    & $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $OutputDirectory 'cnd-network') .\cmd\cnd-network
    if ($LASTEXITCODE -ne 0) { throw 'linux cnd-network build failed' }
}
finally {
    $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $oldGOOS, $oldGOARCH, $oldCGO
}
Write-Output (Join-Path $OutputDirectory 'cnd-network')
```

This script builds only; it does not install Docker, sing-box, systemd units, or network configuration.

- [ ] **Step 5: Write durable product documentation**

Document these exact boundaries:

```text
- cnd.exe remains passive evidence collection.
- cnd-network is a separate simulation-only Ubuntu network planning CLI in v0.3.0.
- sing-box target is 1.13.12, but the binary is not bundled or installed.
- all samples are synthetic and all generated reports use evidence_mode=synthetic.
- LCSP integration, real nodes, sudo/systemd, and production routing are deferred.
- node material and generated candidate configuration are sensitive assets.
```

`docs/RFC/0003-ubuntu-network-core.md` must define CLI commands, files, exit codes, state machine, candidate secrecy, and incompatibility behavior.

- [ ] **Step 6: Run all pre-version quality gates**

```powershell
& $go test ./... -count=1
& $go vet ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1 -SkipRace
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-linux.ps1
```

Expected: every command exits 0; Windows build lists five executables; Linux build lists `dist\linux-amd64\cnd-network`.

- [ ] **Step 7: Update version only after Step 6 passes**

Set `VERSION` to:

```text
0.3.0
```

Add `## v0.3.0 — 2026-08-12` to `CHANGELOG.md`, explicitly calling it simulation-only and listing LCSP/live-node execution as deferred. Re-run both build scripts so embedded versions match.

- [ ] **Step 8: Commit Task 7**

```powershell
git add -- scripts/build.ps1 scripts/build-linux.ps1 internal/projectassets/assets_test.go README.md docs/MASTER_SPEC.md docs/SDD.md docs/TEST_HANDBOOK.md docs/BACKLOG.md docs/RFC/0003-ubuntu-network-core.md VERSION CHANGELOG.md
git commit -m "release: prepare cnd-network v0.3.0"
```

Do not create or push tag `v0.3.0` until Owner explicitly approves the completed release evidence.

---

### Task 8: Final acceptance and evidence review

**Files:**

- Modify only if evidence reveals a gap: files already listed above
- Do not create production credentials, node files, or deployment scripts

- [ ] **Step 1: Run focused non-cached tests**

```powershell
& $go test ./internal/networkcore ./internal/networkengine/singbox ./internal/networkcli ./cmd/cnd-network -count=1
```

Expected: PASS, zero failures.

- [ ] **Step 2: Run repository regression gates**

```powershell
& $go test ./... -count=1
& $go vet ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1 -SkipRace
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-linux.ps1
```

Expected: PASS; existing four Windows outputs plus `cnd-network.exe`; Ubuntu cross-build succeeds.

- [ ] **Step 3: Run leak and forbidden-action scans**

```powershell
rg -n --hidden -g '!\.git/**' -g '!\.codex/**' '(password|private_key|subscription_url|access_token|Bearer )' schemas samples docs/RFC/0003-ubuntu-network-core.md
rg -n -g '*.go' '(sudo|systemctl|nft|iptables|ip route|exec\.Command)' cmd/cnd-network internal/networkcore internal/networkengine internal/networkcli
```

Expected: first scan has no real secret values; second scan has no host mutation primitives. Documentation mentions are reviewed manually and not mistaken for executable behavior.

- [ ] **Step 4: Verify exact Git scope and generated artifacts**

```powershell
git status --short
git diff --check origin/main...HEAD
git diff --stat origin/main...HEAD
git log --oneline --decorate origin/main..HEAD
```

Expected: only LCnd files from this plan plus local ignored/untracked `.codex`; no LCSP paths, real node data, candidate secrets, binaries, `dist`, or `bin` are staged.

- [ ] **Step 5: Produce a completion report without overclaiming**

Report separately:

```text
Implemented: contracts, planner, protected renderer, simulator, CLI, builds, docs
Focused tests: exact command and pass result
Regression gates: exact command and pass result
Ubuntu status: cross-compiled only; no live Ubuntu acceptance
LCSP status: deferred; no LCSP changes
Node status: synthetic only; no real nodes imported
Production status: not deployed, not authorized, not claimed
Next recommended slice: live Ubuntu read-only inventory and real sing-box check, after separate approval
```

Do not mark the full cross-region networking goal complete. Close only this simulation-core slice.

---

## Implementation stop rules

Stop and request Owner direction if any task requires:

- modifying LCSP or assuming an unfinished LCSP API;
- a real node credential, subscription URL, SSH key, server address, or production secret;
- sudo/root, systemd mutation, Docker installation, firewall/routing changes, or production access;
- weakening `cnd.exe` passive behavior or existing Windows repair safety;
- adding an unpinned external Go dependency or downloading a binary;
- changing the approved LCnd/LCSP product ownership boundary.

Failures inside the synthetic/TDD scope are not blockers: reduce to the current task, fix tests, and continue without opening later tasks.
