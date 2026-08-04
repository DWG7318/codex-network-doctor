# Test Handbook

## Local quality gate

```powershell
$go = ".\.tools\go\bin\go.exe" # 或 PATH 中的 go

& $go test ./...
& $go test -race ./...
& $go vet ./...
& $go build -trimpath -o .\bin\cnd.exe .\cmd\cnd
```

## Required behavior tests

- Evidence 路径拒绝绝对路径、卷路径和 `..`。
- Evidence 文件不可覆盖。
- Collector 失败与超时不阻断后续 Collector。
- 原始 stdout/stderr 在失败时仍保留。
- 相同暂存内容以不同创建顺序和时间写入时产生相同 ZIP。
- Validator 能发现 SHA-256 不符。
- osquery 目录只能包含 `SELECT` 且为 passive。
- Windows DNS 查询不得引用 Windows 构建不存在的 `dns_resolvers` 表。
- PowerShell 目录不得含主动或变更命令。
- 证书采集不得依赖 `Cert:` PSDrive，必须只读打开 X509Store。
- 引导脚本固定 osquery 版本和 SHA-256，且不得安装服务。
- 分层测速必须按系统/TUN、HTTP、SOCKS5 串行配对，不得并发压测。
- 比较性判断必须至少使用每条路径三个成功样本；样本不足输出 `INSUFFICIENT_EVIDENCE`。
- 三次配对中某入口失败至少两次、SOCKS5 三次成功时，必须输出对应 `*_PATH_UNRELIABLE`，
  不得把已完成的三次失败证据误报为样本不足。
- SOCKS5 测试必须把目标域名交给代理解析，且不得读取或输出节点凭据。
- `cnd网速检测.exe` 默认进入只读测速模式，不得因文件名解析错误进入修复模式或触发 UAC。

## Windows smoke tests

### osquery

```powershell
.\scripts\bootstrap-osquery.ps1
.\bin\cnd.exe collect `
  --provider osquery `
  --osquery ".\.tools\osquery-5.23.1.windows_x86_64\Program Files\osquery\osqueryi.exe" `
  --output .\out\osquery-smoke.zip
.\bin\cnd.exe validate .\out\osquery-smoke.zip
```

验收：13 个 Collector 成功，0 失败，Validator `valid=true`。

### PowerShell

```powershell
.\bin\cnd.exe collect `
  --provider powershell `
  --powershell "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe" `
  --output .\out\powershell-smoke.zip
.\bin\cnd.exe validate .\out\powershell-smoke.zip
```

验收：当前支持的 Windows 环境中 15 个 Collector 成功。旧版 Windows 缺少 Cmdlet 时允许生成 `partial` 包，但失败项必须有错误和 stderr 证据。

## Tamper test

1. 解压有效 ZIP。
2. 修改任一 `evidence/` 文件，但不更新 Manifest。
3. 重新压缩并运行 `cnd validate`。
4. 验收：退出码为 2，报告包含 `HASH_MISMATCH` 或结构问题。

## Race and static checks

发布构建必须保存 `go test -race ./...`、`go vet ./...` 和编译命令输出。安装了 `golangci-lint` 时执行：

```powershell
golangci-lint run
```

不得因为本机未安装 linter 而在运行时自动下载未经固定的二进制。
