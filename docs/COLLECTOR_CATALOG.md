# Collector Catalog

所有默认 Collector 均为 `passive`。`complete` 表示当前所选 Provider 的目录全部执行成功。

## osquery 5.23.1

| ID | 分类 | 数据来源 | 默认超时 |
|---|---|---|---:|
| `system_info` | System | `system_info` | 15s |
| `os_version` | System | `os_version` | 15s |
| `interface_addresses` | Adapter | `interface_addresses` | 15s |
| `interface_details` | Adapter | `interface_details` | 15s |
| `routes` | Route | `routes` | 15s |
| `arp_cache` | Route | `arp_cache` | 15s |
| `dns_configuration` | DNS | Windows 字段所在的 `interface_details` | 15s |
| `process_open_sockets` | Socket | `process_open_sockets` | 15s |
| `listening_ports` | Socket | `listening_ports` | 15s |
| `processes` | Process | `processes`，不读取命令行/环境 | 15s |
| `services` | Process | `services` | 15s |
| `certificates` | TLS | `certificates` 公共元数据 | 30s |
| `windows_firewall_rules` | Firewall | `windows_firewall_rules` | 30s |

## Windows PowerShell fallback

| ID | 分类 | 只读数据源 | 默认超时 |
|---|---|---|---:|
| `system_info` | System | CIM OS/ComputerSystem、TimeZone | 15s |
| `net_adapters` | Adapter | `Get-NetAdapter` | 15s |
| `ip_configuration` | Adapter | `Get-NetIPConfiguration` | 15s |
| `routes` | Route | `Get-NetRoute` | 15s |
| `neighbors` | Route | `Get-NetNeighbor` | 15s |
| `dns_servers` | DNS | `Get-DnsClientServerAddress` | 15s |
| `dns_cache` | DNS | `Get-DnsClientCache` | 15s |
| `hosts_file` | DNS | Windows hosts 文件 | 15s |
| `tcp_connections` | Socket | `Get-NetTCPConnection` | 15s |
| `udp_endpoints` | Socket | `Get-NetUDPEndpoint` | 15s |
| `winhttp_proxy` | Proxy | `netsh winhttp show proxy` | 15s |
| `user_proxy` | Proxy/PAC | HKCU Internet Settings | 15s |
| `firewall_profiles` | Firewall | `Get-NetFirewallProfile` | 30s |
| `processes` | Process | `Get-Process`，不读取命令行/环境 | 15s |
| `certificates` | TLS | `.NET X509Store`，`ReadOnly` | 30s |

## Excluded from default baseline

| 分类 | 原因 |
|---|---|
| TLS handshake | 会主动连接远端 |
| WebSocket | 会主动连接并可能触发应用行为 |
| PAC fetch | 下载 PAC 会产生网络流量；默认只记录 URL |
| WFP state capture | 某些命令会启动诊断会话或生成系统文件 |
| Packet capture | 需要驱动/ETW 会话并改变主机状态 |
| v2ray/Cockpit/Codex 深度配置 | 可能包含令牌、会话和用户内容；需独立隐私 RFC |
| EventLog timeline | v4 暂未实现来源时间语义与流式大文件 Artifact |

机器可读的实际目录始终以 `cnd catalog` 和 Evidence ZIP 中的 `collectors/catalog.json` 为准。
