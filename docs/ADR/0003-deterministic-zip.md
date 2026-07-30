# ADR 0003: Deterministic ZIP packaging

状态：Accepted
日期：2026-07-30

ZIP Builder 忽略文件系统 mtime、创建顺序和本机权限差异，使用排序条目、固定时间与固定模式。这样可对相同暂存树进行字节级重现和测试。

确定性只覆盖封装，不声称动态网络环境在两个时间点产生相同证据。
