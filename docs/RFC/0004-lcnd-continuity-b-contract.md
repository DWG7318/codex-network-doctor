# RFC 0004：应用级私有出站与 continuity-b 隔离合同

状态：Accepted for LCnd v0.4.0
日期：2026-08-13

## 1. 目标

LCnd v0.4.0 在 v0.3.0 simulation-only Ubuntu Network Core 上发布可公开审计的应用级私有出站
合同与离线工具链。未来 LCSP 可以托管服务，LCapi 可以按 Provider、模型、渠道或目标集引用命名出口；
本版本不修改 LCSP、LCapi、LCgpu，不连接服务器，不导入真实节点，不部署生产服务。

仓库实现完成不等于 LCSP 接入完成，不等于 LCapi 接入完成，也不等于生产验收完成。

## 2. 永久保护边界

现有公网 TCP/443/HAProxy→Caddy/Xray/客户端线路是永久保护目标。v0.4.0 不拆除、修改、迁移、
替换、复用或验证该生产线路，也不修改 SSH TCP/22、HTTP TCP/80、UDP/443、LCSP Access、
LCgpu、Tailnet、默认路由、全局 DNS/hosts、防火墙或系统代理。

新服务只能监听：

- loopback；或
- 带显式 `lcsp-network://` 批准引用的私有地址。

公网地址、匿名访问、TCP 22/80/443、整机 TUN、`auto_route`、全局流量接管和任何宿主网络变更
都必须在离线验证阶段失败关闭。LCnd 不提供 sudo、systemd、ip、nft、iptables、route、
resolvectl、tailscale、docker、ssh、curl 或 wget 执行入口。

## 3. 稳定线路身份

同一 `RouteIdentity` 模型固定三类身份，不复制国内与海外两套实现：

| `class` | 稳定 `id` | 含义 |
|---|---|---|
| `direct` | `direct` | 命名且已批准的原直连路径 |
| `china_egress` | `lcnd-egress-cn` | 中国出口 |
| `global_egress` | `lcnd-egress-global` | 全球出口 |

`lcsp-continuity-b` 与 `lcnd-egress-cn` 必须拥有不同的 workload、endpoint、credential/secret
reference、probe plan、receipt domain、rollback domain 和 inbound tag。两者都必须声明与
`old-public-tcp-443` 分离，任一交叉引用都使合同无效。

## 4. 服务计划与最小鉴权

`schemas/network-continuity-plan.schema.json` 定义服务计划。公共计划只包含：

- 版本、不可变计划 ID、目标平台和非秘密主机引用；
- workload、稳定 route identity、私有 endpoint reference；
- 命名 consumer allowlist、identity reference 和 credential reference；
- connect/request/idle 有界超时；
- `fail_closed`、`approved_direct` 或 `pause_business` 失败策略；
- candidate SHA-256、保护目标、probe、receipt 和 rollback domain；
- 全部为 `false` 的宿主网络影响声明；
- `execution_authority=lcsp_required`、`production_authorized=false`、`evidence_mode=synthetic`。

`approved_direct` 只允许回到稳定身份 `direct`，不能回到未命名或未知线路。任何失败策略都不能扩大
consumer allowlist。

`schemas/network-secret-ref.schema.json` 只允许 `lcnd-secret://` 引用。秘密值、URL、密码、Token、
订阅和私钥禁止进入 Git、Schema 样例、stdout、stderr、错误、摘要、probe、receipt 或报告。

## 5. LCapi 消费合同

`schemas/network-application-attachment.schema.json` 与 Go `ApplicationAttachment` 是 LCnd 发布的
非秘密消费接口，不是对 LCapi 的代码修改。每个 attachment 必须：

- 命名一个 consumer 和 consumer identity reference；
- 通过 Provider、模型、渠道或目标集选择明确请求；
- 引用 `lcnd-egress-cn` 或 `lcnd-egress-global`、私有 endpoint 和 credential reference；
- 声明有界超时、失败策略和业务 probe；
- 回滚时只 detach 该 attachment，并恢复稳定 `direct` 路径；
- 保持 `production_authorized=false`。

未绑定请求与非指定请求保持原路径。合同禁止匿名、通配所有消费者和静默未知线路回退。

