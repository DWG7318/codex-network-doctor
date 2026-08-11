# Source-host cleanup policy

The source host may be slimmed only after both backups pass:

1. GitHub `main` resolves to the recorded migration commit and the public repository remains readable.
2. The Desktop transfer package's `manifest.sha256` passes, the Git bundle verifies, and an isolated restore from its workspace passes tests, vet, and build.

## Preserve on the source host

Keep a compact archive outside the source working tree containing:

- `01-PROJECT-MEMORY.md` and `02-DECISION-LEDGER.md`;
- sanitized LCnd session exports, relevant session-index records, and hashes/pointers to the original Codex session files;
- the original MasterSpec archive and original incident evidence or immutable pointers to their existing locations;
- package manifest and SHA-256 values;
- GitHub repository URL, branch/tag list, exact commit IDs, and push verification;
- system-resolved Desktop path and transfer-package path;
- build, restore, and cleanup receipts.

The source host's original Codex session files are global app history, not project source: do not copy, edit, or delete them during slimming. Do not keep a second source checkout, `.git`, portable SDK, build cache, binaries, or release archive inside `C:\LCnd` after successful acceptance; those belong to GitHub and the verified Desktop transfer package.

## Never touch as part of LCnd cleanup

- `C:\LCnd\LCSP`
- `C:\LCnd\LCSP-staged-*`
- `C:\LCnd\lcsp-*.tar.gz`
- LCSP Git metadata, credentials, SSH assets, sessions, or receipts
- unrelated user projects or global Codex state

## Cleanup targets after acceptance

Resolve every target to an absolute path and verify it is an LCnd-owned path before deletion. Expected large targets include:

- `.tools`;
- `release`, `bin`, `dist`;
- LCnd source directories (`cmd`, `internal`, `docs`, `schemas`, `scripts`, `samples`, `.github`);
- LCnd root source files and `.git`;
- temporary LCnd diagnostic extraction directories after required evidence is retained.

Record pre/post byte counts and exact removed paths. Permanent deletion is authorized only after the two-backup gate; no broad recursive deletion may target `C:\LCnd` itself because excluded LCSP material is nested there.
