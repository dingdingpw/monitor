param(
  [Parameter(Mandatory=$true)][string]$Server,
  [Parameter(Mandatory=$true)][string]$Token,
  [string]$NodeId = $env:COMPUTERNAME,
  [string]$BinaryPath = ".\vps-agent.exe"
)

$ErrorActionPreference = "Stop"
$installDir = "C:\Program Files\vps-agent"
$configDir = "C:\ProgramData\vps-agent"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
New-Item -ItemType Directory -Force -Path $configDir | Out-Null
icacls $configDir /inheritance:r /grant:r "Administrators:(OI)(CI)F" "SYSTEM:(OI)(CI)F" | Out-Null

Copy-Item $BinaryPath "$installDir\vps-agent.exe" -Force
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
}

sc.exe create vps-agent binPath= "`"$installDir\vps-agent.exe`" run --config `"$configDir\config.env`"" start= auto DisplayName= "VPS Monitor Agent" | Out-Null
Start-Service vps-agent
Write-Host "vps-agent installed"
