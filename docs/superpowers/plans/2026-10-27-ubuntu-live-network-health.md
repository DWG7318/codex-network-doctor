# Ubuntu 24.04 live network health — v0.5.0

Owner approved on 2026-10-27: align with latest main, then implement LCnd-owned real read-only detection and health reporting. LCSP integration remains deferred.

## Scope and acceptance

- Add explicit `cnd-network diagnose --input targets.json --output health.json --text-output health.txt`.
- Linux host observations: OS, interfaces/addresses, resolver and IPv4/IPv6 route snapshots, using bounded file/API reads without shell execution.
- Explicit HTTPS HEAD targets only, serial bounded samples, DNS/TCP/TLS/HTTP evidence. Optional explicit existing HTTP/SOCKS5 proxy comparison; never inherit environment proxy, follow redirects, send credentials, or read response bodies.
- Separate real report schema/evidence mode from existing synthetic continuity contracts. Scores describe selected target/path checks, never all networks or guaranteed causes. Missing/cancelled evidence must not become healthy.
- Output files created exclusively, mode 0600 on Linux; private host evidence never committed. Console/errors only carry stable codes/counts. Existing VPN, routes, DNS, firewall and other services remain unchanged.
- No LCSP/LCapi code, service installation, engine deployment, global tuning, production operation, or automatic repair.

## Work ledger

1. Completed: fetch main; preserve divergent local main as `codex/archive-local-main-20261027`; recreate local main at `6fe3a0c`; preserve prior branch and untracked private data.
2. Completed: test-first `internal/networkhealth` collector/prober/evaluator and separate CLI dispatch. Local HTTP/TLS/proxy fixtures and Linux host fixture files cover the core behavior.
3. Completed: v0.5.0 contracts, sample input, Chinese operator instructions and design/test documentation.
4. Completed local verification and independent review. Delivery uses `codex/ubuntu-live-network-health` and a scoped PR; main merge requires passing PR CI. The Git/CI receipt is retained in GitHub. No Release/tag or deployment is part of this task.

## Verification

Use `.tools/go/bin/go.exe`. Network tests use ephemeral localhost listeners only. Linux validation uses the existing local Ubuntu WSL environment; never connect to a production server. Check Linux release identity before claiming Ubuntu 24.04 coverage. Missing Windows GCC must be reported; do not install dependencies. CI race coverage or available Linux toolchain can supply race evidence.

Real target input and reports belong in private storage outside Git (ignored `out/` only where Linux private modes are enforced). Public examples use reserved documentation domains only. Existing synthetic CLI and Windows surfaces must continue passing unchanged. Completion requires evidence recorded here, not merely this plan.

## Verification evidence — 2026-10-27

- Windows repository-local Go: `go test ./...` and `go vet ./...` pass.
- Existing Ubuntu 24.04.4 amd64 WSL: `GOPROXY=off GOTOOLCHAIN=go1.26.0 go test -race ./...` and `go vet ./...` pass. Host snapshot reports supported platform and complete observations; no raw addresses or routes are included here.
- `scripts/build.ps1 -SkipRace -OutputDirectory out/ubuntu-live-health-v0.5.0/windows` and `scripts/build-linux.ps1 -OutputDirectory out/ubuntu-live-health-v0.5.0/linux-amd64` pass; both network executables identify as 0.5.0. Local Windows MinGW is absent, so no local Windows race/release-grade claim; Linux race evidence is separate.
- Explicit three-attempt HTTPS HEAD smoke from Ubuntu to the public example.com endpoint: live evidence, 3/3 completed, all applicable layers passed, score100, host complete. This validates this local target/path only, not any production server or full-network health.
- Generated JSON validates against `schemas/network-live-health-report.schema.json`; public sample validates against `schemas/network-live-targets.schema.json` using PowerShell `Test-Json`.
- Native Linux outputs verified mode0600. WSL Windows-mount output observed to ignore requested mode, then regression protection added: the CLI now rejects that filesystem before observation and removes only its new empty files. Final real reports retained only in private Linux verification storage, never staged.
- Full optional golangci-lint initially found two new fixture close checks and nine pre-existing errcheck findings. The two new findings were fixed; new-diff lint reports zero issues. Baseline lint debt remains out of scope; no claim of a globally clean lint baseline.
- No server SSH, production deployment, service operations, node changes, VPN/TUN changes or host networking writes were performed.
- Independent read-only review (inherited model identity not exposed, requested reasoning high) found two important issues: detached HTTP transport context misallocated multi-address fallback time; HTTP status600 could violate the report schema. Both reproduced as failing tests, then fixed and focused tests passed. Added stalled HTTP/SOCKS handshakes and active cancellation regressions. Final independent re-review passed, including three repetitions of all four regressions and uncached health/CLI package tests; no blocking findings remain. Parent reran full Windows tests/vet/build, Linux race/vet/build, real HEAD smoke and report schema validation after these fixes.

## Remaining scope outside this delivery

LCSP integration, node secret injection, production/server acceptance, LCapi consumption, continuous monitoring, throughput testing, per-hop root-cause analysis and automatic repair remain separate work. WSL localhost/proxy fixtures do not count as production acceptance.
