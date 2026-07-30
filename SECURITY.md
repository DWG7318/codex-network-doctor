# Security Policy

## Reporting

Do not open a public Issue for a vulnerability that could expose host evidence, escape the Evidence Store, bypass hash validation, execute an unintended command, or silently modify a target system.

Until a dedicated security contact is configured on the future GitHub repository, report privately to the repository owner through GitHub's private vulnerability reporting feature.

Include:

- Affected commit/version.
- Windows version and Provider.
- Minimal reproduction.
- Whether arbitrary file write, command execution, evidence forgery or secret exposure is possible.
- Suggested disclosure timeline.

## Supported versions

Only the latest tagged minor release receives security fixes during the pre-1.0 phase.

## Sensitive test data

Never attach a real Evidence ZIP to a public report. Create a synthetic minimal package or redact the material outside the original evidence chain.
