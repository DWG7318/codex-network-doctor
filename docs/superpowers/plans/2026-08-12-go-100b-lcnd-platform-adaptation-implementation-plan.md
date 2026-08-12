# GO-100B LCnd Platform Adaptation Implementation Plan

> For agentic workers: REQUIRED SUB-SKILL: use `superpowers:subagent-driven-development` or `superpowers:executing-plans` when implementing this plan. Work tracking uses GO -> CELL only.

**Goal:** Build the LCnd-side continuity delivery package for an independent `lcsp-continuity-b` line, plus the reusable contract pieces needed later by `lcnd-egress-cn`, without touching LCSP, LCapi, LCgpu, production servers, or the old public TCP/443 line.

**Architecture:** LCnd remains the network domain owner: it produces deterministic, non-secret plans, candidate summaries, direct/egress probe definitions, protected-target guards, rollback triggers, and evidence expectations. LCSP remains the server platform authority and is the only future executor for root, systemd, endpoint, secret, backup, and rollback operations. `lcsp-continuity-b` and `lcnd-egress-cn` are always separate workloads, endpoints, secrets, receipts, and rollback domains.

**Tech Stack:** Go 1.26, existing `internal/networkcore`, existing `internal/networkengine/singbox`, existing `internal/networkcli`, JSON Schema draft 2020-12, deterministic JSON/SHA-256, repository-local PowerShell validation, GitHub branch/PR/main flow.

**Authoritative Input:** `D:\LCSP\docs\superpowers\specs\2026-08-12-lcsp-multi-workload-platform-and-dual-line-design.md`.

---

## GO Boundary

GO-100B is an LCnd repository planning and future implementation line. It does not grant production authority.

The old line is immutable:

- public TCP/443 ownership remains HAProxy;
- existing Xray backend, HAProxy config, client node config, and current user VPN path are not edited, restarted, wrapped, or reused;
- SSH TCP/22, HTTP TCP/80, public TCP/443, UDP/443, LCSP Access, Caddy, Xray, Shadowsocks, Tailnet, LCapi, LCgpu, and current client proxy settings are protected targets;
- any future LCnd plan that needs to alter those assets must fail closed before it becomes executable.

The new continuity line is additive:

- `lcsp-continuity-b` is an independent LCSP-hosted workload for management and Codex continuity;
- it must use a distinct service identity, endpoint class, config root, state root, secret refs, logs, backup marker, rollback marker, health probes, and release digest;
- it must not run inside the old Xray or HAProxy line and must not listen on public TCP/443;
- first preference is a private Tailnet endpoint when LCSP approves it; a dedicated public non-web endpoint requires a separate Owner authorization.

The later application egress is separate:

- `lcnd-egress-cn` is for approved application traffic such as LCapi domestic provider/model requests;
- it has its own manifest, endpoint, secret refs, probes, rollback trigger set, and consumer attachment;
- failure in `lcnd-egress-cn` may affect only opted-in consumer requests and must not affect `lcsp-continuity-b`.

## File Map

Create these LCnd files:

```text
D:\LCnd\docs\RFC\0004-lcnd-continuity-b-contract.md
D:\LCnd\schemas\network-continuity-plan.schema.json
D:\LCnd\schemas\network-secret-ref.schema.json
D:\LCnd\schemas\network-probe-plan.schema.json
D:\LCnd\samples\network\continuity-b.synthetic.json
D:\LCnd\samples\network\continuity-b-probes.synthetic.json
D:\LCnd\samples\network\protected-platform.synthetic.json
D:\LCnd\internal\networkcore\continuity.go
D:\LCnd\internal\networkcore\continuity_test.go
D:\LCnd\internal\networkcore\secretref.go
D:\LCnd\internal\networkcore\secretref_test.go
D:\LCnd\internal\networkcore\probeplan.go
D:\LCnd\internal\networkcore\probeplan_test.go
D:\LCnd\internal\networkcore\protection.go
D:\LCnd\internal\networkcore\protection_test.go
D:\LCnd\internal\networkcore\rollback.go
D:\LCnd\internal\networkcore\rollback_test.go
```

Modify only these LCnd files:

```text
D:\LCnd\cmd\cnd-network\main.go
D:\LCnd\cmd\cnd-network\main_test.go
D:\LCnd\internal\networkcli\cli.go
D:\LCnd\internal\networkcli\cli_test.go
D:\LCnd\internal\networkengine\singbox\render.go
D:\LCnd\internal\networkengine\singbox\render_test.go
D:\LCnd\internal\projectassets\assets_test.go
D:\LCnd\docs\MASTER_SPEC.md
D:\LCnd\docs\SDD.md
D:\LCnd\docs\TEST_HANDBOOK.md
D:\LCnd\docs\BACKLOG.md
D:\LCnd\README.md
D:\LCnd\CHANGELOG.md
D:\LCnd\VERSION
```

