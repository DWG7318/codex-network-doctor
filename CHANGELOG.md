# Changelog

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
