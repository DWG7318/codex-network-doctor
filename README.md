# Codex Network Doctor

Codex Network Doctor 的 `cnd.exe` 是一个 Windows 网络证据采集器。它只采集和封装证据，不修改网络设置、不执行修复，也不输出诊断、根因、概率或 AI 推测。

项目另外提供显式运行、范围受限的 `cnd网络修复.exe`，用于修复 v2rayN 重复实例、
遗留内核或 Windows 排除端口范围造成的控制端口冲突，并确保 TUN 启用。两个程序职责独立。

`cnd网速检测.exe` 提供显式、只读的分层测速，区分系统/TUN、HTTP 代理和 SOCKS5 代理入口，
帮助判断瓶颈位于本地入口还是入口汇合后的远端链路。它不请求管理员权限，也不修改或重启 v2rayN。

当前交付是 MasterSpec 路线中的 **v4 Evidence Engine CLI**：

- osquery 5.23.1 适配器：13 个被动查询，适合结构化采集。
- Windows PowerShell 降级适配器：15 个只读采集器，不依赖第三方安装。
- 采集器级超时与失败隔离；单项失败不阻断证据包生成。
- 原始 stdout/stderr 留存。
- 每个文件的 SHA-256 与大小写入 `manifest.json`。
- 固定目录顺序、ZIP 时间戳和权限位，保证相同暂存内容得到相同 ZIP 字节。
- 独立的 `validate` 命令检查路径安全、重复条目、必需文件、大小和 SHA-256。

## 应用级私有出站合同（v0.4.0）

`cnd-network` 现在发布一套可供未来 LCSP 托管、供 LCapi 按 Provider/模型消费的应用级私有
出站合同与离线工具链。它只生成和评估非秘密 JSON 工件，不连接服务器、不安装网络引擎、不修改
LCSP/LCapi/LCgpu，也不执行生产部署。

三类稳定线路身份由同一模型表达：`direct`、`lcnd-egress-cn` 和 `lcnd-egress-global`。新服务只能
监听 loopback 或带 `lcsp-network://` 批准引用的私网；公网监听、匿名访问、默认路由、全局
DNS/hosts、防火墙、系统代理、Tailnet 和整机 TUN 变化一律失败关闭。

```powershell
.\bin\cnd-network.exe continuity validate `
  --input .\samples\network\egress-cn.synthetic.json

.\bin\cnd-network.exe continuity candidate-summary `
  --input .\samples\network\egress-cn.synthetic.json `
  --output .\out\egress-cn-summary.json

.\bin\cnd-network.exe continuity probe-plan `
  --input .\samples\network\egress-cn.synthetic.json `
  --output .\out\egress-cn-probes.json

.\bin\cnd-network.exe continuity attachment-render `
  --input .\samples\network\lcapi-egress-attachment.synthetic.json `
  --output .\out\lcapi-egress-attachment.json
