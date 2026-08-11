# LCnd repository instructions

## Scope

- This repository is **Codex Network Doctor (LCnd/CND)** only.
- `LCSP`, `LCSP-staged-*`, `lcsp-*.tar.gz`, cloud credentials, SSH keys, and LCSP artifacts are out of scope and must never be staged, copied into a release, or included in an LCnd migration package.
- Preserve unrelated user files in a mixed working directory.

## Product contract

- `cnd.exe` is passive-by-default network evidence collection and writes `cnd诊断结果-<HOST>-<TIMESTAMP>.zip` to the Windows Desktop resolved through the system folder API.
- `cnd网络修复.exe` is an explicit, elevated, bounded v2rayN TUN repair action with backup, verification, and rollback support.
- `cnd网络修复-撤销.exe` restores the latest repair backup.
- `cnd网速检测.exe` performs explicit, read-only layered speed diagnostics and must not restart or reconfigure v2rayN.
- Never turn passive collection or speed testing into implicit repair.
- Do not terminate unknown port owners, reset Winsock/TCP-IP, change nodes, or publish raw diagnostic evidence.

## Toolchain and commands

Use the repository-local Go toolchain first:

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go test ./...
& $go vet ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1 -SkipRace
```

For release-grade validation, also run race tests when MinGW GCC is available:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1
```

The migration package includes `.tools\go`; a global Go installation is not required. `-SkipRace` is allowed only for an offline/non-release receiving-host verification.

## Versioning

- Versioning began at `0.1.0`.
- Normal incremental upgrades add `0.0.1`.
- Material new capability groups may add `0.1.0`.
- Keep `VERSION`, `CHANGELOG.md`, user documentation, executable metadata, tags, and release names consistent.

## Testing and change discipline

- Use test-first development for new behavior and bug fixes.
- Run focused tests, then `go test ./...`, `go vet ./...`, and the Windows build.
- Keep Windows PowerShell 5.1 compatibility for user-facing `.ps1` files and use UTF-8 BOM when Chinese text is present.
- Preserve deterministic evidence packaging and schema compatibility unless a documented versioned change requires otherwise.

## Privacy and Git

- The GitHub repository is public. Only commit sanitized cases and synthetic/sample evidence.
- Never commit node credentials, subscription URLs, tokens, cookies, SSH material, raw public-server configurations, or unredacted CND result ZIP files.
- Inspect exact staged paths; never stage the whole mixed working tree.
- Durable project decisions live in tracked documentation. Raw Codex sessions and private incident evidence belong only in the offline private migration context.

## Receiving a migrated project

Read these files before changing code:

1. `MIGRATION/00-RECEIVER-START-HERE.md`
2. `MIGRATION/01-PROJECT-MEMORY.md`
3. `MIGRATION/02-DECISION-LEDGER.md`
4. `MIGRATION/03-ENVIRONMENT-INVENTORY.md`
5. `docs/MASTER_SPEC.md`
6. `docs/SDD.md`
7. `docs/TEST_HANDBOOK.md`

Do not claim successful handoff until the migration manifest, Git state, tests, vet, and Windows build have been verified on the receiving computer.
