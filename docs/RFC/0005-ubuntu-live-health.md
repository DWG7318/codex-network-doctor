# RFC 0005：Ubuntu 24.04 真实网络检测与健康报告

状态：LCnd v0.5.0 实现合同；2026-10-27。Owner 授权 LCnd 独立只读检测，LCSP 对接延期。

## 入口与边界

新增独立入口 `cnd-network diagnose`。既有 `plan/render/simulate/report/continuity` 继续使用原 synthetic 合同；`cnd.exe` 被动采集、显式修复、撤销、Windows 测速均保持原边界。

此命令读取本机网络事实并发送显式目标的 HTTPS HEAD；不安装服务、不读取节点凭据、不启动代理、不发起 SSH，不修改 VPN、默认路由、DNS、hosts、防火墙、系统代理、Tailnet、LCSP 或其他服务。它不是带宽测速或逐跳 traceroute。

## 输入

`schemas/network-live-targets.schema.json` 为独立合同，JSON 最大 1 MiB，拒绝未知字段、尾随 JSON 和非法范围。

- 1–8 个目标；ID 唯一且只含字母、数字、短横线、下划线，最长64字符。
- HTTPS URL 最长2048字符；拒绝用户名/密码、query、fragment；端口1–65535。
- 操作者明确指定期待的 200–299 状态码。接口不支持 HEAD 时应选择安全的 HEAD 健康端点，不自动改成 GET/POST。
- 3–5 轮串行采样；每个目标/路径/轮次总超时100–10000ms。所有配置预算相加不超过120秒，运行时也有120秒硬期限。
- 必须有 `direct`；可显式加入已有 `lcnd-egress-cn`/`lcnd-egress-global` 私有代理。代理仅允许明确私网/loopback IP和端口，协议 `http`（HTTPS CONNECT）、`socks5`、`socks5h`。不接受账号、密码或秘密引用；需要鉴权的实际出口对接留待后续。
- 不自动发现、不继承 HTTP_PROXY/HTTPS_PROXY/ALL_PROXY、不切换线路。代理失败不会回退到 direct。

配置只应列出已获准检测的端点。此命令不扫描网段、不枚举端口。`example.invalid` 样例不会成为默认探测目标；必须显式提供输入文件。

## 真实证据

Linux 使用 `net.Interfaces` 和固定小文件 `/etc/os-release`、`/etc/resolv.conf`、`/proc/net/route`、`/proc/net/ipv6_route`。每个文件最大64KiB；不可读或过大记稳定错误码，不使用 shell、sudo 或自动修复。支持验收目标是 Ubuntu 24.04 amd64，其他 Linux 会标记平台未验证并返回非健康退出码；Windows 诊断入口返回 `LINUX_REQUIRED`。

每轮 direct：解析目标域名，再用本轮DNS结果尝试TCP（最多16个地址，剩余期限按剩余地址分配），验证TLS证书及主机名，发送HEAD并观察状态码。IP目标的DNS为 `not_applicable`。

每轮 proxy：TCP仅代表代理端点可连接；目标DNS交给代理，报告为 `not_observed/PROXY_RESOLVES_TARGET`，不伪造远端解析成功。TLS仍验证最终目标证书。禁止跟随重定向，响应头最大32KiB，不读取或保存响应正文、Cookie、授权头或完整URL。错误不输出原始系统/网络错误字符串。收到100–599之外的HTTP状态码时记录 `HTTP_INVALID_STATUS`，不把非法数字写入报告的 `http_status` 字段。

## 评分与状态

每个目标/路径分别报告成功HTTP次数、成功样本总耗时中位数、DNS/TCP/TLS/HTTP结果及检查成功率。

`score = floor(100 × passed_checks / required_checks)`：每个适用层权重相同；IP的DNS和代理不可观测DNS不进入分母。上游失败导致的下游 blocked 仍在分母，不计成功。完整失败是有效证据，而非样本缺失。

- `healthy`：配置的全部适用检查成功；仅表示所选路径与目标。
- `degraded`：部分层失败；不能直接断言故障根因。
- `unhealthy`：完整尝试中没有适用检查成功。
- `insufficient_evidence`：轮次缺失、取消、重复或不匹配；score=null。

总评分采用所有适用层汇总，不比较不可观测DNS，不把评分解释为速率。主机快照完整性独立记录；主机快照partial/不支持的平台即使目标通过，CLI仍返回3。没有默认路由不等于指定内网目标不可达；本版提供路线原始快照，不对policy routing、全部中间跳点或防火墙作穷尽结论。

`direct` 只表示应用不显式使用代理，实际流量仍遵循宿主现有路由/VPN/TUN。无修复、无生产健康或LCSP授权推断。

## 产物与退出码

必须指定新的 JSON 和中文文本输出路径。两个文件在观测前用 O_EXCL、0600 创建，并确认是0600普通文件；任一路径已存在或文件系统不落实私有权限则失败，保留旧文件，不执行网络探测。写入失败时清理仅本次创建的不完整输出。WSL输出需使用支持Linux权限的文件系统，不能假定Windows挂载盘会落实0600。报告含本机接口/地址/路由，按敏感私有证据保存，绝不进入公开 Git。

报告 `evidence_mode=live`、`scope=selected_targets_and_explicit_paths`，Schema独立于旧的 synthetic health report。文本提供状态/评分/路径摘要；详细主机事实在JSON内。控制台只返回状态和计数，稳定错误码不回显输入材料。

- 0：支持平台、完整主机观察且所选检查全部通过。
- 1：输入文件读取或输出文件操作失败。
- 2：参数/输入合同无效或非Linux平台。
- 3：已写报告，但检测退化、失败、证据不足、主机partial或Linux平台未经验证。

## 验收与剩余范围

回归测试覆盖证书不可信、HTTP状态不符、重定向、超时、取消、私有代理、代理失败不回退、输出保护与脱敏。Linux测试覆盖真实接口读取和受限文件错误。真人工验收可在本机Ubuntu执行，结果保留私有。

不包含：LCSP托管、真实节点导入/鉴权秘密注入、LCapi业务接入、生产部署、长期监测、自动修复、逐跳定位或吞吐测速；以上仍需单独实施。
