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

## 编译验证

已验证：

```sh
go test -run '^$' ./...
go build -o dist/shenxian-agent.exe ./cmd/agent
```

完整测试中的 Cloudflare 探测用例依赖外部网站响应，可能因网络环境失败。
