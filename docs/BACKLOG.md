# Issue-ready Backlog

工作区最初没有 Git 仓库或远程仓库，因此没有擅自创建外部 GitHub Issues。以下条目可在建立远程仓库后按顺序转为 Issues。

## Ubuntu Network Core 后续切片

以下工作明确不属于 v0.4.0。LCSP 当前升级完成前，不修改 LCSP，也不实现其适配器。v0.4.0 只完成
公开合同、离线 validator/evaluator 与合成样例；仓库实现完成不等于 LCSP 接入完成，不等于 LCapi
接入完成，也不等于生产验收完成。

1. **[probe] Ubuntu 24.04 真实健康检测：v0.5.0完成首个只读切片**
   - 已实现接口、DNS/路由快照、真实DNS/TCP/TLS/HTTPS HEAD、显式已有代理对比、live报告与可解释评分。
   - 不自动修复，与Windows Evidence ZIP和主动测速合同隔离。
   - 后续仍需独立吞吐测速、长期趋势、policy-routing/逐跳证据扩展；本版不测带宽。

2. **[integration] LCSP 基础设施执行适配器**
   - LCnd 保持网络策略、计划、验证和回滚合同所有权。
   - LCSP 仅代办服务器身份、权限、文件和服务生命周期操作。
   - 接口必须支持 dry-run、幂等、审计和最小权限；需等待 LCSP 升级完成后共同设计。

3. **[nodes] Owner 提供的真实节点导入**
   - 真实材料进入受控 secret store，不进入仓库、命令行、日志或公共报告。
   - 分别验收中国出口与全球出口，且必须有过期、撤销和轮换流程。

4. **[execute] 有审批与回滚的生产路由执行**
   - 在只读探测和 LCSP 适配器验收后单独设计。
   - 必须先备份、预检管理通道、限时验证，并在失败或超时时自动回滚。
   - simulation-only CLI 不得被原地改造成隐式执行入口。

5. **[pilot] 海外 LCapi 到国内模型的首个真实试点**
   - 只在 Owner 提供节点并批准目标服务器后进行。
   - 先建立基线与只读证据，再小范围灰度；不得把试点凭据或原始配置提交到公开仓库。

6. **[integration] LCapi application attachment 实际接入**
   - 由 LCapi 评估和实现按 Provider、模型、渠道或目标集引用命名出口。
   - 未绑定请求保持原 `direct`；失败策略和回滚必须遵循 v0.4.0 合同。
   - 这项工作必须在 LCapi 仓库单独授权、测试和验收，LCnd 不越界修改。

7. **[acceptance] 真实保护基线与生产验收**
   - 由 LCSP/Owner 在单独授权下验证旧公网 TCP/443/HAProxy→Caddy/Xray/客户端线路、SSH、LCSP
     Access、LCgpu、Tailnet 和系统网络设置保持不变。
   - synthetic guard、probe 和 rollback decision 不能替代真实设备、真实节点与真实业务验收。

8. **[platform] LCSP 托管与秘密注入**
   - 实现低权限服务、私有 endpoint、credential injection、备份、回执和单 workload 回滚。
   - 不复用 `lcsp-continuity-b` 与 `lcnd-egress-cn` 的 endpoint、secret、probe、receipt 或 rollback 域。

## v4.1

1. **[security] Add detached manifest signature**
   - 支持组织提供的证书或密钥。
   - 签名写在 ZIP 外部，避免递归。
   - 验收：篡改 ZIP 或摘要后验证失败。

2. **[privacy] Configurable redaction profile**
   - 保留 raw-first 原则，同时允许操作者选择明确的最小化目录。
   - 不允许静默修改原始文件。

3. **[collector] Windows EventLog export**
   - 使用来源事件时间生成 timeline。
   - 支持流式 Artifact 与大小上限。
   - 记录 channel、record ID、provider 和 bookmark。

4. **[collector] Native IP Helper API provider**
   - 为接口、路由、邻居和 sockets 提供不依赖 PowerShell/osquery 的实现。
   - 与 osquery/PowerShell 输出做语义对照，不以其中任一方为绝对真值。

5. **[packaging] Stream large artifacts into ZIP**
   - 不把整个 Artifact 读入内存。
   - 保持字节确定性和 SHA-256。

6. **[compat] Windows version matrix**
   - Windows 10、11、Server 2019、2022、2025。
   - 标记缺失 Cmdlet、权限和字段差异。

## v4.2

7. **[gui] Evidence collection GUI**
   - GUI 只调用现有 CLI/Collector 合同。
   - 运行前显示目录、敏感数据提示和输出路径。
   - 完成后显示 SHA-256 和完整/部分状态。

## v5

8. **[sdk] Signed Collector catalog**
9. **[sdk] Out-of-process plugin protocol**
10. **[sdk] Capability and privacy declarations**