```

还提供 `evaluate`、`guard-check` 和 `rollback-evaluate`。所有输入严格拒绝未知字段并限制为 1 MiB；
所有输出使用确定性 JSON 与 `O_EXCL`，不会覆盖已有文件。Probe 与回滚样例固定为
`evidence_mode=synthetic`；样本不足输出 `INSUFFICIENT_EVIDENCE`，不能解释为真实 Ubuntu、真实节点、
真实 LCapi 请求或生产健康。

`lcsp-continuity-b` 与 `lcnd-egress-cn` 的 workload、endpoint、credential/secret、probe、receipt、
rollback 和 inbound tag 必须全部分离。旧公网 TCP/443/HAProxy→Caddy/Xray/客户端线路是永久保护
目标，本版本没有触碰或验证它。

仓库实现完成不等于 LCSP 接入完成，不等于 LCapi 接入完成，也不等于生产验收完成。详细合同见
[RFC 0004](docs/RFC/0004-lcnd-continuity-b-contract.md)。

## Ubuntu 网络规划与模拟（v0.3.0）

`cnd-network` 是与 `cnd.exe`、Windows 修复器和网速检测器相互隔离的 Ubuntu 网络规划 CLI。
当前版本面向 Ubuntu 24.04 amd64，只实现严格校验、双向分流规划、sing-box 候选配置渲染和
合成执行模拟，不执行真实服务器网络变更。

两种方向使用同一套底层模型：

- `global_direct_domestic_tunnel`：海外主机默认直连，指定国内目标经中国出口节点。
- `domestic_direct_global_tunnel`：国内主机默认直连，指定国外目标经全球出口节点。
- 管理网段作为受保护目标始终直连；它与隧道目标重叠时规划失败。

目标引擎固定为 sing-box `1.13.12`，但仓库不捆绑、不下载也不安装 sing-box。仓库中的节点、
目标集和路由 Profile 样例全部是合成数据；模拟回执与健康报告固定使用
`evidence_mode=synthetic`，不能当作真实服务器健康结论。

```powershell
.\bin\cnd-network.exe version
.\bin\cnd-network.exe validate --kind node --input .\samples\network\node-china-egress.synthetic.json
.\scripts\build-linux.ps1
```

命令面包括 `validate`、`plan`、`render`、`simulate` 和 `report`。`render` 读取的节点材料以及生成的
sing-box candidate 都是敏感资产：candidate 只能写入新文件，不得输出到终端或提交到 Git；
最终 plan 只保存 `secret_ref` 与 candidate SHA-256。

LCSP 对接、真实节点、sudo/systemd 代办和生产路由执行均留待后续版本。详细状态机、文件合同、
退出码和不兼容行为见 [RFC 0003](docs/RFC/0003-ubuntu-network-core.md)。

## 双击离线采集

把 `cnd.exe` 复制到故障电脑，然后直接双击：

1. CND 自动使用 Windows 自带的 PowerShell 只读采集后端。
2. 不需要 Agent、网络、Go 环境或安装程序。
3. CND 使用 Windows Known Folder API 查找当前用户的真实桌面路径，兼容 Win10、Win11、OneDrive 和企业文件夹重定向。
4. 证据包自动保存到桌面，文件名类似：

   ```text
   cnd诊断结果-OFFLINE-PC-20260730-153045.zip
   ```

5. 如果系统桌面 API 不可用，才回退到 `cnd.exe` 同目录。
6. 完成后窗口会显示 ZIP 路径，并等待按回车关闭。

PowerShell 只是 CND 在后台调用的 Windows 状态读取接口，操作者不需要打开或输入 PowerShell 命令。需要查看帮助时运行 `cnd.exe help`。

## 双击修复 v2rayN TUN

适用现象：微信等直连软件可以联网，但浏览器和 Codex 无法访问；v2rayN 日志出现 `external controller listen error`、`bind` 或 TUN 无法保持开启。

1. 启动 v2rayN 并选择一个可用节点。
2. 双击 `cnd网络修复.exe`。
3. 接受 Windows UAC 管理员权限提示。程序本身是原生 EXE，不要求操作者打开 PowerShell。
4. 修复器识别包括 `v2rayN2.exe` 在内的数字改名实例，保留最新实例，精确停止其他 v2rayN GUI 及其安装目录内的已知内核。
5. 修复器先备份 `guiNConfig.json`，再确保 `TunModeItem.EnableTun` 为 `true`，重新启动保留的 v2rayN。
6. 修复器检查 IPv4/IPv6、TCP/UDP 的 Windows excluded port range；控制端口命中保留范围时，
   自动选择连续 32 个安全高位端口并迁移 v2rayN 基础端口。
7. 修复器等待 `singbox_tun` 网卡启动，并以不使用系统代理的直连请求验证 OpenAI HTTPS；该请求应由 TUN 接管。
8. 重新打开浏览器和 Codex。

备份默认位于：

```text
%LOCALAPPDATA%\CND\tun-repair\backups
```

双击 `cnd网络修复-撤销.exe` 可以恢复最近一次修复前的 v2rayN TUN 配置并重新启动原实例。

## 双击网速分层检测

保持 v2rayN 节点和 TUN 状态不变，双击：

```text
cnd网速检测.exe
```

也可以从终端显式运行：

```powershell
.\cnd网络修复.exe speed
```

工具自动发现 v2rayN/sing-box 的本地 mixed 端口，按“系统/TUN → HTTP → SOCKS5”顺序执行
三轮 1 MB 测速。结果显示每个样本的 HTTP 状态、字节数、TTFB、总耗时和 Mbps，并输出稳定代码：

- `TUN_PATH_SLOW`：TUN/系统入口持续慢于同出口 SOCKS5。
- `TUN_PATH_UNRELIABLE`：TUN/系统入口三次中至少失败两次，而 SOCKS5 三次成功。
- `HTTP_PROXY_PATH_SLOW`：HTTP CONNECT 入口持续慢于 SOCKS5。
- `HTTP_PROXY_PATH_UNRELIABLE`：HTTP入口三次中至少失败两次，而 SOCKS5 三次成功。
- `SHARED_REMOTE_PATH_UNSTABLE`：TUN和 SOCKS5 同时大幅波动，问题范围在入口汇合之后。
- `INSUFFICIENT_EVIDENCE`：成功样本不足，不强行下结论。

主动测速全程串行、单项有硬超时；不会切换 TUN、修改 DNS/路由/系统代理、停止或重启进程。
详细合同见 [RFC 0002](docs/RFC/0002-active-speed-path-diagnostics.md)。

修复器不会终止占用控制端口的未知程序，也不修改节点、路由、DNS 服务器或 Windows 系统代理，不执行 Winsock/TCP-IP 重置。v2rayN 7.19.5 会按完整 EXE 路径做单实例判断，因此同目录的 `v2rayN.exe` 与 `v2rayN2.exe` 能同时运行；修复器专门消除这一竞争。实现依据见 v2rayN 官方的 [启动代码](https://github.com/2dust/v2rayN/blob/7.19.5/v2rayN/v2rayN.Desktop/Program.cs)、[运行时端口代码](https://github.com/2dust/v2rayN/blob/7.19.5/v2rayN/ServiceLib/Manager/AppManager.cs) 和 [sing-box TUN 配置](https://github.com/2dust/v2rayN/blob/7.19.5/v2rayN/ServiceLib/Services/CoreConfig/Singbox/SingboxInboundService.cs)。

## 离线案例库

案例正文编译进 `cnd.exe` 和 `cnd网络修复.exe`，故障电脑没有网络时也能查询：

```powershell
.\cnd.exe cases
.\cnd.exe case CND-CASE-2026-001

