# Monitor Party

Monitor Party 是一个轻量级 VPS / 服务器监控面板，包含中心端 `vps-server` 和采集端 `vps-agent`。中心端提供公开监控面板、管理员后台、Agent 安装命令生成、节点管理、购买信息展示和 Agent 二进制下载；Agent 负责上报 CPU、内存、硬盘、网络、负载和在线状态。

默认演示地址为：`https://www.monitor.party`

## 功能特性

- 中心端一体化：Web 面板、管理后台、API、WebSocket、Agent 下载都由 `vps-server` 提供。
- 多平台 Agent：支持 Linux `amd64 / arm64 / armv7 / 386` 和 Windows `amd64 / arm64 / 386`。
- 节点状态监控：CPU、内存、硬盘、网络速率、流量、负载、启动时间、上报时间。
- 后台节点管理：预创建节点、生成安装命令、删除节点、编辑购买信息和到期时间。
- 购买信息：支持卖家、价格、周期、月流量、到期时间、购买链接，周期包含 `日 / 月 / 半年 / 年 / 三年 / 五年 / 十年`。
- 管理后台：黑白灰 Cloudflare 风格控制台，避免花哨配色，重点突出节点管理和安装命令。
- 免输入安装 / 卸载：后台可为指定节点生成 Linux / Windows 一键安装命令，也提供 Agent 一键卸载命令。

## 目录说明

```text
cmd/vps-server/                 中心端入口
cmd/vps-agent/                  Agent 入口
internal/server/                中心端逻辑、内嵌后台、内嵌静态资源
internal/server/web/dist/       中心端内嵌前端构建产物
internal/server/agent_bins/     中心端内嵌 Agent 下载二进制
scripts/                        构建、安装、卸载脚本源文件
release/                        发布用二进制和安装脚本
web/                            Vue 前端源码
data/                           本地开发数据示例
```

## 端口和域名

- 默认监听：`:3000`
- 默认公开地址：`https://www.monitor.party`
- 公开面板：`https://www.monitor.party/`
- 管理后台：`https://www.monitor.party/admin`
- WebSocket：`wss://www.monitor.party/ws`
- Agent 上报接口：`https://www.monitor.party/report`

如果你使用自己的域名，把教程中的 `https://www.monitor.party` 替换成你的域名即可。

## 快速开始

### 1. 下载 Release

从 GitHub Releases 下载对应系统的文件。Linux 中心端常用文件如下：

```text
vps-server-linux-amd64
install-server-linux.sh
uninstall-server-linux.sh
```

Linux Agent 常用文件如下：

```text
vps-agent-linux-amd64
install-agent-linux.sh
uninstall-agent-linux.sh
```

Windows Agent 常用文件如下：

```text
vps-agent-windows-amd64.exe
install-agent-windows.ps1
uninstall-agent-windows.ps1
```

### 2. 安装中心端

在中心端服务器上放入 `vps-server-linux-amd64` 和 `install-server-linux.sh`，然后执行：

```bash
chmod +x install-server-linux.sh vps-server-linux-amd64
sudo ./install-server-linux.sh
```

脚本会依次询问：

```text
Public URL [https://www.monitor.party]
Agent token
Legacy auth secret
Admin username [admin]
Admin password
Listen address [:3000]
Max nodes [2000]
Binary download URL (empty for local file)
```

推荐填写：

```text
Public URL: https://www.monitor.party
Agent token: 自己生成一个强随机 token
Legacy auth secret: 自己生成一个强随机密钥
Admin username: admin 或自定义用户名
Admin password: 后台登录密码
Listen address: :3000
Max nodes: 2000
Binary download URL: 留空
```

安装完成后访问：

```text
https://www.monitor.party/admin
```

如果没有配置反向代理和 HTTPS，可以先用服务器 IP 测试：

```text
http://服务器IP:3000/admin
```

### 3. 配置反向代理

