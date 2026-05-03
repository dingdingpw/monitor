package server

const adminHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta name="monitor-admin-marker" content="monitor-party-admin-v1">
  <meta charset="utf-8">
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <link rel="shortcut icon" href="/favicon.ico">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Monitor Party Console</title>
  <style>
    :root{--ink:#111111;--muted:#6b7280;--line:#e5e7eb;--green:#047857;--red:#b91c1c;--panel:#ffffff;--soft:#f7f7f8;--side:#111111;--focus:#111111}
    *{box-sizing:border-box}body{margin:0;min-height:100vh;background:#f7f7f8;color:var(--ink);font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI","Microsoft YaHei",sans-serif}
    .shell{display:grid;grid-template-columns:268px 1fr;min-height:100vh}.side{padding:26px;background:#111;color:#fff;border-right:1px solid #111}.mark{width:44px;height:44px;border-radius:12px;background:#fff;color:#111;display:grid;place-items:center;font-weight:900;font-size:21px;border:1px solid rgba(255,255,255,.18)}.brand h1{margin:18px 0 8px;font-size:22px}.brand p{color:#c9c9c9;line-height:1.7}.nav{margin-top:30px}.nav a{display:block;color:#f5f5f5;text-decoration:none;padding:12px 13px;border-radius:10px;margin-bottom:8px;border:1px solid rgba(255,255,255,.12);background:#1b1b1b}.nav a:hover{background:#fff;color:#111;border-color:#fff}
    .main{padding:28px;min-width:0}.top{display:flex;align-items:stretch;justify-content:space-between;gap:16px;margin-bottom:18px}.hero{flex:1;padding:24px;border:1px solid var(--line);border-radius:14px;background:#fff;box-shadow:0 12px 36px rgba(17,17,17,.06)}.hero h2{font-size:30px;margin:0 0 8px;letter-spacing:-.04em}.muted{color:var(--muted)}.grid{display:grid;grid-template-columns:1fr 1fr;gap:16px}.card{background:#fff;border:1px solid var(--line);border-radius:14px;padding:20px;margin-bottom:16px;box-shadow:0 10px 28px rgba(17,17,17,.05)}.card h3{margin:0 0 14px;font-size:17px}.row{display:flex;gap:10px;flex-wrap:wrap;align-items:center}input,select{height:42px;border:1px solid var(--line);border-radius:8px;padding:0 12px;font-size:14px;min-width:210px;background:#fff;color:var(--ink);outline:none}input:focus,select:focus{border-color:#111;box-shadow:0 0 0 3px rgba(17,17,17,.12)}select{cursor:pointer}.check{height:42px;display:flex;align-items:center;gap:8px;color:var(--ink);font-size:14px}.check input{min-width:0;width:16px;height:16px;accent-color:#111}button{height:42px;border:0;border-radius:8px;background:#111;color:#fff;padding:0 16px;font-weight:800;cursor:pointer}button:hover{background:#2a2a2a}button.secondary{background:#f3f4f6;color:#111;border:1px solid var(--line)}button.ghost{background:#fff;color:#111;border:1px solid var(--line)}button.danger{background:#fff;color:var(--red);border:1px solid #fecaca}.hidden{display:none!important}.statbar{display:grid;grid-template-columns:repeat(3,1fr);gap:14px;margin:16px 0}.stat{padding:16px;border-radius:14px;background:#fff;border:1px solid var(--line)}.stat b{display:block;font-size:28px}.stat span{font-size:12px;color:var(--muted);letter-spacing:.1em}
    table{width:100%;border-collapse:separate;border-spacing:0;overflow:hidden}th,td{text-align:left;padding:13px;border-bottom:1px solid #edf0f5;font-size:14px}th{color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.08em;background:#f8f8f8}.ok{color:var(--green);font-weight:800}.off{color:var(--red);font-weight:800}.pill{display:inline-block;border-radius:999px;padding:5px 10px;background:#f3f4f6;color:#111;font-weight:800;font-size:12px}textarea{width:100%;min-height:118px;border:1px solid #222;border-radius:10px;padding:12px;background:#111;color:#f5f5f5;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:12px;line-height:1.6}.login{min-height:100vh;display:grid;grid-template-columns:minmax(320px,42vw) 1fr;background:#fff}.login-hero{padding:56px;display:flex;flex-direction:column;justify-content:space-between;background:#111;color:#fff}.login-hero .mark{background:#fff;color:#111}.login-hero h1{margin:48px 0 14px;font-size:46px;line-height:1;letter-spacing:-.06em}.login-hero p{max-width:520px;color:#d1d5db;font-size:16px;line-height:1.8}.login-points{display:grid;gap:10px;margin-top:36px}.login-points span{display:block;padding:12px 0;border-top:1px solid rgba(255,255,255,.14);color:#f5f5f5}.login-form{padding:56px;display:flex;align-items:center;justify-content:center}.login-panel{width:100%;max-width:460px}.login-panel h2{margin:0 0 10px;font-size:30px;letter-spacing:-.04em}.login-panel p{margin:0 0 28px}.login-panel input,.login-panel button{width:100%;min-width:0}.login-panel .row{display:grid;gap:12px}.toast{position:fixed;right:24px;bottom:24px;background:#111;color:#fff;padding:11px 15px;border-radius:10px;box-shadow:0 18px 50px rgba(17,17,17,.22)}
    @media(max-width:900px){.shell{display:block}.side{border-radius:0 0 28px 28px}.grid,.statbar{grid-template-columns:1fr}.top{display:block}.main{padding:18px}input,select{min-width:100%;width:100%}button{width:100%}table{display:block;overflow-x:auto;white-space:nowrap}.login{display:block}.login-hero{min-height:42vh;padding:30px}.login-hero h1{font-size:36px;margin-top:36px}.login-form{padding:28px;display:block}.login-panel{max-width:none}}
  </style>
</head>
<body>
  <div id="login" class="login">
    <section class="login-hero">
      <div><div class="mark">M</div><h1>Monitor Party 后台</h1><p>登录后可管理节点、查看在线状态、维护套餐信息，并生成 Agent 安装命令。</p><div class="login-points"><span>节点：添加、删除、查看在线状态</span><span>套餐：卖家、价格、周期、流量、到期时间</span><span>安装：生成 Linux / Windows 接入命令</span></div></div>
      <p>仅供管理员和运营人员使用。</p>
    </section>
    <section class="login-form">
      <div class="login-panel">
        <h2>登录控制台</h2>
        <p class="muted">输入管理员账号密码进入后台。</p>
        <div class="row"><input id="username" placeholder="用户名" value="admin"><input id="password" placeholder="密码" type="password"><button onclick="login()">登录</button></div>
      </div>
    </section>
  </div>
  <div id="panel" class="shell hidden">
    <aside class="side"><div class="brand"><div class="mark">M</div><h1>Monitor Party</h1><p>节点接入、安装命令和在线状态管理。</p></div><div class="nav"><a href="/">公开面板</a><a href="#nodes">节点管理</a><a href="#commands">安装命令</a></div></aside>
    <main class="main">
      <div class="top"><div class="hero"><h2>Agent 接入控制台</h2><div class="muted">统一管理节点、购买周期和免输入安装命令。</div></div><button class="danger" onclick="logout()">退出登录</button></div>
      <div class="statbar"><div class="stat"><b id="totalCount">0</b><span>TOTAL</span></div><div class="stat"><b id="onlineCount">0</b><span>ONLINE</span></div><div class="stat"><b id="offlineCount">0</b><span>PENDING</span></div></div>
      <div class="grid">
        <section class="card"><h3>添加节点</h3><div class="row"><input id="nodeId" placeholder="US-node-001"><button onclick="addNode()">添加并生成</button><button class="secondary" onclick="loadNodes()">刷新</button><button class="ghost" onclick="exportNodes()">一键导出</button><button class="ghost" onclick="nodeImportFile.click()">一键导入</button><input id="nodeImportFile" type="file" accept="application/json,.json" class="hidden" onchange="importNodes(this)"></div><p class="muted">Node ID 必须唯一，建议前两位使用国家或地区代码。导入会合并节点和套餐信息，不会删除现有节点。</p></section>
        <section class="card"><h3>站点设置</h3><div class="row"><input id="siteName" placeholder="Monitor Party"><button onclick="saveSettings()">保存设置</button></div><p class="muted">站名默认 Monitor Party，可在这里修改。</p></section>
      </div>
      <section id="editInfo" class="card hidden"><h3>编辑主机信息</h3><div class="row"><input id="editNodeName" readonly><input id="editSeller" placeholder="卖家"><input id="editPrice" placeholder="价格"><select id="editCycle"><option value="">选择周期</option><option value="日">日</option><option value="月">月</option><option value="半年">半年</option><option value="年">年</option><option value="三年">三年</option><option value="五年">五年</option><option value="十年">十年</option></select><input id="editBandwidth" placeholder="带宽，例如 1Gbps"><input id="editTraffic" placeholder="月流量，例如 1TB/月"><input id="editDueTime" type="date" min="1970-01-01" max="9999-12-31" title="到期时间" oninput="normalizeDueDateInput()" onchange="normalizeDueDateInput()"><input id="editBuyUrl" placeholder="购买链接"><label class="check"><input id="editShowPurchase" type="checkbox"> 此节点前台显示购买信息</label><button onclick="saveNodeInfo()">保存信息</button><button class="secondary" onclick="hideEditInfo()">取消</button></div></section>
      <section id="commands" class="card hidden"><h3>免输入安装 / 卸载命令</h3><p><span class="pill">Linux 安装</span></p><textarea id="linuxCmd" readonly></textarea><p><button class="secondary" onclick="copyText('linuxCmd')">复制 Linux 安装命令</button></p><p><span class="pill">Linux 卸载</span></p><textarea id="linuxUninstallCmd" readonly></textarea><p><button class="secondary" onclick="copyText('linuxUninstallCmd')">复制 Linux 卸载命令</button></p><p><span class="pill">Windows PowerShell 管理员安装</span></p><textarea id="windowsCmd" readonly></textarea><p><button class="secondary" onclick="copyText('windowsCmd')">复制 Windows 安装命令</button></p><p><span class="pill">Windows PowerShell 管理员卸载</span></p><textarea id="windowsUninstallCmd" readonly></textarea><p><button class="secondary" onclick="copyText('windowsUninstallCmd')">复制 Windows 卸载命令</button></p></section>
      <section id="nodes" class="card"><h3>节点列表</h3><table><thead><tr><th>节点</th><th>状态</th><th>卖家</th><th>价格</th><th>周期</th><th>带宽</th><th>月流量</th><th>到期时间</th><th>最后上报</th><th>操作</th></tr></thead><tbody id="nodeRows"></tbody></table></section>
    </main>
  </div>
<script>
async function api(path, opts){const r=await fetch(path,Object.assign({credentials:'include'},opts||{}));if(!r.ok)throw new Error(await r.text());return r.json()}
function toast(t){const el=document.createElement('div');el.className='toast';el.textContent=t;document.body.appendChild(el);setTimeout(function(){el.remove()},1800)}
async function check(){try{const r=await api('/api/admin/me');if(r.authenticated){showPanel();loadNodes()}}catch(e){}}
function showPanel(){loginEl().classList.add('hidden');document.getElementById('panel').classList.remove('hidden')}
function loginEl(){return document.getElementById('login')}
async function login(){try{await api('/api/admin/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:username.value,password:password.value})});showPanel();loadNodes();toast('登录成功')}catch(e){toast('登录失败')}}
async function logout(){await api('/api/admin/logout',{method:'POST'});location.reload()}
async function loadSettings(){try{const s=await api('/api/admin/settings');siteName.value=s.site_name||'Monitor Party'}catch(e){}}
async function saveSettings(){try{await api('/api/admin/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({site_name:siteName.value.trim()||'Monitor Party'})});toast('设置已保存')}catch(e){toast(e.message)}}
async function addNode(){const id=nodeId.value.trim();if(!id){toast('请输入节点 ID');return}try{await api('/api/admin/nodes',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({node_id:id})});await showCommands(id);await loadNodes();toast('节点已添加')}catch(e){toast(e.message)}}
async function exportNodes(){try{const data=await api('/api/admin/nodes/export');const blob=new Blob([JSON.stringify(data,null,2)],{type:'application/json'});const a=document.createElement('a');a.href=URL.createObjectURL(blob);a.download='monitor-nodes-'+new Date().toISOString().slice(0,10)+'.json';document.body.appendChild(a);a.click();URL.revokeObjectURL(a.href);a.remove();toast('节点已导出')}catch(e){toast(e.message)}}
async function importNodes(input){const file=input.files&&input.files[0];input.value='';if(!file)return;if(!confirm('导入会合并节点和套餐信息，不会删除现有节点。继续导入？'))return;try{const text=await file.text();JSON.parse(text);const r=await api('/api/admin/nodes/import',{method:'POST',headers:{'Content-Type':'application/json'},body:text});await loadNodes();toast('已导入 '+r.imported+' 个节点')}catch(e){toast('导入失败：'+e.message)}}
async function showCommands(id){const r=await api('/api/admin/install-command?node_id='+encodeURIComponent(id));linuxCmd.value=r.linux;windowsCmd.value=r.windows;linuxUninstallCmd.value=r.linux_uninstall;windowsUninstallCmd.value=r.windows_uninstall;commands.classList.remove('hidden');commands.scrollIntoView({behavior:'smooth',block:'start'})}
function dateValue(v){if(!v)return '';const t=Number(v);const d=new Date(t>0&&t<1000000000000?t*1000:v);if(isNaN(d.getTime()))return '';return d.toISOString().slice(0,10)}
function dateText(v){return dateValue(v)||'-'}
function validDueDate(v){return !v||/^\d{4}-\d{2}-\d{2}$/.test(v)}
function normalizeDueDateInput(){if(!validDueDate(editDueTime.value)){editDueTime.value='';toast('到期时间年份只能是 4 位')}}
function cell(text,className){const td=document.createElement('td');if(className)td.className=className;td.textContent=text;return td}
function actionButton(text,className,handler){const btn=document.createElement('button');btn.className=className;btn.type='button';btn.textContent=text;btn.addEventListener('click',handler);return btn}
async function loadNodes(){await loadSettings();const list=await api('/api/admin/nodes');window.nodeCache=list;totalCount.textContent=list.length;onlineCount.textContent=list.filter(function(n){return n.online}).length;offlineCount.textContent=list.filter(function(n){return !n.online}).length;nodeRows.replaceChildren();list.forEach(function(n){const info=n.info||{};const tr=document.createElement('tr');const nameCell=document.createElement('td');const bold=document.createElement('b');bold.textContent=n.node_id;nameCell.appendChild(bold);tr.appendChild(nameCell);tr.appendChild(cell(n.online?'在线':'待安装/离线',n.online?'ok':'off'));tr.appendChild(cell(info.seller||'-'));tr.appendChild(cell(info.price||'-'));tr.appendChild(cell(info.cycle||'-'));tr.appendChild(cell(info.bandwidth||'-'));tr.appendChild(cell(info.traffic||'-'));tr.appendChild(cell(dateText(info.due_time)));tr.appendChild(cell(n.last_seen?new Date(n.last_seen*1000).toLocaleString():'-'));const actions=document.createElement('td');actions.appendChild(actionButton('命令','ghost',function(){showCommands(n.node_id)}));actions.appendChild(document.createTextNode(' '));actions.appendChild(actionButton('编辑','ghost',function(){editNode(n.node_id)}));actions.appendChild(document.createTextNode(' '));actions.appendChild(actionButton('删除','danger',function(){deleteNode(n.node_id)}));tr.appendChild(actions);nodeRows.appendChild(tr)})}
function editNode(id){const n=(window.nodeCache||[]).find(function(x){return x.node_id===id})||{};const info=n.info||{};editNodeName.value=id;editSeller.value=info.seller||'';editPrice.value=info.price||'';editCycle.value=info.cycle||'';editBandwidth.value=info.bandwidth||'';editTraffic.value=info.traffic||'';editDueTime.value=dateValue(info.due_time);editBuyUrl.value=info.buy_url||'';editShowPurchase.checked=!!info.show_purchase_info;editInfo.classList.remove('hidden');editInfo.scrollIntoView({behavior:'smooth',block:'start'})}
function hideEditInfo(){editInfo.classList.add('hidden')}
async function saveNodeInfo(){if(!validDueDate(editDueTime.value)){toast('到期时间年份只能是 4 位');return}try{const due=editDueTime.value?new Date(editDueTime.value+'T00:00:00').getTime():0;await api('/info',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:editNodeName.value,seller:editSeller.value,price:editPrice.value,cycle:editCycle.value,bandwidth:editBandwidth.value,traffic:editTraffic.value,buy_url:editBuyUrl.value,due_time:due,show_purchase_info:editShowPurchase.checked})});hideEditInfo();await loadNodes();toast('主机信息已保存')}catch(e){toast(e.message)}}
async function deleteNode(id){if(!confirm('确定删除 '+id+' ?'))return;try{await api('/delete',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:id})});await loadNodes();toast('节点已删除')}catch(e){toast(e.message)}}
async function copyText(id){const el=document.getElementById(id);await navigator.clipboard.writeText(el.value);toast('已复制')}
check();
</script>
</body>
</html>`

const linuxInstallTemplate = `#!/usr/bin/env sh
set -eu

SERVER=""
TOKEN=""
NODE_ID=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --server) SERVER="$2"; shift 2 ;;
    --token) TOKEN="$2"; shift 2 ;;
    --node-id) NODE_ID="$2"; shift 2 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

if [ -z "$SERVER" ] || [ -z "$TOKEN" ] || [ -z "$NODE_ID" ]; then
  echo "usage: install-agent-linux.sh --server URL --token TOKEN --node-id NODE" >&2
  exit 2
fi

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7l|armv7*) ARCH="armv7" ;;
  i386|i686) ARCH="386" ;;
  *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

