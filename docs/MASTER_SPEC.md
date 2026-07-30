# Codex Network Doctor MasterSpec

版本：1.1
状态：唯一权威设计文档
适用交付：v4 Evidence Engine CLI

## 1. 项目使命

构建一个企业级 Windows 网络证据采集平台。工具负责采集、封装和验证证据；人或外部分析系统负责解释证据。

CND 不执行：

- 网络、代理、防火墙、DNS、路由或服务修复。
- 主动 TLS、HTTP、WebSocket、DNS 或端口探测，除非未来由操作者显式选择主动配置。
- Diagnosis、Root Cause、Probability、Recommendation 或 AI Guess。

## 2. 规范性原则

1. **Evidence over inference**：工具输出观察值和采集错误，不输出因果结论。
2. **Read-only by default**：默认目录只能调用读取 API、只读查询和状态显示命令。
3. **Deterministic packaging**：相同暂存文件必须产生相同 ZIP 字节。实时主机在不同时间的观察值不要求相同。
4. **Collector isolation**：每个采集器有独立上下文、超时、stdout、stderr、状态和错误码。
5. **Raw evidence first**：保存供应方原始 JSON 或文本，不以归一化结果替换原始输出。
6. **Failure isolation**：单个采集器失败后继续执行后续采集器；最终状态必须为 `partial`，不得静默标为完整。
7. **Explicit completeness**：`complete` 仅表示本次所选目录全部成功，不表示主机不存在未采集证据。
8. **Chain of custody**：除 `manifest.json` 自身外，每个普通文件必须记录大小、媒体类型和 SHA-256。
9. **No silent overwrite**：证据文件不可重写，输出 ZIP 默认不可覆盖。
10. **Sensitive by construction**：所有 Evidence ZIP 均按敏感材料处理。

## 3. 信任和边界

- CND 可以证明 ZIP 内文件与 manifest 一致，不能证明受调查主机或操作系统 API 未被攻击者篡改。
- `started_at` 和 `finished_at` 来自主机时钟；不自动认定其准确。
- `timeline.json` 只承载证据中具有来源时间戳的事件。当前快照目录不创造伪事件，因此为空。
- `validation` 只做结构、完整性和 Schema 检查，不做诊断。

## 4. Provider

### 4.1 osquery

- 固定测试版本：5.23.1。
- 以一次性 `osqueryi` 进程执行 `SELECT`。
- 禁用事件、扩展和日志插件；使用内存数据库默认行为。
- 选择 Apache-2.0 许可使用路径。

### 4.2 Windows PowerShell

- 以 `-NoProfile -NonInteractive` 启动。
- 不覆盖执行策略。
- 默认脚本只能包含读取操作。
- 证书存储必须通过 .NET `X509Store` 的 `ReadOnly` 标志打开。

## 5. Evidence ZIP 规范

必须包含：

- `manifest.json`
- `timeline.json`
- `collectors/catalog.json`
- `validation/preflight.json`
- `collectors/`
- `evidence/`
- `validation/`
- `logs/`

不得包含：

- 诊断、根因或概率字段。
- 明文私钥。
- 为运行采集器而下载的第三方二进制。
- ZIP 绝对路径、反斜杠路径、`..` 逃逸路径或重复条目。

`manifest.json` 的正式约束位于 `schemas/manifest.schema.json`。

## 6. Collector 合同

每个 Collector 必须声明：

- 稳定 ID 和分类。
- Provider、描述、权限和 passive/active 模式。
- 精确查询或脚本。
- 超时。
- 原始输出路径和媒体类型。
- 错误状态。
- 自动化测试和验收条件。

标准错误码：

- `COLLECTOR_FAILED`
- `COLLECTOR_TIMEOUT`

后续扩展错误码时不得改变既有含义。

## 7. 发布质量门

发布前必须执行：

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/cnd
cnd collect（osquery 冒烟）
cnd collect（PowerShell 冒烟）
cnd validate（两份冒烟 ZIP）
```

安装 `golangci-lint` 的发布环境还必须执行对应 lint；本仓库不在运行时自动下载 linter。

## 8. Roadmap

- **v4（当前）**：CLI、Evidence Store、Manifest、确定性 ZIP、Validation、osquery 与 PowerShell 基线。
- **v4.1**：原生 Windows API Collector、签名侧车文件、可配置隐私策略。
- **v4.2**：GUI；GUI 只能编排相同 Collector 合同，不复制采集逻辑。
- **v5**：带签名目录和兼容性检查的 Plugin SDK。
- **v6**：跨平台 Provider；Windows 专属 Schema 不得强行复用。

## 9. 变更治理

任何与本文件冲突的 RFC、ADR、Issue 或代码都必须先更新 MasterSpec，并在 PR 中明确列出兼容性影响。