如果你使用 `https://www.monitor.party`，需要把域名解析到中心端服务器，并将 HTTPS 反代到本机 `3000` 端口。

Nginx 示例：

```nginx
server {
    listen 80;
    server_name www.monitor.party;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name www.monitor.party;

    ssl_certificate /etc/letsencrypt/live/www.monitor.party/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/www.monitor.party/privkey.pem;

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

## 中心端管理

### 登录后台

打开：

```text
https://www.monitor.party/admin
```

使用安装中心端时填写的 `Admin username` 和 `Admin password` 登录。

后台界面采用黑白灰控制台风格，左侧为深色导航，主区域为白色卡片，用于集中管理节点、站点设置、购买信息、带宽、到期时间和安装命令。

### 添加节点

1. 进入后台 `节点管理`。
2. 在 `添加节点` 中输入 Node ID，例如 `US-node-001`。
3. 点击 `添加并生成`。
4. 后台会生成 Linux 和 Windows 的免输入安装命令。
5. 复制命令到 Agent 服务器执行。

Node ID 建议使用国家或地区前缀，例如：

```text
US-node-001
JP-node-001
HK-node-001
DE-node-001
```

前两位会用于前台地区分类和旗帜显示。

### 编辑购买信息

在节点列表点击 `编辑`，可填写：

```text
卖家
价格
周期：日 / 月 / 半年 / 年 / 三年 / 五年 / 十年
月流量：例如 500GB/月、1TB/月、不限流量
到期时间
购买链接
是否在前台显示购买信息
```

`月流量` 是自定义文本字段，可以按你的销售或套餐写法填写。`到期时间` 会用于前台的剩余时间和详情页到期日期显示。勾选 `此节点前台显示购买信息` 后，公开面板展开节点详情时会展示卖家、价格、周期、月流量和购买链接。

如果不填写到期时间，前台剩余时间会显示 `-`。

### 删除节点

在节点列表点击 `删除`。删除会移除：

```text
节点上报数据
预创建节点记录
购买信息
```

删除后不可恢复。

## Agent 安装

### 推荐方式：后台生成免输入命令

这是最推荐的方式。

1. 登录 `https://www.monitor.party/admin`。
2. 添加节点，例如 `US-node-001`。
3. 点击生成命令。
4. 复制 Linux 或 Windows 命令到目标服务器执行。

这种方式会自动带上中心端地址、Agent Token 和 Node ID。

### Linux Agent 交互安装

将对应架构的 Agent 二进制和安装脚本放到同一目录：

```text
vps-agent-linux-amd64
install-agent-linux.sh
```

执行：

```bash
chmod +x install-agent-linux.sh vps-agent-linux-amd64
sudo ./install-agent-linux.sh
```

脚本会询问：

```text
Center server URL [https://www.monitor.party]
Agent token
Node ID [当前主机名]
Basic interval [2s]
Disk interval [30s]
Connection interval [60s]
Mounts [auto]
Binary download URL (empty for local file)
```

安装后会创建：

```text
/usr/local/bin/vps-agent
/etc/vps-agent/config.env
/etc/systemd/system/vps-agent.service
```

常用管理命令：

```bash
sudo systemctl status vps-agent
sudo systemctl restart vps-agent
sudo journalctl -u vps-agent -f
```

### Windows Agent 交互安装

将对应架构的 Agent 二进制和安装脚本放到同一目录：

```text
vps-agent-windows-amd64.exe
install-agent-windows.ps1
```

