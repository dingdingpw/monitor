#!/usr/bin/env sh
set -eu

AUTH_SECRET=""
ADMIN_USER="admin"
ADMIN_PASS=""
PUBLIC_URL=""
BIN_URL=""

random_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
    return
  fi
  dd if=/dev/urandom bs=32 count=1 2>/dev/null | od -An -tx1 | tr -d ' \n'
}

is_weak_secret() {
  value="$1"
  [ -z "$value" ] || [ "$value" = "change-me" ]
}

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

if is_weak_secret "$AUTH_SECRET"; then
  AUTH_SECRET="$(random_secret)"
  GENERATED_AUTH_SECRET="1"
else
  GENERATED_AUTH_SECRET="0"
fi

if is_weak_secret "$ADMIN_PASS"; then
  ADMIN_PASS="$(random_secret)"
  GENERATED_ADMIN_PASS="1"
else
  GENERATED_ADMIN_PASS="0"
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
if [ "$GENERATED_AUTH_SECRET" = "1" ]; then
  echo "generated internal AUTH_SECRET in /etc/vps-monitor/server.env"
fi
if [ "$GENERATED_ADMIN_PASS" = "1" ]; then
  echo "generated admin login: $ADMIN_USER / $ADMIN_PASS"
fi
