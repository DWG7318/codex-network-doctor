# Ubuntu 24.04 网络健康检测

LCnd v0.5.0 新增 `cnd-network diagnose`。它读取网络现状，并向你明确指定的 HTTPS 端点发送少量 HEAD 请求。无需 root，不改网络、VPN或服务。

## 使用

将 Linux amd64 版 `cnd-network` 放到本机任意私有工作目录，准备 `targets.json`：

```json
{
  "schema_version": "1.0.0",
  "samples": 3,
  "timeout_ms": 3000,
  "targets": [
    {"id": "my-service", "url": "https://example.invalid/health", "expected_status": 200}
  ],
  "routes": [{"id": "direct"}]
}
```

将 `example.invalid` 换成你获准检测、支持 HEAD 的真实健康端点。不要填写API密钥、订阅URL、账号密码、query参数或带凭据的路径。

```bash
./cnd-network diagnose --input targets.json --output health.json --text-output health.txt
```

两个输出文件必须尚不存在。报告是敏感资料，只留在本地私有目录；创建后检查Linux输出确实为0600，否则在采集前报错。WSL请把报告放在Linux家目录等原生文件系统，不要放到未启用Linux权限的 `/mnt/c`、`/mnt/d` 等Windows挂载盘。不要提交到公开仓库。

## 对比已有代理

如果本机已存在批准使用的 HTTP/SOCKS5 私有代理，可在 `routes` 中增加一项，例如：

```json
{"id":"lcnd-egress-cn","proxy_url":"socks5h://127.0.0.1:18080"}
```

这只是用已有端点做检测，不会启动代理、导入节点、打开TUN或修改系统代理。必须使用私网/loopback IP和明确端口；当前不接受代理账号密码。未配置代理时只测direct，不读取环境代理。代理失败不会切换到其他路径。

## 看懂结果

- DNS、TCP、TLS、HTTP分层记录结果和稳定错误码。HTTPS必须通过系统证书验证。
- `direct`仍遵循现有系统路由，可能经过既有VPN/TUN。
- 代理路径的TCP仅证明代理入口可连接；目标DNS是代理端处理，报告标记不可观测。
- 评分是本次指定路径的分层成功率，不是网速，也不是整台服务器的总体健康保证。
- 有完整失败样本会给出失败评分；样本不全则评分为空，不伪装为健康。
- 401/403等响应可以说明网络已到达HTTP层，但不符合期待的健康状态；要结合各层观察判断，不直接认定为网络故障。
- 文本列出各条路径HTTP成功次数、成功样本耗时中位数与逐轮状态；详细接口、DNS和路由快照在JSON中。

退出码0表示本次指定检查通过；3表示报告已生成，但有失败、证据不足、主机信息不全或平台未经验证；1表示文件错误；2表示参数无效或非Linux。

当前验证平台是Ubuntu 24.04 amd64。本版不提供带宽测速、每一跳的根因判断或自动修复。LCSP服务托管、LCapi接入和生产试点后续单独完成。
