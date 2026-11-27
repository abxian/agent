# 神仙监控 Agent

这是神仙监控的 Agent 仓库，基于哪吒监控 Agent fork 改名。

## 本次改动

- Go module 改为 `github.com/shenxianhq/agent`。
- Agent 二进制名为 `shenxian-agent`。
- CLI 使用说明和系统服务描述改为“神仙监控 Agent”。
- Proto 服务名从 `NezhaService` 改为 `ShenxianService`。
- Proto 文件从 `proto/nezha.*` 改为 `proto/shenxian.*`。
- 日志前缀改为 `SHENXIAN@...`。
- 自更新默认 GitHub 仓库改为 `shenxianhq/agent`。
- Agent 安装脚本和二进制使用神仙 `SX_` 环境变量，例如 `SX_SERVER`、`SX_CLIENT_SECRET`。

## 公开下载

公开软件放在 dufs：

```text
http://114.80.36.225:15667/sxjc/releases/agent/
```

已上传：

- `shenxian-agent-linux-amd64`
- `shenxian-agent-linux-arm64`
- `shenxian-agent-darwin-amd64`
- `shenxian-agent-darwin-arm64`
- `shenxian-agent-windows-amd64.exe`

## 安装

Linux/macOS：

```sh
curl -L http://114.80.36.225:15667/sxjc/agent-install.sh -o agent.sh && chmod +x agent.sh && env SX_SERVER=dashboard.example.com:8008 SX_TLS=false SX_CLIENT_SECRET=EXAMPLE SX_UUID=your_server_uuid ./agent.sh
```

Windows PowerShell：

```powershell
$env:SX_SERVER="dashboard.example.com:8008";$env:SX_TLS="false";$env:SX_CLIENT_SECRET="EXAMPLE";$env:SX_UUID="your_server_uuid";Invoke-WebRequest http://114.80.36.225:15667/sxjc/agent-install.ps1 -OutFile C:\install.ps1;powershell.exe C:\install.ps1
```

安装逻辑参考哪吒官方 Agent 文档：https://nezha.wiki/guide/agent.html

神仙监控安装文档见 Dashboard 仓库的 `docs/INSTALL.md`。

## 面板域名自动切换

在支持此功能的 Agent 本地配置中保留现有 UUID、密钥和 `server`，并加入
`server_discovery_url`（或环境变量 `SX_SERVER_DISCOVERY_URL`），也可加入
`server_discovery_fallback_url`（`SX_SERVER_DISCOVERY_FALLBACK_URL`）。启用时面板需要
有效的 TLS 证书，Agent 配置为 `tls: true`、`insecure_tls: false`。

```yaml
server: panel-old.example.net:5443
tls: true
insecure_tls: false
server_discovery_url: https://discovery.example.org/v1/shenxian-panel.json
server_discovery_fallback_url: https://raw.githubusercontent.com/OWNER/REPO/main/manifests/shenxian-panel.json
```

将 JSON 清单放在与面板域名独立、由管理员控制写权限的 HTTPS 静态托管入口。
清单只发布可直接访问的 **Agent gRPC** 域名和端口，无需包含密钥或 UUID。
使用不跳转的直达链接，并建议由托管端返回 `Cache-Control: no-store`。

```json
{"server":"panel-new.example.net:5443","revision":1795740000000000}
```

两源并行读取，任一源可用即可发现。两份内容更新不同步时，优先使用较大的 `revision`，
同版本优先配置中的首选源。旧单源清单仍可省略该字段，两个镜像应发布字节一致的清单。
需要回滚时，把旧地址作为更高的版本重新发布。

先准备新域名的 DNS、有效证书和 gRPC 反代，确认它指向使用相同设备身份和认证数据的面板，
再更新清单。Agent 启动时读取清单，运行中约每分钟刷新一次。新地址须通过 TLS 和
认证的系统信息上报才会接替现有连接。读取清单或验证新地址失败时，在线连接继续使用原地址。
断线重连会尝试清单地址、本进程上次连通的地址和本地配置地址。进程重启后会重新读取清单，
因此长期迁移稳定后宜把本地 `server` 更新到仍受控的后备地址。

已离线且仍运行旧版的 Agent 需要通过独立管理入口升级并配置一次，或先通过 DNS 恢复
旧入口。关闭功能时移除两项发现源设置或对应环境变量，然后重启 Agent。

## 编译验证

已验证：

```sh
go test -run '^$' ./...
go build -trimpath -buildvcs=false -ldflags="-s -w -X github.com/shenxianhq/agent/pkg/monitor.Version=shenxian-agent -X main.arch=amd64" -o dist/shenxian-agent.exe ./cmd/agent
```

完整测试中的 Cloudflare 探测用例依赖外部网站响应，可能因网络环境失败。
隔离分支构建可在 GitHub Actions 手动运行 `Gray candidate verification`，只生成临时工件。
