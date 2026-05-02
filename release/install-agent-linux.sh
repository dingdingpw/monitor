#!/usr/bin/env sh
set -eu

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "please run as root: sudo sh $0" >&2
    exit 1
  fi
}

ask() {
  prompt="$1"
  default="${2:-}"
  if [ -n "$default" ]; then
    printf "%s [%s]: " "$prompt" "$default" >&2
  else
    printf "%s: " "$prompt" >&2
  fi
  read -r value
  if [ -z "$value" ]; then value="$default"; fi
  printf "%s" "$value"
}

ask_secret() {
  prompt="$1"
  printf "%s: " "$prompt" >&2
  stty -echo 2>/dev/null || true
  read -r value
  stty echo 2>/dev/null || true
  printf "\n" >&2
  printf "%s" "$value"
}

arch_name() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l|armv7*) echo "armv7" ;;
    i386|i686) echo "386" ;;
    *) echo "unsupported" ;;
  esac
}

find_binary() {
  arch="$1"
  for name in "./vps-agent-linux-$arch" "./release/vps-agent-linux-$arch" "./vps-agent"; do
    if [ -f "$name" ]; then echo "$name"; return 0; fi
  done
  return 1
}

need_root

ARCH="$(arch_name)"
if [ "$ARCH" = "unsupported" ]; then
  echo "unsupported architecture: $(uname -m)" >&2
  exit 1
fi

echo "VPS Monitor agent installer"
echo "detected: linux/$ARCH"

SERVER="$(ask "Center server URL" "https://www.monitor.party")"
TOKEN="$(ask_secret "Agent token")"
NODE_ID="$(ask "Node ID" "$(hostname)")"
BASIC_INTERVAL="$(ask "Basic interval" "2s")"
DISK_INTERVAL="$(ask "Disk interval" "30s")"
CONNECTION_INTERVAL="$(ask "Connection interval" "60s")"
MOUNTS="$(ask "Mounts" "auto")"
BIN_URL="$(ask "Binary download URL (empty for local file)" "")"

install -d /etc/vps-agent /usr/local/bin

if [ -n "$BIN_URL" ]; then
  TMP="$(mktemp)"
  curl -fsSL "$BIN_URL" -o "$TMP"
  install -m 0755 "$TMP" /usr/local/bin/vps-agent
  rm -f "$TMP"
else
  BIN="$(find_binary "$ARCH")" || { echo "vps-agent binary not found for linux/$ARCH" >&2; exit 1; }
  install -m 0755 "$BIN" /usr/local/bin/vps-agent
fi

cat >/etc/vps-agent/config.env <<EOF
SERVER=$SERVER
TOKEN=$TOKEN
NODE_ID=$NODE_ID
BASIC_INTERVAL=$BASIC_INTERVAL
DISK_INTERVAL=$DISK_INTERVAL
CONNECTION_INTERVAL=$CONNECTION_INTERVAL
MOUNTS=$MOUNTS
NETWORK_EXCLUDE=lo,docker*,veth*,br-*
DISK_EXCLUDE_FS=tmpfs,devtmpfs,overlay,squashfs,proc,sysfs,cgroup,cgroup2
EOF

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

/usr/local/bin/vps-agent test --config /etc/vps-agent/config.env || true
systemctl daemon-reload
systemctl enable --now vps-agent
systemctl --no-pager --full status vps-agent || true
echo "agent installed: $NODE_ID -> $SERVER"
