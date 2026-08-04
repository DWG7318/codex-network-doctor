# RFC 0002: Explicit Active Speed Path Diagnostics

Status: Accepted
Target: `cnd网速检测.exe` and `cnd网络修复.exe speed`

## Context

The passive `cnd.exe` evidence collector intentionally does not create network traffic or
infer a root cause. Operators also need an explicit, bounded way to distinguish local
system/TUN ingress, HTTP proxy ingress, SOCKS5 proxy ingress, and the remote path shared
after those ingress paths converge.

A single speed value cannot locate a bottleneck. Samples must be paired in time and must
keep the selected node, TUN state, and local proxy endpoint unchanged during the run.

## Decision

Add active speed diagnostics to the separately delivered repair/diagnostic executable.
The feature is never called by `cnd.exe` and is active only when the operator runs
`cnd网速检测.exe` or supplies the `speed` operation explicitly.

Each of three attempts executes sequentially:

1. `system_path`: no application proxy; TUN captures it when TUN is enabled.
2. `http_proxy`: HTTP CONNECT through the selected local v2rayN/sing-box listener.
3. `socks5_proxy`: SOCKS5 remote-name resolution through the same listener.

The default target is a 1 MB Cloudflare download. Each request has a 25-second timeout,
responses have a 2 MiB safety cap, keep-alive is disabled, and tests are never concurrent.
The tool discovers a local v2rayN-related listener but does not read node credentials.

## Finding contract

Comparative findings require at least three successful samples on both paths:

- `TUN_PATH_SLOW`: system/TUN median is less than half the SOCKS5 median.
- `TUN_PATH_UNRELIABLE`: system/TUN fails at least two of three attempts while SOCKS5
  succeeds three times.
- `HTTP_PROXY_PATH_SLOW`: HTTP proxy median is less than half the SOCKS5 median.
- `HTTP_PROXY_PATH_UNRELIABLE`: HTTP proxy fails at least two of three attempts while
  SOCKS5 succeeds three times.
- `SHARED_REMOTE_PATH_UNSTABLE`: both system/TUN and SOCKS5 vary by at least fourfold;
  the suspected scope is after the local ingress paths converge.
- `INSUFFICIENT_EVIDENCE`: fewer than three successful paired samples.
- `NO_PATH_SPECIFIC_ISSUE`: adequate samples do not cross a finding threshold.

These findings delimit a component scope; they do not claim a particular third-party
server is faulty without server-side evidence. Remote SSH diagnosis and credential
handling are deliberately outside this RFC.

## Safety and privacy

- No elevation is requested.
- No process is stopped or restarted.
- TUN, DNS, routes, system proxy, and v2rayN configuration are not changed.
- Tests are sequential and bounded to avoid proxy-core load spikes.
- Output contains local endpoint/process metadata and network performance values but no
  passwords, node configuration, SSH keys, or process command lines.

## Compatibility

The Evidence ZIP schema and passive collector catalog do not change. Existing repair and
rollback modes retain their behavior. The new double-click executable is a copy of the
repair binary whose filename selects the read-only speed mode.
