$ErrorActionPreference = "Stop"

$root = Resolve-Path "$PSScriptRoot\.."
$release = Join-Path $root "release"
$embedBins = Join-Path $root "internal\server\agent_bins"
$web = Join-Path $root "web"
$webDist = Join-Path $web "dist"
$embedWebDist = Join-Path $root "internal\server\web\dist"
New-Item -ItemType Directory -Force -Path $release | Out-Null
New-Item -ItemType Directory -Force -Path $embedBins | Out-Null

Push-Location $web
try {
  npm run build
} finally {
  Pop-Location
}

if (Test-Path $embedWebDist) {
  Remove-Item $embedWebDist -Recurse -Force
}
New-Item -ItemType Directory -Force -Path (Split-Path $embedWebDist -Parent) | Out-Null
Copy-Item $webDist $embedWebDist -Recurse -Force

$targets = @(
  @{ OS = "linux"; Arch = "amd64"; Arm = "" },
  @{ OS = "linux"; Arch = "arm64"; Arm = "" },
  @{ OS = "linux"; Arch = "arm"; Arm = "7" },
  @{ OS = "linux"; Arch = "386"; Arm = "" },
  @{ OS = "windows"; Arch = "amd64"; Arm = "" },
  @{ OS = "windows"; Arch = "arm64"; Arm = "" },
  @{ OS = "windows"; Arch = "386"; Arm = "" }
)

function Build-One($cmd, $name, $target) {
  $env:CGO_ENABLED = "0"
  $env:GOOS = $target.OS
  $env:GOARCH = $target.Arch
  if ($target.Arm) { $env:GOARM = $target.Arm } else { Remove-Item Env:\GOARM -ErrorAction SilentlyContinue }

  $suffix = "$($target.OS)-$($target.Arch)"
  if ($target.Arm) { $suffix = "${suffix}v$($target.Arm)" }
  $ext = ""
  if ($target.OS -eq "windows") { $ext = ".exe" }
	$out = Join-Path $release "$name-$suffix$ext"
	Write-Host "building $out"
	go build -p 1 -trimpath -ldflags "-s -w" -o $out $cmd
	if ($LASTEXITCODE -ne 0) { throw "go build failed for $name-$suffix$ext" }
}

foreach ($target in $targets) {
  if ($target.OS -eq "linux" -or $target.OS -eq "windows") {
    Build-One "./cmd/vps-agent" "vps-agent" $target
  }
}

Copy-Item (Join-Path $release "vps-agent-*") $embedBins -Force

foreach ($target in $targets) {
  if ($target.OS -eq "linux" -or $target.OS -eq "windows") {
    Build-One "./cmd/vps-server" "vps-server" $target
  }
}

Copy-Item (Join-Path $root "scripts\install-server-interactive-linux.sh") (Join-Path $release "install-server-linux.sh") -Force
Copy-Item (Join-Path $root "scripts\install-agent-interactive-linux.sh") (Join-Path $release "install-agent-linux.sh") -Force
Copy-Item (Join-Path $root "scripts\install-agent-interactive-windows.ps1") (Join-Path $release "install-agent-windows.ps1") -Force
Copy-Item (Join-Path $root "scripts\uninstall-server-linux.sh") (Join-Path $release "uninstall-server-linux.sh") -Force
Copy-Item (Join-Path $root "scripts\uninstall-agent-linux.sh") (Join-Path $release "uninstall-agent-linux.sh") -Force
Copy-Item (Join-Path $root "scripts\uninstall-agent-windows.ps1") (Join-Path $release "uninstall-agent-windows.ps1") -Force

Write-Host "release files written to $release"
