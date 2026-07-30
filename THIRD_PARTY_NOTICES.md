# Third-party notices

## osquery

Project: [osquery/osquery](https://github.com/osquery/osquery)
Pinned integration version: 5.23.1
License selected for CND integration: Apache License 2.0
Upstream also offers GPL-2.0-only; see the upstream `LICENSE` for its dual-license statement.

CND does not copy osquery source code or commit its binary. The optional bootstrap script downloads the official Windows release archive and verifies SHA-256 before extraction. Users who redistribute osquery must preserve the upstream license and notices.

## Go

The local development environment may use an official portable Go toolchain. The toolchain and download archives are excluded from the repository and are not part of CND source distribution.

## golang.org/x/sys

Project: [golang/sys](https://github.com/golang/sys)
Version: 0.47.0
License: BSD 3-Clause

The repair executable uses `golang.org/x/sys/windows` for typed Windows process and registry interoperability. Its license is compatible with this Apache-2.0 project and must be preserved in source redistributions.

## Evaluated but not incorporated

Velociraptor, Wireshark, Hayabusa, etl2pcapng, gopsutil, KapeFiles and v2rayN were evaluated. The repair behavior was cross-checked against v2rayN's system-proxy implementation, but no v2rayN source or binary is incorporated into CND.