Do not modify these paths:

```text
D:\LCSP\**
D:\LCapi\**
D:\LCgpu\**
D:\LCnd\internal\tunrepair\**
D:\LCnd\internal\proxyrepair\**
D:\LCnd\internal\speedtest\**
D:\LCnd\cmd\cnd\**
D:\LCnd\cmd\cnd-repair\**
```

Do not commit:

```text
D:\LCnd\.codex\**
D:\LCnd\docs\LCSP-LCND-PRIVATE-EGRESS-REQUIREMENT.md
any real node URL, subscription URL, password, token, cookie, SSH material, server config, raw diagnostic ZIP, or generated candidate containing secrets
```

## Contract Shape

Use these public, non-secret fields for `network-continuity-plan.schema.json` and matching Go types:

```json
{
  "schema_version": "1.0.0",
  "plan_id": "lcnd-plan-20260812-continuity-b-synthetic",
  "purpose": "lcsp_continuity_b",
  "target_platform": "ubuntu-24.04-amd64",
  "lcsp_host_ref": "lcsp://hosts/owner-approved-primary",
  "workload": {
    "id": "lcsp-continuity-b",
    "class": "management_continuity",
    "endpoint_class": "tailnet_private_tcp",
    "public_ports_forbidden": [22, 80, 443],
    "separate_from": ["old-public-tcp-443", "lcnd-egress-cn"]
  },
  "engine": {
    "name": "sing-box",
    "version": "1.13.12",
    "candidate_sha256": "sha256:7265a50cf0ba3069353b7958dc98d138802ad94da92c6443b070921679d8ca6b"
  },
  "secret_refs": [
    {
      "ref": "lcnd-secret://lcsp/continuity-b/server-node",
      "consumer": "lcsp-continuity-b",
      "material_class": "engine_runtime_config",
      "value_visibility": "never_in_git_stdout_logs_or_receipts"
    }
  ],
  "protected_targets_ref": "lcnd://samples/network/protected-platform.synthetic.json",
  "probe_plan_ref": "lcnd://samples/network/continuity-b-probes.synthetic.json",
  "rollback_domain": "lcsp-continuity-b",
  "execution_authority": "lcsp_required",
  "production_authorized": false
}
```

The sample `candidate_sha256` is closed over the fixed UTF-8 synthetic input `synthetic-lcsp-continuity-b-candidate-v1`. It is not a digest of a real server candidate or node secret.

Use these public, non-secret fields for `network-secret-ref.schema.json`:

```json
{
  "schema_version": "1.0.0",
  "ref": "lcnd-secret://lcsp/continuity-b/server-node",
  "owner": "LCSP",
  "consumer": "lcsp-continuity-b",
  "material_class": "engine_runtime_config",
  "rotation_policy": "owner_managed",
  "allowed_outputs": ["redacted_digest", "presence_status"],
  "forbidden_outputs": ["value", "url", "password", "token", "subscription", "private_key"]
}
```

Use these public, non-secret probe classes for `network-probe-plan.schema.json`:

```text
direct_control_probe
direct_application_probe
egress_continuity_probe
egress_application_probe
protected_target_probe
rollback_probe
secret_leak_probe
```

## Protected Target Set

`D:\LCnd\samples\network\protected-platform.synthetic.json` must include these synthetic entries and the domain validator must require equivalent entries in any real LCSP plan:

```text
old-public-tcp-443-owner
old-haproxy-config-fingerprint
old-xray-service-fingerprint
old-client-node-config-fingerprint
ssh-tcp-22
http-tcp-80
udp-443
lcsp-access-loopback-18082
lcapi-loopback-18080
lcgpu-gateway-loopback-18081
owner-console-loopback-19982
tailnet-existing-preferences
system-default-route
system-global-dns
system-firewall-rules
system-proxy
```

Validation rule: a continuity plan is rejected when any candidate endpoint, route rule, DNS rule, listener, consumer, or rollback action overlaps this set unless the overlap is read-only evidence collection.

## Direct And Egress Probe Rules

Define probes as data, not live network calls in this GO:

