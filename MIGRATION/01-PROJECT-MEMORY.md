# LCnd project memory

## Identity and purpose

LCnd is **Codex Network Doctor**, a Windows-first offline network evidence, repair, rollback, and layered speed-diagnostic toolkit. It was created for machines whose browser and Codex may already be offline, so user-facing operations must remain usable without an agent or an internet connection.

Public repository: `https://github.com/DWG7318/codex-network-doctor`

Current capability version at migration planning time: `0.2.0`.

## Owner requirements that must persist

- Prefer a double-clickable Windows executable over instructions that require an end user to type PowerShell.
- The affected computer may have no usable web or Codex connection; always provide a local executable path before a web URL during an incident.
- Resolve the Desktop through the Windows system folder API so Windows 10, Windows 11, redirected Desktop, and OneDrive Desktop work correctly.
- Diagnostic ZIP names begin with `cnd诊断结果` and include the host and timestamp.
- Collection is passive by default. Repair is explicit, bounded, backed up, verifiable, and reversible.
- Keep raw evidence private. Only sanitized cases and synthetic samples may enter the public repository.
- Use Chinese for user-facing operational instructions.
- Versioning starts at `0.1.0`; routine upgrades add `0.0.1`, while a substantial capability group may add `0.1.0`.
- LCSP is a separate project and must never be mixed into LCnd Git, releases, migration context, or cleanup decisions.

## Development chronology

### 2026-07-30: project creation and evidence engine

- The project began from `Codex_Network_Doctor_MasterSpec.zip`.
- Existing open-source tools were evaluated rather than blindly reimplemented. osquery became an optional structured provider; Windows PowerShell remained the built-in fallback.
- A Go CLI was built to collect Windows system, adapter, route, DNS, proxy, process, socket, firewall, certificate, and hosts evidence into a validated deterministic ZIP.
- A one-click executable writes the result to the real Windows Desktop with the required Chinese filename prefix.
- Schemas, provider selection, collector catalog, threat model, test handbook, SDD, ADRs, and sample evidence were added.

### 2026-07-30: v2rayN TUN incident and repair executable

- Initial symptoms were that WeChat could connect while browsers and Codex could not.
- The first confirmed failure involved a v2rayN/sing-box controller port entering a Windows excluded port range and failing with WSAEACCES/10013.
- The explicit repair executable evolved to locate the active portable v2rayN instance, back up configuration, ensure TUN is enabled, avoid unknown port owners, relocate a conflicted local port block when necessary, remove known duplicate/legacy managed processes, restart v2rayN, and verify the TUN interface, local listener, and HTTPS reachability.
- A rollback executable restores the most recent repair backup and restarts the preserved v2rayN executable.
- The complete sanitized incident became embedded case `CND-CASE-2026-001`.
- Release `0.1.0` established the public baseline.

### 2026-07-30 to 2026-08-01: recurrence and corrected diagnosis

- A restart recurrence showed that not every failure was an excluded-port event.
- In later fault snapshots, the v2rayN GUI remained alive while sing-box/xray, the local mixed listener, `singbox_tun`, and TUN routes were absent.
- CND restored service without changing the base port or TUN enable flag by performing its controlled elevated restart path.
- `CodexWatchdog.App.exe` was initially suspected, but it remained running after repair while sing-box stayed healthy. This weakens continuous unconditional killing, while leaving one-shot or startup-order behavior unproven.
- The root cause of the post-boot missing core remains unconfirmed. Important hypotheses include v2rayN elevation differences, startup timing, a core launch failure without retry, a one-time process action, transient port ownership, or TUN driver readiness.
- A detailed read-only self-audit prompt was written to `out/DWG-Codex网络自检提示词.md`. It requires auditing the original Codex task's actual tool calls before blaming Watchdog or applying another repair.

### 2026-08-04: layered speed diagnostics

- An external diagnostic script was reviewed and used to isolate system/TUN, HTTP proxy, and SOCKS5 paths without destabilizing v2rayN.
- Testing established that local entry paths could converge on the same remote chain and that large throughput swings could originate after that convergence. Private host addresses and credentials are intentionally absent from this document.
- LCnd gained `cnd网速检测.exe` and `cnd网络修复.exe speed`.
- The implementation performs three serial paired samples, records HTTP status, byte count, TTFB, duration and Mbps, and emits stable bounded diagnostic codes.
- Speed diagnostics remain non-elevated and read-only: no TUN switch, DNS/route/proxy edit, or process restart.
- This capability is release `0.2.0`, commit lineage ending at `02e3d47` before migration documentation.

## Current product surfaces

| Surface | Role | Mutation boundary |
|---|---|---|
| `cnd.exe` | Passive evidence collection | Writes evidence package only |
| `cnd网络修复.exe` | Explicit v2rayN TUN repair | Bounded config/process changes with backup and verification |
| `cnd网络修复-撤销.exe` | Restore latest repair backup | Restores known backup and restarts preserved executable |
| `cnd网速检测.exe` | Layered speed diagnosis | Network requests only; no configuration or process mutation |
| Embedded casebook | Offline incident lookup | Read-only |

## Current unresolved work

1. Obtain direct evidence for the recurrent “GUI alive, core/TUN missing after boot” incident before adding automatic self-healing.
2. If self-healing is later approved, require multiple simultaneous failure signals, a cooldown, one bounded repair attempt, logging, and no unknown-process termination.
3. Continue the MasterSpec backlog without weakening the passive-default boundary or evidence schema guarantees.
4. Treat external raw diagnostic archives as private source evidence; extract only sanitized reusable cases.

## What not to infer

- A successful CND restart does not prove that “restart” alone was the cause; CND also changes the privilege and process-lifecycle context.
- Historical excluded-port evidence does not prove every recurrence is another excluded-port failure.
- Watchdog coexisting after repair does not prove it never performed a startup-only action.
- A fast or slow local proxy protocol sample does not by itself locate a bottleneck when all paths later share the same remote chain.

## Historical sources in the private migration context

The private transfer package contains sanitized LCnd Codex session JSONL exports, relevant session-index records, the original MasterSpec archive, user-supplied fault snapshots, selected screenshots/logs, and a hash/provenance index. Credential values, opaque encrypted reasoning, LCSP-related records, and the current migration turn are deliberately excluded. These materials are private evidence and must never be pushed to the public repository.
