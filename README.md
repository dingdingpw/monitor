# Monitor Party

Monitor Party 是一个轻量级 VPS / 服务器监控面板，由中心端 `vps-server` 和采集端 `vps-agent` 组成。中心端提供公开监控面板、管理员后台、节点管理、Agent 安装命令生成、Agent 二进制下载和 WebSocket 数据推送；Agent 负责采集并上报 CPU、内存、硬盘、网络、负载、连接数和在线状态。

默认示例域名：`https://www.monitor.party`

## 严重升级提示

本次更新缩减了发布文件体积，清理了历史残留的前端构建文件，并重新生成了内嵌前端和 release 二进制。更新过程中如果替换、卸载或导入操作不当，可能会造成节点、套餐信息、站点设置或 token 数据丢失。升级前请务必备份中心端数据文件：

```bash
sudo cp /var/lib/vps-monitor/server.json /var/lib/vps-monitor/server.json.bak.$(date +%Y%m%d%H%M%S)
```

如果已进入后台，也建议先使用「节点管理」里的「一键导出」导出节点备份 JSON，再执行升级。

## 本次更新

- Agent 新增系统/负载采集字段：真实主机名、发行版、内核、CPU 架构、虚拟化、CPU 型号、物理/逻辑核心、磁盘读写速率和进程数。
- Server 将新增 Agent 字段透传到 WebSocket 的 `Host` / `State` 数据中，前台展开节点详情即可查看。
- 前台详情页补充系统、内核、CPU、磁盘读写、进程、TCP / UDP、运行时长、数据更新时间等展示。
- 后台节点管理新增「一键导出 / 一键导入」，导出 JSON 备份，导入时合并节点和套餐信息，不主动删除现有节点。
- Release 构建脚本现在会自动构建并同步前端，再编译中心端，避免漏嵌前端资源。
- 清理历史残留的前端 hash 产物，`vps-server` 发布文件体积明显缩小。

## 功能

- 一体化中心端：公开面板、管理后台、API、WebSocket、Agent 下载均由 `vps-server` 提供。
- 多平台 Agent：Linux `amd64 / arm64 / armv7 / 386`，Windows `amd64 / arm64 / 386`。
- 实时监控：CPU、内存、Swap、硬盘、网络速率、累计流量、负载、运行时间和在线状态。
- 节点管理：预创建节点、生成安装命令、删除节点、编辑套餐/购买信息。
- 节点级 token：每个节点独立 Agent token，中心端不再使用全局 `AGENT_TOKEN`。
- 安全加固：拒绝默认弱密钥、限制 Node ID 字符集、配置文件权限收紧、Agent 默认要求 HTTPS。

## 架构

```text
vps-server
  - 公开监控面板 /
  - 管理后台 /admin
  - Agent 上报接口 /api/agent/report
  - WebSocket /ws
  - Agent 安装脚本 /install/*
  - Agent 二进制下载 /download/*

vps-agent
  - 读取本机配置
  - 定时采集系统指标
  - 使用节点专属 token 上报到中心端
```

## 目录

```text
cmd/vps-server/                 中心端入口
cmd/vps-agent/                  Agent 入口
internal/server/                中心端逻辑、后台页面、静态资源、内嵌 Agent 二进制
internal/server/web/dist/       中心端内嵌前端构建产物
internal/server/agent_bins/     中心端内嵌 Agent 下载文件
internal/agent/                 Agent 指标采集
internal/config/                Agent 配置解析
internal/reporter/              Agent 上报逻辑
scripts/                        构建、安装、卸载脚本源文件
release/                        发布用二进制和安装脚本
web/                            Vue 前端源码
data/                           本地开发数据示例
```

## 快速安装中心端

下载 GitHub Release 中适合服务器架构的中心端二进制和安装脚本，例如 Linux amd64：

```text
vps-server-linux-amd64
install-server-linux.sh
uninstall-server-linux.sh
```

放到同一目录后执行：

```bash
chmod +x install-server-linux.sh vps-server-linux-amd64
sudo ./install-server-linux.sh
```

脚本会询问：

```text
Public URL [https://www.monitor.party]
Internal secret (leave empty to generate)
Admin username [admin]
Admin password (leave empty to generate)
Listen address [:3000]
Max nodes [2000]
Binary download URL (empty for local file)
```

说明：

- `Public URL`：公网访问地址，生产环境必须使用 HTTPS，例如 `https://monitor.example.com`。
- `Internal secret`：中心端内部安全密钥，不是后台登录密码；留空会自动生成。
- `Admin username`：后台管理员用户名，默认 `admin`。
- `Admin password`：后台管理员登录密码，和 `Internal secret` 不需要一致；留空会自动生成。
- `Listen address`：中心端监听地址，默认 `:3000`。
- `Max nodes`：最大节点数，默认 `2000`。
- `Binary download URL`：留空时使用当前目录的本地二进制。