```text
direct_control_probe:
  route: current host direct path
  targets: SSH 22, LCSP Access, old public TCP/443 fingerprint
  pass: reachable or unchanged according to LCSP read-only receipt
  fail: any ownership or fingerprint drift

egress_continuity_probe:
  route: client -> lcsp-continuity-b -> approved outside target
  pass: three successful TCP/TLS/HTTP samples and stable endpoint identity
  fail: endpoint collision, timeout quorum failure, or secret leak

direct_application_probe:
  route: LCapi existing path
  targets: non-opted-in providers and overseas providers
  pass: existing behavior remains in direct baseline
  fail: any non-opted-in traffic bound to new egress

egress_application_probe:
  route: LCapi opted-in provider -> lcnd-egress-cn -> domestic target
  pass: three successful samples and explicit provider/model binding
  fail: silent fallback to unknown route or broad consumer attachment
```

Probe output must record sample count, timeout, status class, route label, redacted endpoint identity, and evidence mode. It must not record full URLs, credentials, tokens, request bodies, or response bodies.

## Rollback Triggers

Any future LCSP execution receipt must demand rollback or fail-closed planning when one of these triggers appears:

```text
PROTECTED_TARGET_DRIFT
OLD_LINE_HEALTH_REGRESSION
PUBLIC_443_COLLISION
SSH_OR_LCSP_ACCESS_REGRESSION
GLOBAL_ROUTE_CHANGED
GLOBAL_DNS_CHANGED
FIREWALL_CHANGED
SYSTEM_PROXY_CHANGED
TAILNET_PREFERENCE_CHANGED
LCGPU_CONTROL_CHANGED
LCAPI_NON_OPTED_TRAFFIC_CHANGED
SECRET_VALUE_EXPOSED
CANDIDATE_DIGEST_MISMATCH
WORKLOAD_DOMAIN_COLLISION
PROBE_TIMEOUT_QUORUM_FAILED
RESOURCE_BUDGET_EXCEEDED
ROLLBACK_NOT_ARMED
PRODUCTION_AUTHORITY_MISSING
```

`lcsp-continuity-b` rollback may stop and remove only the continuity workload and its private endpoint. `lcnd-egress-cn` rollback may detach only its approved consumers and remove only its application egress workload. Neither rollback path may edit old TCP/443, HAProxy, Xray, SSH, Tailnet preferences, LCapi default route, or LCgpu control paths.

## CELL List

GO-100B contains exactly six CELL outcomes, matching the LCSP total engineering graph: continuity plan, independent new-line candidate, secret schema, direct/egress probes, protected targets, and rollback.

### CELL-100B.1 Continuity Plan Contract

Files:

```text
D:\LCnd\docs\RFC\0004-lcnd-continuity-b-contract.md
D:\LCnd\schemas\network-continuity-plan.schema.json
D:\LCnd\internal\networkcore\continuity.go
D:\LCnd\internal\networkcore\continuity_test.go
D:\LCnd\samples\network\continuity-b.synthetic.json
D:\LCnd\docs\MASTER_SPEC.md
D:\LCnd\docs\BACKLOG.md
```

TDD and verification commands:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkcore -run 'TestContinuityPlan' -count=1
rg -n "lcsp-continuity-b|lcnd-egress-cn|public TCP/443|HAProxy|Xray|secret_ref|rollback" D:\LCnd\docs D:\LCnd\samples\network
$bad = @('T'+'BD','T'+'ODO','FIX'+'ME','St'+'ep','st'+'eps')
foreach ($p in $bad) { rg -n $p D:\LCnd\docs\RFC\0004-lcnd-continuity-b-contract.md; if ($LASTEXITCODE -eq 0) { exit 1 } }
```

Required tests:

```text
accept purpose lcsp_continuity_b with workload id lcsp-continuity-b
accept future purpose lcnd_egress_cn only when workload id is lcnd-egress-cn
reject a plan that gives both purposes the same workload id
reject public port 443, 80, 22, or udp/443 listeners
reject production_authorized true in repository samples
reject execution_authority other than lcsp_required for LCSP-managed hosts
reject candidate digest mismatch
```

Acceptance evidence:

```text
RFC states old public TCP/443 immutability.
RFC states LCSP-only execution authority.
RFC states continuity and application egress separation.
Synthetic plan is deterministic across two serial encodes.
Continuity plan cannot be interpreted as an application egress attachment.
Backlog records future live execution as blocked on LCSP and Owner production authority.
```

### CELL-100B.2 Independent New-Line Candidate

Files:

```text
D:\LCnd\internal\networkengine\singbox\render.go
D:\LCnd\internal\networkengine\singbox\render_test.go
D:\LCnd\internal\networkcli\cli.go
D:\LCnd\internal\networkcli\cli_test.go
D:\LCnd\cmd\cnd-network\main.go
D:\LCnd\cmd\cnd-network\main_test.go
```

Add these read-only or file-generation commands:

```text
cnd-network continuity validate --input PLAN.json --format text|json
cnd-network continuity candidate-summary --input PLAN.json --output SUMMARY.json --format text|json
```

TDD and verification commands:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkengine/singbox ./internal/networkcli ./cmd/cnd-network -run 'TestContinuity|TestCLI|TestRender' -count=1
```