## 6. 分层 probe 与离线 evaluator

Probe 类固定为：

- `direct_control_probe`
- `direct_application_probe`
- `egress_continuity_probe`
- `egress_application_probe`
- `protected_target_probe`
- `rollback_probe`
- `secret_leak_probe`

continuity 与 application probe 不能混用。direct 与 egress 分别记录 route identity、脱敏 endpoint
identity、样本数和状态类，不记录完整 URL、query、请求体、响应体或凭据。比较性判断每层至少需要
三个样本；缺少任一层或样本不足时输出 `INSUFFICIENT_EVIDENCE`。

v0.4.0 evaluator 只接受 `evidence_mode=synthetic`，输出 `SYNTHETIC_PASS`、`SYNTHETIC_FAIL` 或
`INSUFFICIENT_EVIDENCE`，并固定 `production_healthy=false`。这些结果不能描述为真实 Ubuntu、真实
节点、真实 LCapi 请求或生产健康。

## 7. 保护目标与回滚

`samples/network/protected-platform.synthetic.json` 固定保护旧公网 443 所有权、HAProxy/Xray/客户端
指纹、SSH 22、HTTP 80、UDP 443、LCSP/LCapi/LCgpu/Owner loopback、Tailnet 偏好、默认路由、
全局 DNS、防火墙和系统代理。guard 只读检查 endpoint 冲突和合同声明，不执行网络操作。

所有生成文件使用确定性 JSON、SHA-256 和 `O_EXCL`，不得覆盖已有文件。rollback receipt 同时绑定
expected 与 observed candidate 摘要，并只允许一个 workload 和一个 attachment scope。

以下情况至少触发 rollback 或 fail-closed：保护目标漂移、旧线路健康回归、公网 443 冲突、
SSH/LCSP Access 回归、全局路由/DNS/防火墙/系统代理/Tailnet 变化、LCgpu 控制变化、LCapi
非 opt-in 流量变化、秘密泄露、candidate 摘要不匹配、workload 域冲突、probe 超时 quorum 失败、
资源越界、未武装回滚或缺少生产授权。

`lcsp-continuity-b` 的回滚不能 detach `lcnd-egress-cn`；`lcnd-egress-cn` 的回滚不能停止
`lcsp-continuity-b`。两者都不能修改旧 TCP/443、HAProxy、Caddy、Xray 或客户端线路。

## 8. CLI

v0.4.0 增加这些纯离线命令：

```text
cnd-network continuity validate --input PLAN.json [--format text|json]
cnd-network continuity candidate-summary --input PLAN.json --output SUMMARY.json [--format text|json]
cnd-network continuity probe-plan --input PLAN.json --output PROBES.json [--format text|json]
cnd-network continuity evaluate --plan PLAN.json --evidence EVIDENCE.json --output EVALUATION.json [--format text|json]
cnd-network continuity guard-check --input PLAN.json --protected PROTECTED.json --output GUARD.json [--format text|json]
cnd-network continuity rollback-evaluate --receipt RECEIPT.json --output DECISION.json [--format text|json]
cnd-network continuity attachment-render --input ATTACHMENT.json --output CANONICAL.json [--format text|json]
```

文件读取失败返回 1，已有输出或写入失败返回 1，参数或合同违规返回 2。错误只输出稳定代码，不回显
原始输入、文件内容或秘密。所有输入上限为 1 MiB，所有 JSON 严格拒绝未知字段与尾随文档。

这些命令只生成候选摘要、probe、evaluation、guard、rollback decision 和 attachment 文件；没有
apply、deploy、service lifecycle 或主机网络执行语义。

## 9. 交付状态

v0.4.0 完成的是公开仓库合同、Go validator/evaluator、确定性文件工具与明显合成样例。以下事项未完成：

- LCSP 托管、秘密注入、服务生命周期和真实回执；
- LCapi Provider/模型接入；
- LCgpu 变更；
- 真实 Ubuntu、真实节点、真实业务 probe；
- 生产灰度、生产验收、tag 和 release。

任何未来生产工作都需要 LCSP、LCapi 与 Owner 分别授权和验收，不能从本 RFC 推导授权。
