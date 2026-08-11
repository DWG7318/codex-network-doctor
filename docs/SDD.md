# System Design Description

## 1. 目标

CND 的 `cnd.exe` 把一次主机观察转换为可独立校验的 Evidence ZIP。该路径按
“Provider → Collector Engine → Evidence Store → Manifest → Deterministic ZIP → Validation”单向流动，
不存在修复回路。独立的 `cnd-network` 使用另一条 simulation-only 路径，不改变 Evidence Engine。

```mermaid
flowchart LR
    CLI["CLI"] --> Selection["Provider Selection"]
    Selection --> OQ["osquery SELECT Collectors"]
    Selection --> PS["PowerShell Read-only Collectors"]
    OQ --> Engine["Collector Engine"]
    PS --> Engine
    Engine --> Store["Immutable Evidence Store"]
    Store --> Manifest["Manifest Builder"]
    Manifest --> Zip["Deterministic ZIP Builder"]
    Zip --> Validator["Integrity Validator"]
```

## 2. 模块

### CLI

命令为 `collect`、`validate`、`catalog` 和 `version`。CLI 负责参数和退出码，不包含采集逻辑。

### Provider Selection

`auto` 优先选择显式路径或 PATH 中的 `osqueryi.exe`，否则选择 `powershell.exe`/`pwsh.exe`。显式 Provider 不可用时直接失败，不偷偷更换实现。

### Collector Engine

Engine 顺序执行当前目录，给每个采集器创建独立超时上下文。顺序执行避免多个系统枚举同时竞争 WMI、CIM 或 IP Helper 资源。失败被记录后继续。

### Evidence Store

Store 只接受相对安全路径，并以 `O_EXCL` 创建文件。绝对路径、卷路径、根路径、`..` 逃逸和重复写入都会被拒绝。

### Manifest

Manifest 保存工具版本、主机名、采集窗口、完整/部分状态、Collector 结果及文件 SHA-256。Collector 与文件在序列化前按稳定键排序。

### ZIP Builder

ZIP 条目按名称排序，时间固定为 1980-01-01，文件与目录权限固定。目录条目显式存在。ZIP 本身的 SHA-256 在命令摘要中返回。

### Validator

Validator 检查：

- 不安全路径和重复条目。
- 必需目录及文件。
- Manifest JSON 与 Schema 版本。
- Manifest 重复路径。
- 缺失、额外、大小不符或 SHA-256 不符的文件。
- 单文件 256 MiB 校验上限。

### Ubuntu Network Core

`internal/networkcore` 保存不含任意 map 的严格领域合同、验证器、确定性规划器、状态机和报告构造器。
`internal/networkengine/singbox` 是 sing-box `1.13.12` 适配器；只有该适配器接触受保护节点材料，
并把材料渲染到新建的 candidate 文件。`internal/networkcli` 只负责参数、文件状态门和退出码。

```mermaid
flowchart LR
    Inputs["Strict public contracts"] --> Planner["Deterministic planner"]
    Planner --> Draft["VALIDATED draft"]
    Material["Protected node material"] --> Renderer["sing-box renderer"]
    Draft --> Renderer
    Renderer --> Candidate["Sensitive candidate (0600)"]
    Renderer --> Plan["Redacted PLAN_READY plan"]
    Plan --> Simulation["Synthetic state machine"]
    Simulation --> Receipt["Synthetic receipt"]
    Receipt --> Report["Synthetic health report"]
```

候选配置中的管理规则位于隧道规则之前，最终出口为 `direct`。规划器在离线状态下检查域名后缀、
协议、端口和 CIDR 重叠，不执行 DNS 或任何网络 I/O。

## 3. 数据状态

```mermaid
stateDiagram-v2
    [*] --> Collecting
    Collecting --> Complete: 所选 Collector 全部成功
    Collecting --> Partial: 至少一个失败或超时
    Complete --> Packaged
    Partial --> Packaged
    Packaged --> Validated
    Packaged --> Rejected: 完整性校验失败
```

`partial` 是合法证据包状态，不等同于 ZIP 无效。ZIP 有效只说明结构和已记录内容一致。

Ubuntu Network Core 使用独立状态：

```mermaid
stateDiagram-v2
    [*] --> VALIDATED: plan
    VALIDATED --> PLAN_READY: render protected candidate
    PLAN_READY --> VERIFIED: simulate success
    PLAN_READY --> ROLLED_BACK: simulate verification failure or timeout
    VERIFIED --> SIMULATED_PASS: report
    ROLLED_BACK --> SIMULATED_ROLLBACK: report
```

这些状态全部是合成状态，不映射到真实服务状态。

## 4. 错误处理

- 命令执行失败：保留 stdout/stderr，Collector 标为 `failed`。
- 超时：外部进程由上下文终止，Collector 标为 `timed_out`。
- Evidence Store、Manifest、ZIP 或 Validator 失败：整个命令返回非零。
- 已有输出：默认拒绝；`--force` 才可覆盖。
- `cnd-network` 公共输出与 candidate 均使用 `O_EXCL`，不提供覆盖开关。
- `cnd-network` 将文件错误收敛为角色和稳定代码，不回显输入内容或节点材料。

## 5. 安全设计

- osquery 禁用事件、扩展和日志，避免后台状态及第三方扩展。
- PowerShell 不使用 `ExecutionPolicy Bypass`。
- 默认目录不包含联网探测或配置写入命令。
- 证据和暂存目录以仅当前用户权限创建；最终组织级 ACL 由部署环境负责。
- Ubuntu 节点描述只保存 `lcnd-secret://` 引用；节点材料和 candidate 是敏感资产。
- 最终 plan、render summary、receipt、report、stdout/stderr 和错误不得包含节点材料。
- v0.3.0 不包含 sudo、systemd、引擎安装、真实节点或生产网络命令。

## 6. 已知限制

- 主机名和时间来自可能不可信的被调查主机。
- CND 没有硬件根信任或远程证明。
- 当前没有内建加密、数字签名、GUI、EventLog 时间线或 WFP/ETW 持续捕获。
- ZIP 构建当前按文件读取到内存；超大事件日志应由未来流式 Artifact 类型实现。
- Ubuntu Network Core 当前只支持 `ubuntu-24.04` / `amd64` 目标和 sing-box `1.13.12` 配置形状。
- 当前健康报告只基于 `evidence_mode=synthetic`；真实链路探测、LCSP 适配和生产执行均未实现。