Required tests:

```text
candidate summary for lcsp-continuity-b has unique inbound tag
candidate summary for lcnd-egress-cn has a different inbound tag
candidate summary rejects public tcp/443
candidate summary rejects auto_route and whole-host tun
candidate summary includes only digest and endpoint class in public plan
commands reject missing files with exit code 1
commands reject contract violations with exit code 2
commands create outputs with no overwrite
commands do not call sudo, systemd, ip, nft, iptables, route, resolvectl, tailscale, docker, ssh, curl, or wget
JSON output is redacted and deterministic
```

Acceptance evidence:

```text
CLI can generate and validate the continuity delivery package locally without server access.
sing-box remains an adapter detail; LCnd public contracts do not depend on raw sing-box config values.
Candidate files remain high sensitivity and are never added to samples.
```

### CELL-100B.3 Secret Reference Schema

Files:

```text
D:\LCnd\schemas\network-secret-ref.schema.json
D:\LCnd\internal\networkcore\secretref.go
D:\LCnd\internal\networkcore\secretref_test.go
```

TDD and verification commands:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkcore -run 'TestSecretRef' -count=1
```

Required tests:

```text
accept lcnd-secret://lcsp/continuity-b/server-node
accept lcnd-secret://lcsp/egress-cn/provider-node
reject ss:// material in public input
reject https:// subscription material in public input
reject password, token, private_key, url, subscription fields
reject unknown JSON fields
confirm public summaries expose only ref, consumer, material class, rotation policy, and redacted digest
```

Acceptance evidence:

```text
No schema, Go error, test fixture, stdout, or sample contains a real node secret.
```

### CELL-100B.4 Direct And Egress Probe Contract

Files:

```text
D:\LCnd\schemas\network-probe-plan.schema.json
D:\LCnd\samples\network\continuity-b-probes.synthetic.json
D:\LCnd\internal\networkcore\probeplan.go
D:\LCnd\internal\networkcore\probeplan_test.go
D:\LCnd\internal\networkcli\cli.go
D:\LCnd\internal\networkcli\cli_test.go
```

Add this read-only or file-generation command:

```text
cnd-network continuity probe-plan --input PLAN.json --output PROBES.json --format text|json
```

TDD and verification commands:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkcore ./internal/networkcli -run 'TestProbePlan|TestContinuityProbeCLI' -count=1
```

Required tests:

```text
accept direct_control_probe for protected target baselines
accept egress_continuity_probe for lcsp-continuity-b only
accept egress_application_probe for lcnd-egress-cn only
require at least three samples for comparative latency or throughput claims
reject full URLs with query strings in public probe targets
reject probe output that contains request or response bodies
reject a probe that mixes continuity and application egress domains
```

Acceptance evidence:

```text
The plan contains direct and egress probe definitions but no live network execution.
```

### CELL-100B.5 Protected Target Guard

Files:

```text
D:\LCnd\samples\network\protected-platform.synthetic.json
D:\LCnd\internal\networkcore\protection.go
D:\LCnd\internal\networkcore\protection_test.go
D:\LCnd\internal\networkcli\cli.go
D:\LCnd\internal\networkcli\cli_test.go
```

Add this read-only command:

```text
cnd-network continuity guard-check --input PLAN.json --protected PROTECTED.json --format text|json
```

