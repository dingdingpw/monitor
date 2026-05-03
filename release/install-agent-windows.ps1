param(
  [string]$Server = "",
  [string]$Token = "",
  [string]$NodeId = "",
  [string]$BinaryPath = ""
)

$ErrorActionPreference = "Stop"

function Ask($Prompt, $Default = "") {
  if ($Default) { $value = Read-Host "$Prompt [$Default]" } else { $value = Read-Host $Prompt }
  if (-not $value) { return $Default }
  return $value
}

function Ask-SecretText($Prompt) {
  $secure = Read-Host $Prompt -AsSecureString
  $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
  try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr) }
  finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr) }
}

function Get-ArchName() {
  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { return "amd64" }
    "ARM64" { return "arm64" }
    "x86" { return "386" }
    default { return "amd64" }
  }
}

function Find-Binary($Arch) {
  $names = @(
    ".\vps-agent-windows-$Arch.exe",
    ".\release\vps-agent-windows-$Arch.exe",
    ".\vps-agent.exe"
  )
  foreach ($name in $names) {
    if (Test-Path $name) { return (Resolve-Path $name).Path }
  }
  return ""
}

$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
  throw "please run PowerShell as Administrator"
}

$arch = Get-ArchName
Write-Host "VPS Monitor Windows agent installer"
Write-Host "detected: windows/$arch"

if (-not $Server) { $Server = Ask "Center server URL" "https://www.monitor.party" }
if (-not $Token) { $Token = Ask-SecretText "Agent token" }
if (-not $NodeId) { $NodeId = Ask "Node ID" $env:COMPUTERNAME }
$BasicInterval = Ask "Basic interval" "2s"
$DiskInterval = Ask "Disk interval" "30s"
$ConnectionInterval = Ask "Connection interval" "60s"
if (-not $BinaryPath) { $BinaryPath = Find-Binary $arch }
if (-not $BinaryPath) { throw "vps-agent binary not found for windows/$arch" }

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
BASIC_INTERVAL=$BasicInterval
DISK_INTERVAL=$DiskInterval
CONNECTION_INTERVAL=$ConnectionInterval
MOUNTS=auto
"@
[System.IO.File]::WriteAllText("$configDir\config.env", $configText, (New-Object System.Text.UTF8Encoding($false)))
icacls "$configDir\config.env" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" | Out-Null

$service = Get-Service -Name "vps-agent" -ErrorAction SilentlyContinue
if ($service) {
  Stop-Service vps-agent -ErrorAction SilentlyContinue
  sc.exe delete vps-agent | Out-Null
  Start-Sleep -Seconds 1
}

sc.exe create vps-agent binPath= "`"$installDir\vps-agent.exe`" run --config `"$configDir\config.env`"" start= auto DisplayName= "VPS Monitor Agent" | Out-Null
Start-Service vps-agent
Get-Service vps-agent
Write-Host "agent installed: $NodeId -> $Server"
