# Changelog

## v0.4.0 — 2026-08-13

- 新增应用级私有出站服务计划、secret reference、分层 probe、保护目标、rollback receipt 与 LCapi application attachment 严格合同。
- 固定 `direct`、`lcnd-egress-cn`、`lcnd-egress-global` 三类稳定线路身份，国内与海外方向共用同一模型。
- 强制新服务仅监听 loopback 或带批准引用的私网，拒绝公网监听、匿名访问、宿主默认路由、全局 DNS/hosts、防火墙、系统代理、Tailnet 和整机 TUN 变化。
- 新增 `continuity validate/candidate-summary/probe-plan/evaluate/guard-check/rollback-evaluate/attachment-render` 离线命令；所有生成物确定、脱敏且不可覆盖。
- 新增 direct/egress 分层 synthetic evaluator；样本不足或缺层输出 `INSUFFICIENT_EVIDENCE`，不声明真实 Ubuntu、节点或生产健康。
- 将旧公网 TCP/443/HAProxy→Caddy/Xray/客户端线路、SSH、LCSP Access、LCgpu 与系统网络设置列为永久保护目标。
- 保持 Windows 被动采集、显式修复、撤销和只读测速边界不变；未修改 LCSP、LCapi、LCgpu，未连接服务器、导入真实节点或部署生产。
- 仓库实现完成不等于 LCSP 接入完成，不等于 LCapi 接入完成，也不等于生产验收完成。

## v0.3.0 — 2026-08-12

- 新增独立的 simulation-only `cnd-network`，面向 Ubuntu 24.04 amd64 的双向网络规划。
- 新增严格 node、target set、route profile、change plan、execution receipt 和 health report 合同与合成样例。
- 新增确定性路由规划，支持海外主机访问国内目标和国内主机访问国外目标，并保护管理直连路径。
- 新增 sing-box 1.13.12 candidate 渲染；节点材料只进入权限受限 candidate，最终计划与摘要保持脱敏。
- 新增 success、verification failure 和 timeout 合成状态机；所有报告固定为 `evidence_mode=synthetic`。
- Windows 构建保留原有四个可执行文件并增加 `cnd-network.exe`；新增 Ubuntu amd64 无副作用交叉构建。
- 保持 `cnd.exe` 被动采集、Windows 修复和只读网速检测合同不变。
- LCSP 集成、真实节点、sudo/systemd 代办、真实链路探测与生产路由执行继续延期。

## v0.2.0 — 2026-08-04

- 新增可双击的只读 `cnd网速检测.exe` 和 `cnd网络修复.exe speed`。
- 串行对比系统/TUN、HTTP代理和 SOCKS5代理路径，记录完整响应吞吐、TTFB和失败。
- 新增基于三轮配对样本的有限分层判断，识别 TUN入口、HTTP入口或共享远端链路异常。
- 主动测速不请求管理员权限，不修改代理、TUN、DNS、路由或进程状态。
- 保持 `cnd.exe` Evidence ZIP、被动 Collector 和验证合同不变。

## v0.1.0 — 2026-07-30

- 发布 Windows 离线网络证据采集器 `cnd.exe`。
- 发布显式授权的 v2rayN TUN 修复与撤销程序。
- 修复 Windows excluded port range 导致的 sing-box 控制端口 WSAEACCES/10013。
- 加入编译进 EXE 的脱敏案例库和离线查询命令。
- 加入合成证据包、确定性 ZIP、完整性校验和隐私边界文档。
