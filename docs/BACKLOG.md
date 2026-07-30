# Issue-ready Backlog

工作区最初没有 Git 仓库或远程仓库，因此没有擅自创建外部 GitHub Issues。以下条目可在建立远程仓库后按顺序转为 Issues。

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
