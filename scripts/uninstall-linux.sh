#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "please run as root: sudo sh $0" >&2
  exit 1
fi

systemctl disable --now vps-agent 2>/dev/null || true
rm -f /etc/systemd/system/vps-agent.service /usr/local/bin/vps-agent
rm -rf /etc/vps-agent
systemctl daemon-reload 2>/dev/null || true
echo "vps-agent uninstalled"
