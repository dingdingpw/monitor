#!/usr/bin/env sh
set -eu

AUTH_SECRET=""
ADMIN_USER="admin"
ADMIN_PASS=""
PUBLIC_URL=""
BIN_URL=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --auth-secret) AUTH_SECRET="$2"; shift 2 ;;
    --admin-user) ADMIN_USER="$2"; shift 2 ;;
    --admin-pass) ADMIN_PASS="$2"; shift 2 ;;
    --public-url) PUBLIC_URL="$2"; shift 2 ;;
    --bin-url) BIN_URL="$2"; shift 2 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

if [ -z "$ADMIN_PASS" ]; then
  ADMIN_PASS="$AUTH_SECRET"
fi

if [ -z "$AUTH_SECRET" ]; then
  echo "usage: install-server-linux.sh --auth-secret SECRET [--admin-user admin] [--admin-pass PASSWORD] [--public-url https://monitor.example.com] [--bin-url URL]" >&2
  exit 2
fi

install -d /etc/vps-monitor /usr/local/bin /var/lib/vps-monitor
umask 077

if [ -n "$BIN_URL" ]; then
  TMP="$(mktemp)"
  curl -fsSL "$BIN_URL" -o "$TMP"
  install -m 0755 "$TMP" /usr/local/bin/vps-server
  rm -f "$TMP"
else
  if [ ! -x ./vps-server ]; then
    echo "./vps-server not found; pass --bin-url or run from a build directory" >&2
    exit 1
  fi
  install -m 0755 ./vps-server /usr/local/bin/vps-server
fi

cat >/etc/vps-monitor/server.env <<EOF
ADDR=:3000
AUTH_SECRET=$AUTH_SECRET
ADMIN_USER=$ADMIN_USER
ADMIN_PASS=$ADMIN_PASS
PUBLIC_URL=$PUBLIC_URL
DATA_PATH=/var/lib/vps-monitor/server.json
EOF
chmod 600 /etc/vps-monitor/server.env

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
echo "vps-server installed"
