# LCnd decision ledger

| ID | Decision | Reason | Durable consequence |
|---|---|---|---|
| D-001 | Ship a native Windows executable | Faulted machines may not have Codex or usable web access | Core operator flows must work offline and by double click |
| D-002 | Resolve Desktop through the Windows folder API | Desktop location differs across Windows versions, redirection, and OneDrive | Never hard-code `C:\Users\<name>\Desktop` |
| D-003 | Prefix result archives with `cnd诊断结果` | The user must recognize evidence immediately | Preserve filename contract |
| D-004 | Keep evidence collection passive by default | Diagnostics must not destroy the state being diagnosed | No implicit repair in `cnd.exe` |
| D-005 | Use osquery as optional provider with a built-in PowerShell fallback | Structured collection is useful, but offline portability is mandatory | The executable remains usable without installing osquery |
| D-006 | Package deterministic, schema-validated evidence | Evidence must be reproducible and auditable | Preserve manifest, timeline, validation, and deterministic ZIP behavior |
| D-007 | Separate repair from collection | Repair is stateful and elevated | `cnd网络修复.exe` is explicit and independently tested |
| D-008 | Backup before repair and ship rollback | v2rayN config/process changes need recovery | Every repair path must preserve rollback evidence |
| D-009 | Never terminate an unknown port owner | Port conflicts can belong to unrelated software | Fail safely or relocate known ports instead of killing unknown processes |
| D-010 | Handle Windows excluded port ranges | v2rayN controller ports can fail with WSAEACCES/10013 | Inspect the required local port block and migrate only when supported by evidence |
| D-011 | Preserve an offline sanitized casebook | The same incident should be diagnosable without internet access | Add reusable cases only after redaction and validation |
| D-012 | Use SemVer starting at `0.1.0` | Owner corrected the increment policy | Routine work `+0.0.1`; substantial capability `+0.1.0` |
| D-013 | Provide local paths first during a network incident | A GitHub link is useless when the affected machine is offline | Handoffs and support messages name the local executable first |
| D-014 | Do not equate recovery with root cause | A controlled restart changes several variables | Investigation must distinguish port, privilege, timing, crash, and external termination |
| D-015 | Keep speed diagnostics read-only and serial | Concurrent probes or tuning could destabilize v2rayN and confound evidence | No automatic tuning, TUN switching, or process restart in speed mode |
| D-016 | Keep LCSP completely separate | LCSP has its own repository, credentials, history, and lifecycle | Exclude all LCSP paths and archives from Git, the transfer package, and cleanup |
| D-017 | Put mandatory takeover knowledge in Git | Codex memories are generated and auxiliary | `AGENTS.md` and `MIGRATION/*.md` are the receiving agent's source of truth |
| D-018 | Export sanitized task history privately, not publicly | Full context aids takeover, but raw tasks can contain credentials and unrelated-project state | Redacted LCnd-only JSONL exports live in `private-context`; source-host originals remain untouched in Codex storage |
| D-019 | Do not migrate authentication state | Tokens and SSH keys are machine/account secrets and often nonportable | Receiving host signs in independently only when required |
| D-020 | Require two verified backups before source cleanup | Cleanup is destructive even when authorized | GitHub commit and the Desktop transfer-package manifest must both verify before deleting local source/tooling |