中心端现在不需要填写 `Agent token`。Agent token 会在后台为每个节点单独生成。

`Internal secret` 和 `Admin password` 不是同一个东西，不需要一致。前者用于中心端内部安全用途，后者只用于 `/admin` 后台登录。

安装完成后访问：

```text
https://你的域名/admin
```

## Nginx 反向代理

生产环境建议使用 HTTPS 反向代理到本机 `3000` 端口：

```nginx
server {
    listen 80;
    server_name monitor.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name monitor.example.com;

    ssl_certificate /etc/letsencrypt/live/monitor.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitor.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 添加 Agent 节点

推荐只通过后台生成安装命令：

1. 打开 `https://你的域名/admin`。
2. 使用中心端安装时设置的管理员账号密码登录。
3. 在 `添加节点` 中输入 Node ID，例如 `US-node-001`。
4. 点击 `添加并生成`。
5. 复制后台生成的 Linux 或 Windows 安装命令到目标服务器执行。

每次生成命令都会为该节点签发新的独立 token。服务端只保存 token hash，不保存明文 token。

Node ID 规则：

```text
支持中文、英文、数字和常见分隔符
长度 1-96
不能包含换行、引号、斜杠、反斜杠、Shell 控制符或 HTML 尖括号
```

建议使用地区前缀：

```text
US-node-001
JP-node-001
HK-node-001
DE-node-001
CN-上海-腾讯云
```

## Linux Agent

后台生成的命令示例：

```bash
curl -fsSL https://monitor.example.com/install/agent-linux.sh | sudo sh -s -- --server https://monitor.example.com --token NODE_TOKEN --node-id US-node-001
```

安装后会创建：

```text
/usr/local/bin/vps-agent
/etc/vps-agent/config.env
/etc/systemd/system/vps-agent.service
```

常用命令：

```bash
sudo systemctl status vps-agent
sudo systemctl restart vps-agent
sudo journalctl -u vps-agent -f
```

卸载：

```bash
curl -fsSL https://monitor.example.com/uninstall/agent-linux.sh | sudo sh
```

## Windows Agent

后台生成的命令需要在管理员 PowerShell 中执行，示例：

```powershell
powershell -ExecutionPolicy Bypass -Command "iwr https://monitor.example.com/install/agent-windows.ps1 -UseBasicParsing | iex; Install-VpsAgent -Server 'https://monitor.example.com' -Token 'NODE_TOKEN' -NodeId 'US-win-001'"
```

安装后会创建：

```text
C:\Program Files\vps-agent\vps-agent.exe
C:\ProgramData\vps-agent\config.env
Windows 服务：vps-agent
```

常用命令：

```powershell
Get-Service vps-agent
Restart-Service vps-agent
Stop-Service vps-agent
Start-Service vps-agent
```

卸载：

```powershell
powershell -ExecutionPolicy Bypass -Command "iwr https://monitor.example.com/uninstall/agent-windows.ps1 -UseBasicParsing | iex"
```

## 配置文件

中心端配置：

```text
/etc/vps-monitor/server.env
```

示例：

```env
ADDR=:3000
AUTH_SECRET=replace-with-strong-random-secret
ADMIN_USER=admin
ADMIN_PASS=replace-with-strong-random-password
PUBLIC_URL=https://monitor.example.com
DATA_PATH=/var/lib/vps-monitor/server.json
MAX_NODES=2000
```

修改后重启：

```bash
sudo systemctl restart vps-server
```

Agent 配置：

```text
Linux:   /etc/vps-agent/config.env
Windows: C:\ProgramData\vps-agent\config.env
```

示例：

```env
SERVER=https://monitor.example.com
TOKEN=node-specific-token
NODE_ID=US-node-001
BASIC_INTERVAL=2s
DISK_INTERVAL=30s
CONNECTION_INTERVAL=60s
MOUNTS=auto
NETWORK_EXCLUDE=lo,docker*,veth*,br-*
DISK_EXCLUDE_FS=tmpfs,devtmpfs,overlay,squashfs,proc,sysfs,cgroup,cgroup2
```

## 数据文件

中心端默认数据文件：

```text
/var/lib/vps-monitor/server.json
```

其中包含节点列表、上报数据、购买信息、站点设置和节点 token hash。建议升级前备份：

```bash
sudo cp /var/lib/vps-monitor/server.json /var/lib/vps-monitor/server.json.bak.$(date +%Y%m%d%H%M%S)
```