install -d /etc/vps-agent /usr/local/bin
umask 077
TMP="$(mktemp)"
curl -fsSL "%s/download/vps-agent-linux-$ARCH" -o "$TMP"
install -m 0755 "$TMP" /usr/local/bin/vps-agent
rm -f "$TMP"

cat >/etc/vps-agent/config.env <<EOF
SERVER=$SERVER
TOKEN=$TOKEN
NODE_ID=$NODE_ID
BASIC_INTERVAL=2s
DISK_INTERVAL=30s
CONNECTION_INTERVAL=60s
MOUNTS=auto
NETWORK_EXCLUDE=lo,docker*,veth*,br-*
DISK_EXCLUDE_FS=tmpfs,devtmpfs,overlay,squashfs,proc,sysfs,cgroup,cgroup2
EOF
chmod 600 /etc/vps-agent/config.env

cat >/etc/systemd/system/vps-agent.service <<'EOF'
[Unit]
Description=Lightweight VPS Monitor Agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/vps-agent run --config /etc/vps-agent/config.env
Restart=always
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now vps-agent
echo "vps-agent installed: $NODE_ID -> $SERVER"
`

const windowsInstallTemplate = `$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

function Install-VpsAgent {
  param(
    [Parameter(Mandatory=$true)][string]$Server,
    [Parameter(Mandatory=$true)][string]$Token,
    [Parameter(Mandatory=$true)][string]$NodeId
  )

  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal($identity)
  if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw "please run PowerShell as Administrator"
  }

  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $arch = "amd64" }
    "ARM64" { $arch = "arm64" }
    "x86" { $arch = "386" }
    default { $arch = "amd64" }
  }

  $installDir = "C:\\Program Files\\vps-agent"
  $configDir = "C:\\ProgramData\\vps-agent"
  New-Item -ItemType Directory -Force -Path $installDir | Out-Null
  New-Item -ItemType Directory -Force -Path $configDir | Out-Null
  icacls $configDir /inheritance:r /grant:r "Administrators:(OI)(CI)F" "SYSTEM:(OI)(CI)F" | Out-Null

  $tmp = Join-Path $env:TEMP "vps-agent.exe"
  Invoke-WebRequest "%s/download/vps-agent-windows-$arch.exe" -OutFile $tmp -UseBasicParsing
  Copy-Item $tmp "$installDir\\vps-agent.exe" -Force
  Remove-Item $tmp -Force

  $configText = @"
