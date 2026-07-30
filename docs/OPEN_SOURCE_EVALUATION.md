# Open-source Evaluation

评估日期：2026-07-30。Stars 与活跃时间是该日期的快照，仅用于成熟度参考，不构成安全背书。

| 项目 | 规模快照 | 许可证 | 优点 | 结论 |
|---|---:|---|---|---|
| [osquery/osquery](https://github.com/osquery/osquery) | 23.4k stars，2026-07-29 有更新 | Apache-2.0 OR GPL-2.0-only | Windows 表覆盖网卡、路由、ARP、套接字、进程、服务、证书和防火墙；一次性 JSON 输出 | **采用**，按 Apache-2.0 路径使用固定 5.23.1 |
| [Velocidex/velociraptor](https://github.com/Velocidex/velociraptor) | 4.1k stars，2026-07-28 有更新 | AGPL-3.0 | 完整 DFIR 平台、VQL、远程编排、Artifacts | 不嵌入 v4；功能过宽且 AGPL 分发义务不适合默认核心 |
| [wireshark/wireshark](https://github.com/wireshark/wireshark) | 9.7k stars，2026-07-29 有更新 | GPL-2.0 | 最成熟的包捕获与协议解析 | 不进入 passive 基线；抓包会改变现场且需要额外驱动/权限 |
| [Yamato-Security/hayabusa](https://github.com/Yamato-Security/hayabusa) | 3.3k stars，2026-07-29 有更新 | AGPL-3.0 | Windows Event Log 时间线和 Sigma 分析成熟 | 作为外部分析工具候选，不嵌入采集核心 |
| [microsoft/etl2pcapng](https://github.com/microsoft/etl2pcapng) | 734 stars，2025-06-05 有更新 | MIT | 可把 Windows 网络 ETL 转为 pcapng | 未来主动捕获配置的可选转换器 |
| [shirou/gopsutil](https://github.com/shirou/gopsutil) | 11.9k stars，2026-07-26 有更新 | BSD | Go 原生跨平台系统枚举 | v4 未引入；标准库加外部 Provider 已满足目标，减少依赖 |
| [EricZimmerman/KapeFiles](https://github.com/EricZimmerman/KapeFiles) | 861 stars，2026-06-17 有更新 | MIT | 大量社区取证目标与模块定义 | 适合后续 Artifact 目录设计参考，不直接复制 KAPE 运行时 |

## Why osquery

osquery 把操作系统状态暴露为 SQL 表，适合把“采集什么”固定成可审计目录。CND 不启动 `osqueryd` 服务，只运行短生命周期 `osqueryi`：

```text
--json
--disable_events=true
--disable_extensions=true
--disable_logging=true
--logger_stderr=false
--utc=true
SELECT ...
```

这保留了 osquery 的 Windows API 覆盖，同时避免长期数据库、计划任务、扩展和后台日志。

## Version and integrity

- 固定版本：`5.23.1`
- Windows x86_64 ZIP：`osquery-5.23.1.windows_x86_64.zip`
- GitHub Release asset SHA-256：`7bd411050ef6b5aae1b23956aec0dc5ce6e800c5656f0cd463ac70a6e1bdf30b`
- 发布地址：[osquery 5.23.1](https://github.com/osquery/osquery/releases/tag/5.23.1)

`scripts/bootstrap-osquery.ps1` 下载 ZIP 后必须先通过该哈希，才允许解压。

## Source-use boundary

CND 没有复制候选项目源代码。仓库默认不分发 osquery 二进制，只提供适配器和经过哈希固定的下载脚本。Evidence ZIP 也不得包含 Provider 二进制。