.\cnd网络修复.exe cases
.\cnd网络修复.exe case CND-CASE-2026-001
```

`CND-CASE-2026-001` 记录了真实处置中 v2rayN TUN 因 Windows excluded port range
触发 WSAEACCES/10013 的完整脱敏诊断、修复和验证过程。可直接阅读
[脱敏案例日志](internal/casebook/cases/CND-CASE-2026-001.md)。仓库只保存脱敏案例，
不保存真实诊断包。

## 快速开始

项目自带的便携 Go 工具链目录不会进入版本控制。本机已有 Go 时可以直接构建：

```powershell
.\scripts\build.ps1
```

使用系统 PowerShell 采集：

```powershell
.\bin\cnd.exe collect `
  --provider powershell `
  --output .\out\evidence.zip
```

使用固定版本 osquery：

```powershell
.\scripts\bootstrap-osquery.ps1

.\bin\cnd.exe collect `
  --provider osquery `
  --osquery ".\.tools\osquery-5.23.1.windows_x86_64\Program Files\osquery\osqueryi.exe" `
  --output .\out\evidence-osquery.zip
```

`--provider auto` 会优先使用 PATH 或显式路径中的 `osqueryi.exe`，否则回退到 Windows PowerShell。

验证证据包：

```powershell
.\bin\cnd.exe validate .\out\evidence.zip
```

查看将要执行的采集目录：

```powershell
.\bin\cnd.exe catalog --provider powershell
```

生成不包含真实主机数据的示例包：

```powershell
.\bin\cnd.exe sample --output .\samples\example-evidence.zip
```

默认拒绝覆盖已有 ZIP。只有显式传入 `--force` 才允许替换目标文件。

## Evidence ZIP

```text
manifest.json
timeline.json
collectors/
  catalog.json
evidence/
  osquery/         # 或 windows/
validation/
  preflight.json
logs/
  ...stderr.txt    # 仅在命令产生 stderr 时存在
```

`manifest.json` 不包含自身哈希，其他普通文件必须全部登记。`timeline.json` 在当前快照型采集中为空数组；工具不会把采集完成时间伪装成网络事件时间。

## 安全与隐私

证据可能包含主机名、用户名相关路径、IP、MAC、进程路径、代理/PAC 地址、DNS 缓存、证书元数据和网络连接。请把 ZIP 当作敏感取证材料：

- 仅在获得授权的设备上运行。
- 使用受控存储和传输通道。
- 不要把真实证据包提交到 Git。
- 需要长期保存时，在 ZIP 外层使用组织批准的加密与签名机制。

工具不会导出证书私钥、进程环境变量或进程命令行。详细边界见 [Threat Model](docs/THREAT_MODEL.md)。

## 工程文档

- [MasterSpec](docs/MASTER_SPEC.md)
- [系统设计](docs/SDD.md)
- [Collector Catalog](docs/COLLECTOR_CATALOG.md)
- [Evidence ZIP RFC](docs/RFC/0001-evidence-package.md)
- [Ubuntu 网络核心 RFC](docs/RFC/0003-ubuntu-network-core.md)
- [应用级私有出站 RFC](docs/RFC/0004-lcnd-continuity-b-contract.md)
- [开源工具评估](docs/OPEN_SOURCE_EVALUATION.md)
- [测试手册](docs/TEST_HANDBOOK.md)
- [Backlog](docs/BACKLOG.md)
- [版本规则](docs/VERSIONING.md)
- [更新记录](CHANGELOG.md)

## 开源与许可证

CND 自有代码采用 Apache License 2.0。项目不把 osquery 二进制提交到仓库；引导脚本从其 GitHub Release 下载固定版本并验证 SHA-256。第三方选择与许可说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