SERVER=$Server
TOKEN=$Token
NODE_ID=$NodeId
BASIC_INTERVAL=2s
DISK_INTERVAL=30s
CONNECTION_INTERVAL=60s
MOUNTS=auto
"@
  [System.IO.File]::WriteAllText("$configDir\config.env", $configText, (New-Object System.Text.UTF8Encoding($false)))
  icacls "$configDir\config.env" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" | Out-Null

  $service = Get-Service -Name "vps-agent" -ErrorAction SilentlyContinue
  if ($service) {
    Stop-Service vps-agent -ErrorAction SilentlyContinue
    sc.exe delete vps-agent | Out-Null
    for ($i = 0; $i -lt 10; $i++) {
      Start-Sleep -Seconds 1
      if (-not (Get-Service -Name "vps-agent" -ErrorAction SilentlyContinue)) { break }
    }
  }

  $quote = [char]34
  $binPath = $quote + $installDir + "\vps-agent.exe" + $quote + " run --config " + $quote + $configDir + "\config.env" + $quote
  New-Service -Name "vps-agent" -BinaryPathName $binPath -DisplayName "VPS Monitor Agent" -StartupType Automatic | Out-Null
  Start-Service vps-agent
  Write-Host "vps-agent installed: $NodeId -> $Server"
}
`

const linuxUninstallTemplate = `#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "please run as root: sudo sh" >&2
  exit 1
fi

systemctl disable --now vps-agent 2>/dev/null || true
rm -f /etc/systemd/system/vps-agent.service /usr/local/bin/vps-agent
rm -rf /etc/vps-agent
systemctl daemon-reload 2>/dev/null || true
echo "vps-agent uninstalled"
`

const windowsUninstallTemplate = `$ErrorActionPreference = "SilentlyContinue"

$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
  throw "please run PowerShell as Administrator"
}

Stop-Service vps-agent
sc.exe delete vps-agent | Out-Null
Remove-Item -Recurse -Force "C:\Program Files\vps-agent"
Remove-Item -Recurse -Force "C:\ProgramData\vps-agent"
Write-Host "vps-agent uninstalled"
`
