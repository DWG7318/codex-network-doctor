# Test Handbook

## Local quality gate

```powershell
$go = ".\.tools\go\bin\go.exe" # 或 PATH 中的 go

& $go test ./...
& $go test -race ./...
& $go vet ./...
& $go build -trimpath -o .\bin\cnd.exe .\cmd\cnd
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build.ps1 -SkipRace
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-linux.ps1
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
- 网络 node、target set 和 route profile 必须严格拒绝未知字段、嵌入式凭据和方向不匹配。
- 两个批准的路由 Profile 必须分别绑定 `china_egress` 与 `global_egress`，并产生确定性计划。
- 管理目标必须直连；与隧道目标发生域名、协议、端口或 CIDR 重叠时规划失败。
- sing-box candidate 允许包含受保护材料，但必须以新文件创建；最终 plan、摘要、回执、报告和错误不得泄漏材料。
- `simulate` 成功只产生 `VERIFIED`，验证失败或超时只产生 `ROLLED_BACK`。
- 所有网络报告必须使用 `evidence_mode=synthetic`，状态只能是 `SIMULATED_PASS` 或 `SIMULATED_ROLLBACK`。
- `cnd-network` 不得包含或执行 sudo、systemd、路由写入、服务安装和任意生产命令。

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

## Ubuntu network-core synthetic smoke test

先交叉构建：

```powershell
.\scripts\build-linux.ps1
```

再使用 `samples/network/*.synthetic.json` 依次运行 `plan`、`render`、`simulate` 和 `report`。
临时节点材料只能使用文档地址与明显标注的测试字符串，并放在受控临时目录。验收条件：

- 草案、最终计划、回执和报告均可解析为 JSON。
- 相同输入产生字节一致的草案和摘要。
- candidate 含测试材料，但 stdout、stderr 和四个公共产物不含测试材料。
- report 为 `evidence_mode=synthetic`；不得出现真实服务器健康声明。
- 测试后逐文件清理 candidate 和临时材料。

## Tamper test

1. 解压有效 ZIP。
2. 修改任一 `evidence/` 文件，但不更新 Manifest。
3. 重新压缩并运行 `cnd validate`。
4. 验收：退出码为 2，报告包含 `HASH_MISMATCH` 或结构问题。

## Race and static checks

发布构建必须保存 `go test -race ./...`、`go vet ./...`、Windows 五个可执行文件构建和
Ubuntu amd64 `cnd-network` 交叉构建的命令输出。安装了 `golangci-lint` 时执行：

```powershell
golangci-lint run
```

不得因为本机未安装 linter 而在运行时自动下载未经固定的二进制。
