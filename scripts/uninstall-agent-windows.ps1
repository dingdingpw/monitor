$ErrorActionPreference = "SilentlyContinue"
Stop-Service vps-agent
sc.exe delete vps-agent | Out-Null
Remove-Item -Recurse -Force "C:\Program Files\vps-agent"
Remove-Item -Recurse -Force "C:\ProgramData\vps-agent"
Write-Host "vps-agent uninstalled"
