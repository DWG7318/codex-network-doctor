# LCnd asset and provenance ledger

## Public, version-controlled assets

- Product source under `cmd/` and `internal/`.
- Schemas and collector catalog under `schemas/` and `docs/`.
- Tests next to implementation packages.
- Sanitized embedded casebook under `internal/casebook/cases/`.
- Synthetic example evidence under `samples/`.
- Build and bootstrap scripts under `scripts/`.
- Public architecture, threat, test, ADR, RFC, release, and migration documentation.

## Offline package-only assets

- `.tools` portable SDKs, optional providers, source references, and download archives.
- `bin`, `dist`, and historical `release` executables.
- Full Git metadata in `workspace/.git`.
- Independent `repository/codex-network-doctor.bundle`.
- Sanitized Codex JSONL exports for the LCnd main and speed-diagnostics tasks.
- Relevant `session_index.jsonl` records.
- Original MasterSpec archive and supplied incident evidence.
- Package hashes, GitHub receipt, build receipt, receiving verification, and source cleanup receipt.

## Private evidence rules

- Private evidence is indexed by filename, origin, date, size, and SHA-256.
- It may contain hostnames, local usernames, process paths, network addresses, or diagnostic configuration.
- It must remain in `private-context` and must not be staged, committed, attached to a public issue, or uploaded without a separate redaction review.
- SSH private keys, GitHub tokens, Codex authentication files, subscription credentials, and unrelated global Codex state are not included at all.

## Source-of-truth order

1. Verified Git commit and tag.
2. Tracked product documentation and tests.
3. Package manifest and build receipts.
4. Sanitized casebook.
5. Private raw sessions and incident evidence for historical reconstruction.

Historical chat statements are not automatically product truth. When a historical answer conflicts with later evidence or code/tests, the later verified artifact wins and the contradiction should be documented.
