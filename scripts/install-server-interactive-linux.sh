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
  for name in "./vps-server-linux-$arch" "./release/vps-server-linux-$arch" "./vps-server"; do
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

echo "VPS Monitor center server installer"
echo "detected: linux/$ARCH"

PUBLIC_URL="$(ask "Public URL" "https://www.monitor.party")"
AGENT_TOKEN="$(ask_secret "Agent token")"
AUTH_SECRET="$(ask_secret "Legacy auth secret")"
ADMIN_USER="$(ask "Admin username" "admin")"
ADMIN_PASS="$(ask_secret "Admin password")"
ADDR="$(ask "Listen address" ":3000")"
MAX_NODES="$(ask "Max nodes" "2000")"
BIN_URL="$(ask "Binary download URL (empty for local file)" "")"

install -d /etc/vps-monitor /usr/local/bin /var/lib/vps-monitor

systemctl stop vps-server 2>/dev/null || true
pkill -f '/usr/local/bin/vps-server' 2>/dev/null || true
sleep 1
rm -f /usr/local/bin/vps-server 2>/dev/null || true
if [ -e /usr/local/bin/vps-server ]; then
  chattr -i /usr/local/bin/vps-server 2>/dev/null || true
  rm -f /usr/local/bin/vps-server 2>/dev/null || true
fi
if [ -e /usr/local/bin/vps-server ]; then
  echo "failed to remove old /usr/local/bin/vps-server" >&2
  ls -l /usr/local/bin/vps-server >&2 || true
  exit 1
fi

if [ -n "$BIN_URL" ]; then
  TMP="$(mktemp)"
  curl -fsSL "$BIN_URL" -o "$TMP"
  install -m 0755 "$TMP" /usr/local/bin/vps-server
  rm -f "$TMP"
else
  BIN="$(find_binary "$ARCH")" || { echo "vps-server binary not found for linux/$ARCH" >&2; exit 1; }
  install -m 0755 "$BIN" /usr/local/bin/vps-server
fi

if ! strings /usr/local/bin/vps-server 2>/dev/null | grep -q "monitor-party-admin-v1"; then
  echo "installed vps-server does not contain admin backend marker; wrong binary may have been installed" >&2
  exit 1
fi

cat >/etc/vps-monitor/server.env <<EOF
ADDR=$ADDR
AGENT_TOKEN=$AGENT_TOKEN
AUTH_SECRET=$AUTH_SECRET
ADMIN_USER=$ADMIN_USER
ADMIN_PASS=$ADMIN_PASS
PUBLIC_URL=$PUBLIC_URL
DATA_PATH=/var/lib/vps-monitor/server.json
MAX_NODES=$MAX_NODES
EOF

cat >/etc/systemd/system/vps-server.service <<'EOF'
[Unit]
Description=VPS Monitor Center Server
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/vps-monitor/server.env
ExecStart=/usr/local/bin/vps-server
Restart=always
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now vps-server
sleep 1
PORT="${ADDR##*:}"
if [ -z "$PORT" ] || [ "$PORT" = "$ADDR" ]; then PORT="3000"; fi
if ! curl -fsS "http://127.0.0.1:$PORT/admin" 2>/dev/null | grep -q "monitor-party-admin-v1"; then
  echo "warning: local /admin check failed; inspect journalctl -u vps-server" >&2
fi
systemctl --no-pager --full status vps-server || true
echo "server installed: $PUBLIC_URL"
