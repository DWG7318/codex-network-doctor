# Codex Network Doctor MasterSpec

版本：1.4
状态：唯一权威设计文档
适用交付：v4 Evidence Engine CLI、LCnd v0.3.0 Ubuntu Network Core、LCnd v0.4.0 应用级私有出站合同

## 1. 项目使命

构建一个企业级网络证据与受控网络能力平台。Windows Evidence Engine 负责采集、封装和验证证据；
独立的 Ubuntu Network Core 负责严格校验、双向分流规划、受保护候选渲染与合成模拟。

CND 的被动证据采集器 `cnd.exe` 不执行：

- 网络、代理、防火墙、DNS、路由或服务修复。
- 主动 TLS、HTTP、WebSocket、DNS 或端口探测。
- Diagnosis、Root Cause、Probability、Recommendation 或 AI Guess。

独立交付的 `cnd网速检测.exe` / `cnd网络修复.exe speed` 允许操作者显式启动有界主动测速和
规则化的组件范围判断。它不属于 Evidence ZIP 采集目录，不改变 `cnd.exe` 的被动边界；合同见
[RFC 0002](RFC/0002-active-speed-path-diagnostics.md)。

独立交付的 `cnd-network` 不改变 `cnd.exe` 的被动边界。v0.3.0 仅面向 Ubuntu 24.04 amd64 的
simulation-only 工作流，不执行 sudo、systemd、服务安装或生产路由变更；合同见
[RFC 0003](RFC/0003-ubuntu-network-core.md)。

v0.4.0 在相同 simulation-only 核心上增加应用级私有出站服务、LCapi 消费、分层 probe、保护目标
与单域回滚合同。它只执行严格验证和离线文件生成；合同见
[RFC 0004](RFC/0004-lcnd-continuity-b-contract.md)。

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
11. **Explicit active diagnostics**：主动测速只存在于独立诊断/修复工具，必须由操作者显式运行，
    且不得修改 TUN、DNS、路由、系统代理或进程状态。
12. **Simulation before execution**：Ubuntu 网络能力必须先生成确定性计划、受保护 candidate、
    合成回执和合成健康报告；v0.3.0 不提供真实执行入口。
13. **Secret indirection**：公共节点合同只允许 `lcnd-secret://` 引用。节点材料与生成的 candidate
    不得进入 Git、stdout、最终 plan、receipt、report 或错误信息。
14. **Protected management path**：管理目标必须直连；与隧道目标发生域名、端口、协议或 CIDR
    重叠时，规划必须失败关闭。
15. **Application scope only**：私有出站只服务命名 allowlist consumer；不得改变宿主默认路由、
    全局 DNS/hosts、防火墙、系统代理、Tailnet 或整机 TUN。
16. **Explicit failure policy**：出口失败只能 fail-closed、暂停该业务或回到命名且已批准的
    `direct`；不得静默使用未知线路或扩大 consumer 范围。
17. **Independent workload domains**：`lcsp-continuity-b` 与 `lcnd-egress-cn` 的 endpoint、secret、
    probe、receipt、rollback 和 inbound tag 必须分离。

## 3. 信任和边界

