#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "please run as root: sudo sh $0" >&2
  exit 1
fi

KEEP_DATA=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --keep-data) KEEP_DATA="1"; shift ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

systemctl disable --now vps-server 2>/dev/null || true
systemctl stop vps-server 2>/dev/null || true
pkill -f '/usr/local/bin/vps-server' 2>/dev/null || true
sleep 1
rm -f /etc/systemd/system/vps-server.service
rm -f /usr/local/bin/vps-server 2>/dev/null || true
if [ -e /usr/local/bin/vps-server ]; then
  chattr -i /usr/local/bin/vps-server 2>/dev/null || true
  rm -f /usr/local/bin/vps-server 2>/dev/null || true
fi
if [ -e /usr/local/bin/vps-server ]; then
  echo "failed to remove /usr/local/bin/vps-server" >&2
  ls -l /usr/local/bin/vps-server >&2 || true
  exit 1
fi
hash -r 2>/dev/null || true
rm -rf /etc/vps-monitor

if [ -z "$KEEP_DATA" ]; then
  rm -rf /var/lib/vps-monitor
else
  echo "kept data directory: /var/lib/vps-monitor"
fi

systemctl daemon-reload 2>/dev/null || true
systemctl reset-failed vps-server 2>/dev/null || true
echo "vps-server uninstalled"
