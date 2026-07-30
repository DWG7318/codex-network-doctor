# RFC 0001: Evidence Package v1

状态：Accepted
日期：2026-07-30

## Problem

Windows 网络诊断通常把命令输出复制到文本文件，缺少统一状态、失败证据、文件哈希和可重复封装。分析者无法快速判断缺失文件是“没有数据”还是“采集失败”。

## Decision

CND 使用 ZIP 作为传输容器，并用 `manifest.json` 作为包内索引。除 Manifest 自身外，所有普通文件必须记录：

- 相对 POSIX 路径。
- 字节大小。
- 小写十六进制 SHA-256。
- 媒体类型。

Collector 结果记录稳定 ID、分类、Provider、开始/结束时间、耗时、状态、错误和生成 Artifact。

## Determinism

相同暂存树必须产生相同 ZIP：

- 条目按 UTF-8 名称排序。
- 时间固定为 1980-01-01 UTC。
- 目录权限固定为 0755，文件权限固定为 0644。
- 目录条目显式写入。
- Manifest 中 Collector 与文件按稳定键排序。

实时采集内容、采集 ID 和采集时间会变化，因此不同采集运行不要求 ZIP 相同。

## Completeness

- `complete`：所选目录全部成功。
- `partial`：至少一个 Collector 失败或超时。

两者都可以是结构完整的有效 ZIP。Validator 不把 `partial` 解释成根因。

## Security rules

- ZIP 名称必须使用 `/`，不得以 `/` 开头、包含 `\` 或经清理后逃逸到 `..`。
- 重复条目无效。
- Manifest 之外的未登记普通文件无效。
- Validator 对单文件设置 256 MiB 上限。
- Evidence Store 不允许重复写入路径。

## Alternatives

- **目录树**：易于查看，但传输时容易丢文件和元数据。
- **tar**：Windows 原生用户体验较差。
- **自定义二进制容器**：增加解析器和长期兼容成本。
- **把 Manifest 哈希写入自身**：产生递归定义，拒绝采用。ZIP 的外部 SHA-256 由命令摘要返回。

## Compatibility

Schema version 为 `1.0.0`。Validator 对未知 Schema version 报 `SCHEMA_VERSION_UNSUPPORTED`。破坏性变更必须提升主版本。
