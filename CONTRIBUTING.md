# Contributing

## Principles

- Do not add diagnosis or repair behavior to the collector process.
- New default Collectors must be passive and read-only.
- Preserve raw output before adding normalization.
- Never silently convert a failed Collector to success.
- Treat Evidence ZIP compatibility as a public contract.

## Workflow

1. Create an Issue from `docs/BACKLOG.md` or an Issue template.
2. Add or update an RFC/ADR when behavior or compatibility changes.
3. Write a failing test first.
4. Implement the smallest change.
5. Run the quality gate in `docs/TEST_HANDBOOK.md`.
6. Include Evidence/Privacy impact in the PR.

## Collector checklist

- Stable ID and category.
- Exact query/script committed in the machine-readable catalog.
- Provider, privilege, passive/active mode, timeout and media type.
- Raw stdout and stderr behavior.
- Test for success, failure and relevant platform drift.
- Documentation entry and acceptance condition.

## Commits

Use Conventional Commits such as:

```text
feat(collector): add native adapter snapshot
fix(osquery): use Windows DNS interface fields
docs(rfc): define detached signature
test(evidence): reject duplicate ZIP paths
```