TDD and verification commands:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkcore ./internal/networkcli -run 'TestProtectedTarget|TestContinuityGuardCLI' -count=1
```

Required tests:

```text
reject listener overlap with public tcp/443
reject route rule that captures SSH 22
reject route rule that captures LCSP Access loopback 18082
reject DNS rule that changes global DNS behavior
reject firewall mutation marker
reject Tailnet preference mutation marker
allow read-only fingerprint checks for protected targets
```

Acceptance evidence:

```text
The old public TCP/443 line is protected by data, schema, validation, and tests.
```

### CELL-100B.6 Rollback Trigger And Publish Gate

Files:

```text
D:\LCnd\internal\networkcore\rollback.go
D:\LCnd\internal\networkcore\rollback_test.go
D:\LCnd\internal\networkcli\cli.go
D:\LCnd\internal\networkcli\cli_test.go
D:\LCnd\docs\SDD.md
D:\LCnd\README.md
D:\LCnd\docs\TEST_HANDBOOK.md
D:\LCnd\CHANGELOG.md
D:\LCnd\VERSION
D:\LCnd\.github\PULL_REQUEST_TEMPLATE.md
```

Add this read-only or file-generation command:

```text
cnd-network continuity rollback-evaluate --receipt RECEIPT.json --output DECISION.json --format text|json
```

Version rule:

```text
The current repository version is 0.3.0.
GO-100B is a larger function, architecture, and product-scope change.
Freeze the implementation target version at 0.4.0.
When the implementation is complete and accepted, synchronize VERSION, CHANGELOG.md, README.md, executable metadata, release notes, and the later v0.4.0 Git tag plan.
This planning CELL does not create or push a tag.
```

TDD and verification commands:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkcore ./internal/networkcli -run 'TestRollback|TestContinuityRollbackCLI' -count=1
rg -n "lcsp-continuity-b|lcnd-egress-cn|production_authorized|LCSP-only|TCP/443|Xray|HAProxy" D:\LCnd\README.md D:\LCnd\docs D:\LCnd\CHANGELOG.md
$bad = @('T'+'BD','T'+'ODO','FIX'+'ME','St'+'ep','st'+'eps')
foreach ($p in $bad) { rg -n $p D:\LCnd\README.md D:\LCnd\docs\RFC\0004-lcnd-continuity-b-contract.md D:\LCnd\docs\TEST_HANDBOOK.md; if ($LASTEXITCODE -eq 0) { exit 1 } }
```

Acceptance evidence:

```text
PROTECTED_TARGET_DRIFT blocks commit.
OLD_LINE_HEALTH_REGRESSION blocks commit.
PUBLIC_443_COLLISION blocks commit.
SECRET_VALUE_EXPOSED blocks commit.
ROLLBACK_NOT_ARMED blocks apply.
lcsp-continuity-b rollback domain cannot detach lcnd-egress-cn.
lcnd-egress-cn rollback domain cannot stop lcsp-continuity-b.
Rollback logic is deterministic and scoped to exactly one workload domain.
Docs state that no production deployment, server connection, LCSP modification, LCapi modification, or LCgpu modification is included in the LCnd implementation.
Clear version, commit, pushed branch, PR URL, and post-merge main commit exist after future implementation.
This planning CELL itself only creates this plan document, commits it, pushes the current branch, and reports back to the Supervisor.
```

Required full GO-100B local verification before publication:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./internal/networkcore ./internal/networkengine/singbox ./internal/networkcli ./cmd/cnd-network -count=1
& $go test ./...
& $go vet ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1 -SkipRace
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-linux.ps1
git diff --check
git status --short
```

Required GitHub route after all six CELL outcomes pass:

```text
Work on a codex-prefixed feature branch.
Commit only exact LCnd files for GO-100B.
Push the feature branch to GitHub repository DWG7318/codex-network-doctor.
Open a PR against DWG7318/codex-network-doctor main.
Require reviewer or Owner acceptance before merge.
Merge through PR into DWG7318/codex-network-doctor main only after all local gates and any GitHub checks pass.
After merge, verify main points at the accepted commit and record the version evidence.
No production deployment is part of this publish route.
```

## Hard Stops

Stop immediately and request Owner or LCSP direction if any future implementation needs one of these actions:

```text
editing D:\LCSP, D:\LCapi, or D:\LCgpu
connecting to a production server
reading or writing real server credentials
installing Docker or any runtime
using sudo or systemd
modifying SSH, HAProxy, Caddy, Xray, Shadowsocks, Tailnet, firewall, DNS, global routes, or system proxy
opening any new public proxy endpoint
touching public TCP/443 or UDP/443
changing the user's current v2rayN or client node config
combining lcsp-continuity-b with lcnd-egress-cn
placing secrets, raw configs, subscription URLs, diagnostic ZIPs, or candidate files in Git
```

## Evidence Required Before Claiming GO-100B Complete

Future implementers must provide:

```text
exact changed file list
focused test output for every CELL
full go test output
go vet output
Windows build output
Linux amd64 cross-build output
placeholder scan output
secret scan output over changed files
git diff --check output
git commit SHA
git push evidence
PR URL
main merge SHA after PR acceptance
statement that no production deployment was performed
statement that old public TCP/443, Xray, HAProxy, and client config were not modified
```
