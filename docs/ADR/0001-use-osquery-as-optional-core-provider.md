# ADR 0001: Use osquery as the preferred optional provider

状态：Accepted
日期：2026-07-30

## Context

项目需要覆盖 Windows 网络接口、路由、邻居、套接字、进程、服务、证书与防火墙。直接为每项维护 Win32/IP Helper/WMI 实现会显著增加版本兼容成本。

## Decision

优先适配 osquery 5.23.1 的一次性 `osqueryi` 查询，同时保留只使用系统 PowerShell 的降级 Provider。

osquery 使用 Apache-2.0 许可路径。仓库不提交其二进制；固定版本下载必须验证发布资产 SHA-256。

## Consequences

正面：

- 利用成熟 Windows 表实现。
- SQL 目录可审计、可测试。
- JSON 原始输出稳定。
- 未来可以替换或增加 Provider，而不改变 Evidence ZIP。

负面：

- 额外约 19 MB 下载。
- Windows 表与其他平台存在差异，例如 DNS 字段位于 `interface_details`。
- osquery 自身仍属于信任链的一部分。

## Rejected

- 把 Velociraptor 作为内嵌核心：功能过宽且 AGPL-3.0 对默认分发模型影响较大。
- 仅用 PowerShell：在语言版本、模块装载和输出结构上更容易漂移。
- 直接绑定全部 Windows API：适合 v4.1 的高价值 Collector，而非 v4 首发。
