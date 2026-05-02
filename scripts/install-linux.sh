#!/usr/bin/env sh
set -eu

SERVER=""
TOKEN=""
NODE_ID="$(hostname)"
BIN_URL=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --server) SERVER="$2"; shift 2 ;;
    --token) TOKEN="$2"; shift 2 ;;
    --node-id) NODE_ID="$2"; shift 2 ;;
    --bin-url) BIN_URL="$2"; shift 2 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

if [ -z "$SERVER" ] || [ -z "$TOKEN" ]; then
  echo "usage: install-linux.sh --server https://monitor.example.com --token TOKEN [--node-id NODE] [--bin-url URL]" >&2
  exit 2
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) GOARCH="amd64" ;;
  aarch64|arm64) GOARCH="arm64" ;;
  armv7l|armv7*) GOARCH="armv7" ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac

install -d /etc/vps-agent /usr/local/bin

if [ -n "$BIN_URL" ]; then
  TMP="$(mktemp)"
  curl -fsSL "$BIN_URL" -o "$TMP"
  install -m 0755 "$TMP" /usr/local/bin/vps-agent
  rm -f "$TMP"
else
  if [ ! -x ./vps-agent ]; then
    echo "./vps-agent not found; pass --bin-url or run from a build directory" >&2
    exit 1
  fi
  install -m 0755 ./vps-agent /usr/local/bin/vps-agent
fi

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
echo "vps-agent installed for linux/$GOARCH"
