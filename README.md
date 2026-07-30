# Codex Network Doctor

Codex Network Doctor 的 `cnd.exe` 是一个 Windows 网络证据采集器。它只采集和封装证据，不修改网络设置、不执行修复，也不输出诊断、根因、概率或 AI 推测。

项目另外提供显式运行、范围受限的 `cnd网络修复.exe`，用于修复 v2rayN 重复实例、
遗留内核或 Windows 排除端口范围造成的控制端口冲突，并确保 TUN 启用。两个程序职责独立。

当前交付是 MasterSpec 路线中的 **v4 Evidence Engine CLI**：

- osquery 5.23.1 适配器：13 个被动查询，适合结构化采集。
- Windows PowerShell 降级适配器：15 个只读采集器，不依赖第三方安装。
- 采集器级超时与失败隔离；单项失败不阻断证据包生成。
- 原始 stdout/stderr 留存。
- 每个文件的 SHA-256 与大小写入 `manifest.json`。
- 固定目录顺序、ZIP 时间戳和权限位，保证相同暂存内容得到相同 ZIP 字节。
- 独立的 `validate` 命令检查路径安全、重复条目、必需文件、大小和 SHA-256。

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
- [开源工具评估](docs/OPEN_SOURCE_EVALUATION.md)
- [测试手册](docs/TEST_HANDBOOK.md)
- [Backlog](docs/BACKLOG.md)
- [版本规则](docs/VERSIONING.md)
- [更新记录](CHANGELOG.md)

## 开源与许可证

CND 自有代码采用 Apache License 2.0。项目不把 osquery 二进制提交到仓库；引导脚本从其 GitHub Release 下载固定版本并验证 SHA-256。第三方选择与许可说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
