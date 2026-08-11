# LCnd development environment inventory

## Included offline components

| Component | Location | Purpose |
|---|---|---|
| Go SDK 1.26.5 for Windows amd64 | `.tools\go` | Tests, vet, builds; no global Go required |
| Go module vendor tree | `vendor` | Offline dependency resolution |
| osquery 5.23.1 Windows x86_64 | `.tools\osquery-5.23.1.windows_x86_64` | Optional structured evidence provider |
| golangci-lint 2.12.2 | `.tools\golangci-lint-2.12.2-windows-amd64` | Optional static analysis |
| v2rayN 7.19.5 source snapshot | `.tools\v2rayN-source` | Reference for process, port, and TUN behavior |
| Download archives | `.tools\downloads` | Offline reconstruction fallback |
| Built executables | `bin`, `dist`, `release` | Immediate use and historical releases |

The receiving computer does not need to reinstall Go, osquery, or project Go modules for normal verification and development.

## Required base software

- Windows 10 or Windows 11, x64.
- Windows PowerShell 5.1 or newer.
- Codex/ChatGPT desktop app for agent takeover.
- Git is recommended for development; Codex normally supplies a bundled Git even when it is not on the user's global PATH.

## Optional software

- MinGW-w64 GCC is needed only for Windows `go test -race` through the release-grade build path.
- GitHub CLI and authentication are needed only when publishing future changes.
- Neither is required for receiving, reading history, running ordinary tests/vet, or building non-race Windows executables.

## Source-host snapshot

- Git source-host version at migration planning: 2.51.0.windows.1.
- GitHub CLI source-host version: 2.96.0.
- Repository module directive: Go 1.26.0.
- Direct external Go dependency: `golang.org/x/sys v0.47.0`, included in `vendor`.
- Main public repository is intentionally credential-free.

## Standard verification

```powershell
$go = Join-Path $PWD '.tools\go\bin\go.exe'
& $go version
& $go test ./...
& $go vet ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1 -SkipRace
```

Release-grade validation adds race testing when GCC is available:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1
```

## Codex setup on the receiving host

- Add the restored LCnd directory as a local project and make it the primary folder.
- The repository root `AGENTS.md` supplies durable project rules automatically to new tasks.
- Do not copy the source host's global `.codex\config.toml`, SQLite databases, browser state, plugin tokens, or `auth.json`.
- Raw LCnd sessions are evidence files in the package, not a supported drop-in replacement for the receiving host's app database.
- If local memories are enabled, allow the receiving Codex to generate new memories after it reads the tracked handoff documents and verifies the project.