## 升级中心端

替换二进制并重启即可，数据文件不会自动删除：

```bash
sudo systemctl stop vps-server
sudo cp /usr/local/bin/vps-server /usr/local/bin/vps-server.bak.$(date +%Y%m%d%H%M%S)
sudo install -m 0755 ./vps-server-linux-amd64 /usr/local/bin/vps-server
sudo systemctl start vps-server
sudo journalctl -u vps-server -n 100 --no-pager
```

如果从旧的全局 `AGENT_TOKEN` 版本升级到节点级 token 版本，旧 Agent 需要重新在后台生成命令并重装，否则无法通过新鉴权。

## 卸载

卸载中心端：

```bash
sudo ./uninstall-server-linux.sh
```

彻底清空中心端配置和数据：

```bash
sudo rm -rf /etc/vps-monitor /var/lib/vps-monitor
```

注意：删除 `/var/lib/vps-monitor` 会清空节点、购买信息、站点设置和节点 token，所有 Agent 都需要重新添加并安装。

卸载 Linux Agent：

```bash
sudo ./uninstall-agent-linux.sh
```

卸载 Windows Agent：

```powershell
.\uninstall-agent-windows.ps1
```

## 本地开发

运行中心端：

```bash
AUTH_SECRET=replace-with-strong-random-secret \
ADMIN_USER=admin \
ADMIN_PASS=replace-with-strong-random-password \
PUBLIC_URL=http://127.0.0.1:3000 \
DATA_PATH=./data/server.json \
go run ./cmd/vps-server
```

运行 Agent：

```bash
SERVER=http://127.0.0.1:3000 \
TOKEN=node-specific-token \
NODE_ID=local-node-001 \
go run ./cmd/vps-agent run --config /path/to/config.env
```

说明：Agent 默认要求 HTTPS，只有 `localhost` 和 `127.0.0.1` 允许 HTTP。

## 构建

构建前端：

```bash
cd web
npm install
npm run build
cd ..
```

构建 release：

```powershell
powershell -ExecutionPolicy Bypass -File "scripts\build-release.ps1"
```

Linux / macOS：

```bash
sh scripts/build-release.sh
```

产物会写入：

```text
release/
```

主要产物：

```text
vps-server-linux-amd64
vps-server-linux-arm64
vps-server-linux-armv7
vps-server-linux-386
vps-server-windows-amd64.exe
vps-server-windows-arm64.exe
vps-server-windows-386.exe
vps-agent-linux-amd64
vps-agent-linux-arm64
vps-agent-linux-armv7
vps-agent-linux-386
vps-agent-windows-amd64.exe
vps-agent-windows-arm64.exe
vps-agent-windows-386.exe
install-server-linux.sh
install-agent-linux.sh
install-agent-windows.ps1
uninstall-server-linux.sh
uninstall-agent-linux.sh
uninstall-agent-windows.ps1
```

## 常见问题

### 中心端启动失败

检查 `AUTH_SECRET` 和 `ADMIN_PASS` 是否为空、是否仍是 `change-me`。新版本只拒绝空值和默认弱值。

```bash
sudo journalctl -u vps-server -n 100 --no-pager
```

### 后台无法登录

检查 `/etc/vps-monitor/server.env`：

```env
ADMIN_USER=admin
ADMIN_PASS=replace-with-strong-random-password
```

修改后重启：

```bash
sudo systemctl restart vps-server
```

### Agent 一直离线

检查：

```text
SERVER 是否是正确 HTTPS 地址
TOKEN 是否来自后台为该节点生成的安装命令
NODE_ID 是否和后台节点一致
中心端防火墙是否放行 443 或 3000
反向代理 /ws 是否支持 Upgrade
```

查看日志：

```bash
sudo journalctl -u vps-agent -f
sudo journalctl -u vps-server -f
```

### 前台 WebSocket 不刷新

确认 Nginx `/ws` 配置包含：

```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
```

### 重新生成节点 token

进入后台，点击节点对应的 `命令`，系统会为该节点生成新的 token 并更新服务端保存的 hash。旧 token 会失效，需要用新命令重装或更新 Agent 配置。

## 安全建议

- `AUTH_SECRET` 和 `ADMIN_PASS` 不需要一致；不要留空，不要使用 `change-me`。
- 不要把真实生产环境的 `server.env`、`config.env`、token、密码提交到 GitHub。
- 生产环境必须使用 HTTPS。
- 管理后台建议配合防火墙、Cloudflare Access 或 Nginx Basic Auth 做额外保护。
- 节点安装命令包含该节点明文 token，只在可信终端执行，不要公开粘贴。

## 友链

- [Linux.do](https://linux.do/)

## License