- CND 可以证明 ZIP 内文件与 manifest 一致，不能证明受调查主机或操作系统 API 未被攻击者篡改。
- `started_at` 和 `finished_at` 来自主机时钟；不自动认定其准确。
- `timeline.json` 只承载证据中具有来源时间戳的事件。当前快照目录不创造伪事件，因此为空。
- `validation` 只做结构、完整性和 Schema 检查，不做诊断。
- `cnd-network` 的 `SIMULATED_PASS` 只证明合成状态机完成，不证明真实服务器、节点或链路健康。
- sing-box 目标版本为 `1.13.12`，但二进制、安装和生命周期管理不属于 v0.3.0 交付。
- LCSP 未来只可充当基础设施与提权代办层；网络策略、计划和验证合同仍由 LCnd 所有。
- v0.4.0 的 `SYNTHETIC_PASS`、`APPLY_ELIGIBLE` 和 guard pass 都不是生产授权或真实健康证明。
- 仓库实现完成不等于 LCSP 接入完成，不等于 LCapi 接入完成，也不等于生产验收完成。
- 旧公网 TCP/443/HAProxy→Caddy/Xray/客户端线路是永久保护目标，v0.4.0 未触碰或验证它。

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
go test ./internal/networkcore ./internal/networkengine/singbox ./internal/networkcli ./cmd/cnd-network
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/cnd-network
cnd collect（osquery 冒烟）
cnd collect（PowerShell 冒烟）
cnd validate（两份冒烟 ZIP）
```

安装 `golangci-lint` 的发布环境还必须执行对应 lint；本仓库不在运行时自动下载 linter。

## 8. Roadmap

- **v4（当前）**：CLI、Evidence Store、Manifest、确定性 ZIP、Validation、osquery 与 PowerShell 基线。
- **独立运维工具**：显式 v2rayN TUN 修复、撤销和有界网络路径测速；不得改变 Evidence Engine 合同。
- **LCnd v0.3.0 Ubuntu Network Core（v0.4.0 基础）**：双向确定性规划、受保护 sing-box 1.13.12 candidate、
  合成执行和合成健康报告；不包含真实节点或生产执行。
- **LCnd v0.4.0 应用级私有出站合同（当前）**：稳定 direct/中国/全球 route identity、私有 endpoint、
  命名鉴权、LCapi attachment、分层 synthetic probe、保护 guard 与单域 rollback decision；不包含
  LCSP/LCapi 接入或生产验收。
- **Ubuntu 后续切片**：只读链路探测、LCSP 基础设施适配器、受审批的真实节点与可回滚执行；
  必须分别设计和验收，不得从 simulation-only 隐式升级。
- **v4.1**：原生 Windows API Collector、签名侧车文件、可配置隐私策略。
- **v4.2**：GUI；GUI 只能编排相同 Collector 合同，不复制采集逻辑。
- **v5**：带签名目录和兼容性检查的 Plugin SDK。
- **v6**：跨平台 Provider；Windows 专属 Schema 不得强行复用。

## 9. 变更治理

任何与本文件冲突的 RFC、ADR、Issue 或代码都必须先更新 MasterSpec，并在 PR 中明确列出兼容性影响。

## 10. Ubuntu Network Core v0.3.0 合同

- 平台固定为 `ubuntu-24.04` / `amd64`，网络引擎合同固定为 sing-box `1.13.12`。
- 公共输入包括 node descriptor、target set 和 route profile；未知字段、版本不匹配、方向不匹配、
  引用不匹配和受保护目标重叠均拒绝。
- `plan` 产生 `VALIDATED` 草案；`render` 产生权限受限 candidate 和不含材料的 `PLAN_READY` 计划；
  `simulate` 只能产生 `VERIFIED` 或 `ROLLED_BACK` 合成回执；`report` 只能产生
  `SIMULATED_PASS` 或 `SIMULATED_ROLLBACK`。
- candidate 可以包含真实节点材料，因此是敏感资产；其他公共产物只能包含 secret reference 和摘要。
- v0.3.0 不捆绑或安装网络引擎，不调用 sudo/systemd，不接触生产服务器，不与 LCSP 集成。

## 11. 应用级私有出站 v0.4.0 合同

- 稳定线路身份固定为 `direct/direct`、`lcnd-egress-cn/china_egress`、
  `lcnd-egress-global/global_egress`，双向使用同一 Go 与 Schema 模型。
- 服务 endpoint 只能是 loopback 或带批准引用的私网；TCP 22/80/443、公网地址、匿名鉴权和任何
  宿主网络影响声明都被拒绝。
- 公共输入只含 consumer identity、credential/secret reference、摘要、超时、失败策略、probe 和
  rollback scope，不含秘密值或完整目标 URL。
- direct 与 egress probe 分层评估，每层至少三个样本；缺层或样本不足输出
  `INSUFFICIENT_EVIDENCE`。v0.4.0 只接受 synthetic evidence。
- 所有生成物确定且以 `O_EXCL` 新建；candidate 摘要、rollback receipt 与 decision 绑定 SHA-256，
  回滚只作用于一个 workload 和一个 attachment。
- `lcsp-continuity-b` 与 `lcnd-egress-cn` 不能共享 workload、endpoint、secret、probe、receipt、
  rollback 或 inbound tag。
- `cnd-network continuity` 只提供 validate、candidate-summary、probe-plan、evaluate、guard-check、
  rollback-evaluate 和 attachment-render，不提供 apply/deploy 或主机命令执行。
- v0.4.0 不修改 `cnd.exe` 被动采集、`cnd网络修复.exe` 显式修复、撤销程序或
  `cnd网速检测.exe` 只读测速边界。
