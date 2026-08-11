# Changelog

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