用管理员权限打开 PowerShell，执行：

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope Process
.\install-agent-windows.ps1
```

脚本会询问：

```text
Center server URL [https://www.monitor.party]
Agent token
Node ID [当前计算机名]
Basic interval [2s]
Disk interval [30s]
Connection interval [60s]
```

安装后会创建：

```text
C:\Program Files\vps-agent\vps-agent.exe
C:\ProgramData\vps-agent\config.env
Windows 服务：vps-agent
```

常用管理命令：

```powershell
Get-Service vps-agent
Restart-Service vps-agent
Stop-Service vps-agent
Start-Service vps-agent
```

## 一键使用教程

### 中心端一键安装

```bash
chmod +x install-server-linux.sh vps-server-linux-amd64
sudo ./install-server-linux.sh
```

输入域名、Token、管理员账号密码后，中心端会自动安装为 systemd 服务。

### Linux Agent 一键接入

后台生成命令后，在目标 Linux 服务器执行，例如：

```bash
curl -fsSL https://www.monitor.party/install-agent-linux.sh | sudo sh -s -- --server https://www.monitor.party --token YOUR_AGENT_TOKEN --node-id US-node-001
```

### Linux Agent 一键卸载

后台命令区会直接提供卸载命令。也可以手动执行：

```bash
curl -fsSL https://www.monitor.party/uninstall/agent-linux.sh | sudo sh
```

该命令会停止并禁用 `vps-agent` 服务，删除 `/usr/local/bin/vps-agent`、`/etc/vps-agent` 和 systemd service 文件。

如果使用 release 包中的交互脚本，也可以执行：

```bash
chmod +x install-agent-linux.sh vps-agent-linux-amd64
sudo ./install-agent-linux.sh
```

### Windows Agent 一键接入

后台生成命令后，用管理员 PowerShell 执行。

如果使用 release 包中的交互脚本：

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope Process
.\install-agent-windows.ps1 -Server "https://www.monitor.party" -Token "YOUR_AGENT_TOKEN" -NodeId "US-win-001"
```

### Windows Agent 一键卸载

后台命令区会直接提供卸载命令。也可以用管理员 PowerShell 执行：

```powershell
powershell -ExecutionPolicy Bypass -Command "iwr https://www.monitor.party/uninstall/agent-windows.ps1 -UseBasicParsing | iex"
```

该命令会停止并删除 `vps-agent` 服务，删除 `C:\Program Files\vps-agent` 和 `C:\ProgramData\vps-agent`。

## 配置文件

### 中心端配置

中心端安装脚本会写入：

```text
/etc/vps-monitor/server.env
```

示例：

```env
ADDR=:3000
AGENT_TOKEN=your-agent-token
AUTH_SECRET=your-legacy-auth-secret
ADMIN_USER=admin
ADMIN_PASS=your-admin-password
PUBLIC_URL=https://www.monitor.party
DATA_PATH=/var/lib/vps-monitor/server.json
MAX_NODES=2000
OFFLINE_WAIT=60s
```

修改后重启：

```bash
sudo systemctl restart vps-server
```

`OFFLINE_WAIT` 是离线判定阈值，默认 `60s`。中心端会用收到 Agent 上报的时间作为在线时间，不依赖 Agent 机器本地时间，避免因为节点时间不准导致“数据还在更新但显示离线”。如果你的 Agent 上报间隔设置得更长，请把 `OFFLINE_WAIT` 设置为大于上报间隔的值，例如 `120s`。

### Agent 配置

Linux Agent 配置：

```text
/etc/vps-agent/config.env
```

Windows Agent 配置：

```text
C:\ProgramData\vps-agent\config.env
```

示例：

```env
SERVER=https://www.monitor.party
TOKEN=your-agent-token
NODE_ID=US-node-001
BASIC_INTERVAL=2s
DISK_INTERVAL=30s
CONNECTION_INTERVAL=60s
MOUNTS=auto
NETWORK_EXCLUDE=lo,docker*,veth*,br-*
DISK_EXCLUDE_FS=tmpfs,devtmpfs,overlay,squashfs,proc,sysfs,cgroup,cgroup2
```

## 手动运行

### 中心端

```bash
ADDR=:3000 \
AGENT_TOKEN=your-agent-token \
AUTH_SECRET=your-auth-secret \
ADMIN_USER=admin \
ADMIN_PASS=your-admin-password \
PUBLIC_URL=https://www.monitor.party \
DATA_PATH=./data/server.json \
./vps-server-linux-amd64
```

### Agent

```bash
SERVER=https://www.monitor.party \
TOKEN=your-agent-token \
NODE_ID=US-node-001 \
./vps-agent-linux-amd64 run
```

## 构建发布

### 构建前端

```bash
cd web
npm install
npm run build
cd ..
```

构建后需要把 `web/dist` 同步到后端内嵌目录：

```powershell
Copy-Item -Path "web\dist\*" -Destination "internal\server\web\dist" -Recurse -Force
```

### 构建完整 Release

Windows PowerShell：

```powershell
powershell -ExecutionPolicy Bypass -File "scripts\build-release.ps1"
```

Linux / macOS：

```bash
sh scripts/build-release.sh
```

构建完成后，产物会写入：

```text
release/
```

包含：

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

## 升级

### 已经在运营，如何更新中心端

如果你的中心端已经在线运行，更新时只需要替换 `vps-server` 二进制并重启服务。正常情况下不会删除节点数据、购买信息、到期时间和站点设置。

生产环境推荐按这个顺序操作：

```bash
# 1. 查看当前服务状态
sudo systemctl status vps-server

# 2. 备份数据文件
sudo cp /var/lib/vps-monitor/server.json /var/lib/vps-monitor/server.json.bak.$(date +%Y%m%d%H%M%S)

# 3. 停止中心端
sudo systemctl stop vps-server

# 4. 备份旧二进制
sudo cp /usr/local/bin/vps-server /usr/local/bin/vps-server.bak.$(date +%Y%m%d%H%M%S)

# 5. 替换新二进制，以 amd64 为例
sudo install -m 0755 ./vps-server-linux-amd64 /usr/local/bin/vps-server

# 6. 启动中心端
sudo systemctl start vps-server

# 7. 查看日志和状态
sudo systemctl status vps-server
sudo journalctl -u vps-server -n 100 --no-pager
```

更新完成后访问：

```text
https://www.monitor.party/admin
```

确认以下内容：

```text
后台可以登录
节点列表还在
购买信息还在
到期时间还在
前台 WebSocket 正常刷新
Agent 节点陆续在线
```

如果你用的是 release 包里的交互安装脚本，也可以直接重新执行：

```bash
chmod +x install-server-linux.sh vps-server-linux-amd64
sudo ./install-server-linux.sh
```

脚本会覆盖 `/usr/local/bin/vps-server` 和 systemd service，但默认数据文件仍然是：

```text
/var/lib/vps-monitor/server.json
```

只要你不手动删除这个文件，节点数据、购买信息和到期时间都会保留。

### 回滚中心端

如果新版本启动异常，使用刚才备份的旧二进制回滚：

```bash
sudo systemctl stop vps-server
sudo cp /usr/local/bin/vps-server.bak.YYYYMMDDHHMMSS /usr/local/bin/vps-server
sudo chmod +x /usr/local/bin/vps-server
sudo systemctl start vps-server
sudo journalctl -u vps-server -n 100 --no-pager
```

如果数据文件也需要回滚：

```bash
sudo systemctl stop vps-server
sudo cp /var/lib/vps-monitor/server.json.bak.YYYYMMDDHHMMSS /var/lib/vps-monitor/server.json
sudo systemctl start vps-server
```

### 更新 Agent 下载文件

中心端 release 二进制会内嵌新的 Agent 下载文件。也就是说，只要你更新了 `vps-server`，后台生成的一键安装命令和 `/download/vps-agent-*` 会自动使用新版本内嵌的 Agent。

已有 Agent 不会自动升级，需要在每台 Agent 机器上重新执行安装脚本，或者使用后台生成的新命令重新安装。

### 升级中心端

把新的 `vps-server-linux-amd64` 放到服务器，与 `install-server-linux.sh` 同目录，然后重新执行：

```bash
sudo ./install-server-linux.sh
```

数据默认保存在：

```text
/var/lib/vps-monitor/server.json
```

重新安装不会删除该数据文件。

### 升级 Agent

Linux Agent 已经在线运行时，重新执行 Agent 安装脚本即可覆盖旧二进制并重启服务：

```bash
sudo ./install-agent-linux.sh
```

或者手动替换：

```bash
sudo systemctl stop vps-agent
sudo install -m 0755 ./vps-agent-linux-amd64 /usr/local/bin/vps-agent
sudo systemctl start vps-agent
sudo journalctl -u vps-agent -n 100 --no-pager
```

Windows：

```powershell
.\install-agent-windows.ps1
```

Windows 手动更新可以停止服务、替换 `C:\Program Files\vps-agent\vps-agent.exe`，然后启动服务：

```powershell
Stop-Service vps-agent
Copy-Item .\vps-agent-windows-amd64.exe "C:\Program Files\vps-agent\vps-agent.exe" -Force
Start-Service vps-agent
Get-Service vps-agent
```

## 卸载

### 卸载中心端

```bash
sudo ./uninstall-server-linux.sh
```

### 卸载 Linux Agent

```bash
sudo ./uninstall-agent-linux.sh
```

### 卸载 Windows Agent

用管理员 PowerShell 执行：

```powershell
.\uninstall-agent-windows.ps1
```

## 常见问题

### 前台无法连接 WebSocket

检查反向代理是否正确处理 `/ws` 的 Upgrade 头，并确认前端配置中的 WebSocket 地址为：

```text
wss://www.monitor.party/ws
```

### Agent 一直离线

检查：

```text
SERVER 是否能访问
TOKEN 是否和中心端 AGENT_TOKEN 一致
NODE_ID 是否和后台创建的节点一致
中心端防火墙是否放行 3000 或 443
OFFLINE_WAIT 是否小于 Agent 上报间隔
```

在线状态按中心端收到 Agent 上报的时间判断。默认 `OFFLINE_WAIT=60s`，如果 Agent 的 `BASIC_INTERVAL` 设置为 `120s`，前台就可能在两次上报之间显示离线。解决方式是在 `/etc/vps-monitor/server.env` 中调大：

```env
OFFLINE_WAIT=180s
```

然后重启中心端：

```bash
sudo systemctl restart vps-server
```

如果节点确实离线了，中心端会保留它最后一次上报的数据和购买信息；超过 `OFFLINE_WAIT` 后前台和后台都会显示离线。节点恢复联网并再次成功上报后，中心端会用新的接收时间更新在线时间，前台和后台会自动恢复为在线，不需要重新添加节点。

Linux 查看日志：

```bash
sudo journalctl -u vps-agent -f
```

中心端查看日志：

```bash
sudo journalctl -u vps-server -f
```

### 后台无法登录

检查 `/etc/vps-monitor/server.env`：

```env
ADMIN_USER=admin
ADMIN_PASS=your-admin-password
```

修改后重启：

```bash
sudo systemctl restart vps-server
```

### 页面还是旧内容

如果你从源码构建，需要先构建前端并同步到 `internal/server/web/dist`，再构建 release：

```bash
cd web
npm run build
cd ..
powershell -ExecutionPolicy Bypass -File "scripts\build-release.ps1"
```

## 安全建议

- `AGENT_TOKEN`、`AUTH_SECRET`、`ADMIN_PASS` 请使用强随机字符串。
- 不要把真实生产环境的 `server.env`、Token、密码提交到 GitHub。
- 推荐使用 HTTPS 和反向代理，不建议长期裸露 `http://IP:3000`。
- 管理后台 `/admin` 建议配合防火墙、Cloudflare Access 或 Nginx Basic Auth 做额外保护。

## License

请根据你的发布计划补充 License 文件。
