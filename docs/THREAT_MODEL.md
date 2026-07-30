# Threat Model

## Assets

- Evidence ZIP 与其中的主机观察值。
- Manifest、文件哈希和采集状态。
- 操作者选择的目录、Provider 与二进制路径。
- 运行 CND 的 Windows 用户权限。

## Trust boundaries

1. 操作者向 CND 提供参数和可执行文件路径。
2. CND 启动 osquery 或 PowerShell 子进程。
3. Provider 从 Windows API、CIM、注册表和证书存储读取状态。
4. 原始输出进入暂存 Evidence Store。
5. 暂存内容写入 ZIP 并交给外部存储或分析系统。

## Threats and controls

| Threat | Control | Residual risk |
|---|---|---|
| 路径穿越覆盖任意文件 | Store 拒绝绝对、卷、根和 `..` 路径；Validator 拒绝不安全 ZIP 条目 | 获得 CND 进程控制权的攻击者仍可绕过应用逻辑 |
| Collector 覆盖先前证据 | Store 使用排他创建，重复路径失败 | 文件系统过滤驱动仍可能在进程外修改暂存文件 |
| 单项失败被误认为完整 | Collector 状态进入 Manifest，整体标为 `partial` | `complete` 仅覆盖所选目录，不代表证据穷尽 |
| ZIP 内容被事后篡改 | 每个文件 SHA-256、大小检查；ZIP SHA-256 返回给操作者 | 未内建数字签名，攻击者可同时重写 ZIP 与 Manifest |
| ZIP Slip 或重复条目迷惑分析器 | Validator 检查安全路径和重复名字 | 第三方解压器若绕过 Validator 仍需自行防护 |
| 主动网络流量改变现场 | 默认目录只允许 passive；TLS/WebSocket 主动探测不实现 | 某些系统读取 API 自身可能留下系统审计痕迹 |
| PowerShell profile 或策略注入 | `-NoProfile -NonInteractive`，不使用 Bypass | 被替换的 PowerShell 二进制或系统模块仍不可信 |
| osquery 扩展执行第三方代码 | 禁用 extensions、events 和 logging | 指定的 `osqueryi.exe` 本身可能被替换 |
| 私钥或口令泄露 | 不导出私钥、命令行或环境变量 | 代理地址、PAC、进程路径和证书元数据仍可能敏感 |
| 资源耗尽 | 每个 Collector 有超时；Validator 单文件上限 256 MiB | 当前 ZIP Builder 对单文件采用内存读取 |

## Explicit non-guarantees

CND 不保证：

- 被调查主机没有 rootkit、API hooking 或 WMI/CIM 欺骗。
- 主机时钟、主机名、DNS 或路由数据真实。
- SHA-256 等价于独立可信时间戳或签名。
- 非管理员执行能够看到管理员可见的全部状态。
- Evidence ZIP 适合未经加密地通过电子邮件、聊天或公共工单传输。

## Deployment recommendations

- 固定 CND 与 osquery 的 SHA-256，并通过组织软件分发渠道提供。
- 在干净介质或受管终端启动。
- 把命令摘要中的 ZIP SHA-256写入独立案件系统。
- 使用组织 PKI 对 ZIP SHA-256 签名。
- 在受控位置设置保留、访问审计和销毁策略。
