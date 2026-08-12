# RFC 0003: Ubuntu Network Core Simulation Contract

状态：Accepted for LCnd v0.3.0
日期：2026-08-12
目标平台：Ubuntu 24.04 amd64
目标引擎：sing-box 1.13.12

## 1. 目的

LCnd 需要在保持 `cnd.exe` 被动证据合同不变的前提下，为自有服务器和终端建立统一的双向网络能力。
v0.3.0 先交付可审计的基础核心：严格输入、确定性规划、受保护 candidate 渲染、合成执行回执和
合成健康报告。

本 RFC 不授权真实网络执行。LCSP 集成、真实节点、sudo/systemd、引擎安装、服务管理和生产路由
全部延后。

## 2. 方向模型

| Profile | 节点方向 | 默认出口 | 选定目标 | 典型主机 |
| --- | --- | --- | --- | --- |
| `global_direct_domestic_tunnel` | `china_egress` | `direct` | `lcnd-tunnel` | 海外主机访问国内服务 |
| `domestic_direct_global_tunnel` | `global_egress` | `direct` | `lcnd-tunnel` | 国内主机访问国外服务 |

两种方向共享同一套类型、规划器、保护规则和状态机。管理目标固定走 `direct`，并排在所有隧道规则
之前；受保护目标与选定目标重叠时返回 `TARGET_OVERLAP`。

## 3. 命令合同

```text
cnd-network version
cnd-network validate --kind node|target-set|route-profile --input FILE [--format text|json]
cnd-network plan --node FILE --targets FILE --protected FILE --profile FILE --output DRAFT.json [--format text|json]
cnd-network render --draft DRAFT.json --material SECRET.json --candidate CANDIDATE.json --plan PLAN.json [--format text|json]
cnd-network simulate --plan PLAN.json --scenario success|verification_failed|timeout --output RECEIPT.json [--format text|json]
cnd-network report --plan PLAN.json --receipt RECEIPT.json --output REPORT.json [--format text|json]
```

所有输入文件上限为 1 MiB。`--format` 默认是 `text`；JSON 模式只打印不含材料的摘要对象。
所有输出以新文件方式创建，已存在时拒绝覆盖。CLI 不接受 `-` 作为材料或 candidate 路径。

## 4. 文件与敏感度

| 文件 | 内容 | 敏感度 | 可提交 Git |
| --- | --- | --- | --- |
| node descriptor | 元数据和 `lcnd-secret://` 引用 | 公共合同 | 仅合成样例 |
| target set / route profile | 目标与策略引用 | 公共合同 | 仅合成样例 |
| draft | `VALIDATED` 确定性规划 | 公共产物 | 仅合成产物 |
| node material | 原始 outbound 与凭据 | 高敏感 | 否 |
| candidate | 可运行 sing-box 配置 | 高敏感 | 否 |
| finalized plan | candidate SHA-256 与脱敏计划 | 公共产物 | 仅合成产物 |
| receipt / health report | 合成状态与发现 | 公共产物 | 仅合成产物 |

candidate 使用 `O_EXCL` 创建；在支持 Unix 权限的平台上模式为 `0600`。最终 plan、render summary、
receipt、report、stdout、stderr 和错误信息不得包含 node material 或 candidate 字节。错误只暴露文件角色
与稳定错误码，不回显输入内容。

## 5. 状态机

```text
strict inputs
    -> VALIDATED
    -> PLAN_READY + protected candidate
    -> VERIFIED             (scenario=success)
    -> ROLLED_BACK          (scenario=verification_failed|timeout)
    -> SIMULATED_PASS       (report from VERIFIED)
    -> SIMULATED_ROLLBACK   (report from ROLLED_BACK)
```

`VERIFIED` 和 `ROLLED_BACK` 都是合成执行状态。报告固定使用 `evidence_mode=synthetic`，不得翻译为
`HEALTHY`、`UNHEALTHY` 或任何真实生产状态。

## 6. 退出码

| 退出码 | 含义 |
| --- | --- |
| `0` | 命令成功 |
| `1` | 读取、创建、写入或渲染等操作失败 |
| `2` | 参数无效、合同拒绝或状态转换无效 |

## 7. 不兼容与失败关闭

以下情况必须拒绝，不得自动降级或猜测：

- 未知 JSON 字段、尾随第二个 JSON 文档或不支持的 `schema_version`。
- 不符合 `lcnd-secret://` 形式的 secret reference，或公共 node 中嵌入 URL/凭据。
- 不支持的方向、Profile、target kind、协议、端口或未 masked CIDR。
- Profile 与 node ID/方向不匹配，或 target set 引用不匹配。
- 管理目标与隧道目标重叠。
- draft 不是 `VALIDATED`，plan 不是 `PLAN_READY`，或 `simulation_only` 不为 true。
- material 的 `secret_ref` 与 draft 不匹配，或 outbound 类型为 `direct`/`block`。
- candidate、plan、receipt、report 等输出路径已经存在。
- finalized plan 与 receipt 的 plan ID 或 candidate SHA-256 不匹配。

未知的未来 schema、引擎版本或平台不能被当前二进制静默接受。兼容性扩展必须先更新 Schema、Go 合同、
测试和本 RFC。

## 8. 构建与部署边界

Windows 构建在保留现有四个可执行文件的同时增加 `cnd-network.exe`。`scripts/build-linux.ps1` 只以
`GOOS=linux`、`GOARCH=amd64`、`CGO_ENABLED=0` 交叉编译 `dist/linux-amd64/cnd-network`，并在结束时
恢复调用者环境变量。

仓库不捆绑或安装 sing-box，不生成服务单元，不改变路由/DNS/防火墙，不连接真实服务器。未来 LCSP
适配器只能代办基础设施操作；LCnd 继续拥有网络策略、计划、验证、报告与回滚合同。

## 9. 验收

- 严格合同、双向规划、确定性摘要和保护目标重叠测试通过。
- candidate 权限、不可覆盖、脱敏计划和错误不泄漏测试通过。
- success、verification failure、timeout 与 synthetic report 状态测试通过。
- CLI 端到端合成链通过，公共产物不含测试材料。
- `go test ./...`、`go vet ./...`、Windows 构建和 Ubuntu amd64 交叉构建通过。
