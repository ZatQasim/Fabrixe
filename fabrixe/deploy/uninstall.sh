#!/usr/bin/env bash
# Fabrixe Uninstall Script
set -euo pipefail

[[ $EUID -eq 0 ]] || { echo "Run as root."; exit 1; }

echo "This will remove Fabrixe binaries and service files."
echo "Your data (/var/lib/fabrixe) and config (/etc/fabrixe) will NOT be deleted."
read -p "Continue? [y/N] " -r; [[ $REPLY =~ ^[Yy]$ ]] || exit 0

systemctl stop fabrixe   2>/dev/null || true
systemctl disable fabrixe 2>/dev/null || true
rm -f /etc/systemd/system/fabrixe.service
systemctl daemon-reload

rm -rf /opt/fabrixe
rm -f /usr/local/bin/fabrixe

echo ""
echo "Fabrixe removed. Data preserved:"
echo "  /var/lib/fabrixe/"
echo "  /etc/fabrixe/"
echo ""
echo "To remove all data: rm -rf /var/lib/fabrixe /etc/fabrixe"
