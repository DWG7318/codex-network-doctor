# ADR 0002: Passive-only default catalog

状态：Accepted
日期：2026-07-30

默认目录不得发起网络连接、下载 PAC、发送 DNS 请求、启动抓包会话或修改系统配置。主动 Collector 必须在未来使用独立配置、显式操作者同意、独立证据分类和清楚的现场影响说明。

原因是“读取远端状态”并不等于对本地主机只读；TLS、HTTP 和 WebSocket 探测会在远端与中间设备留下状态，也可能触发业务行为。
