#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# Fabrixe v1.0.0 — Installation Script
# Supported: Ubuntu 20.04+, Debian 11+, RHEL/CentOS/AlmaLinux 8+, Fedora 36+
# Must be run as root.
# ──────────────────────────────────────────────────────────────────────────────

set -euo pipefail

# ── Colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'
info()    { echo -e "${BLUE}[INFO]${RESET} $*"; }
success() { echo -e "${GREEN}[OK]${RESET}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${RESET} $*"; }
error()   { echo -e "${RED}[ERROR]${RESET} $*" >&2; exit 1; }
banner()  { echo -e "${CYAN}${BOLD}$*${RESET}"; }

banner "
███████╗ █████╗ ██████╗ ██████╗ ██╗██╗  ██╗███████╗
██╔════╝██╔══██╗██╔══██╗██╔══██╗██║╚██╗██╔╝██╔════╝
█████╗  ███████║██████╔╝██████╔╝██║ ╚███╔╝ █████╗
██╔══╝  ██╔══██║██╔══██╗██╔══██╗██║ ██╔██╗ ██╔══╝
██║     ██║  ██║██████╔╝██║  ██║██║██╔╝ ██╗███████╗
╚═╝     ╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝╚══════╝
Installation Script v1.0.0
"

# ── Root check ────────────────────────────────────────────────────────────────
[[ $EUID -eq 0 ]] || error "This script must be run as root."

# ── Detect package manager ───────────────────────────────────────────────────
PKG_MGR=""
if command -v apt-get &>/dev/null; then PKG_MGR="apt"
elif command -v dnf &>/dev/null;   then PKG_MGR="dnf"
elif command -v yum &>/dev/null;   then PKG_MGR="yum"
else error "Unsupported distribution. Install manually."; fi

info "Detected package manager: $PKG_MGR"

# ── Install runtime dependencies ─────────────────────────────────────────────
info "Installing runtime dependencies..."
case "$PKG_MGR" in
  apt)
    apt-get update -qq
    apt-get install -y --no-install-recommends \
      ca-certificates libsqlite3-0 avahi-daemon libnss-mdns curl
    ;;
  dnf|yum)
    $PKG_MGR install -y ca-certificates sqlite avahi nss-mdns curl
    ;;
esac
success "Dependencies installed."

# ── Detect package directory ──────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
info "Installing from: $SCRIPT_DIR"

# Verify required files exist
[[ -f "$SCRIPT_DIR/bin/fabrixe" ]]        || error "bin/fabrixe not found in package."
[[ -d "$SCRIPT_DIR/web" ]]                 || error "web/ directory not found in package."
[[ -f "$SCRIPT_DIR/fabrixe.service" ]]    || error "fabrixe.service not found in package."

# ── Create system user ────────────────────────────────────────────────────────
if ! id fabrixe &>/dev/null; then
  info "Creating system user 'fabrixe'..."
  useradd --system --no-create-home --shell /usr/sbin/nologin fabrixe
  success "User created."
fi

# ── Create directories ────────────────────────────────────────────────────────
info "Creating directories..."
install -d -m 755 /opt/fabrixe/bin /opt/fabrixe/web
install -d -m 700 /var/lib/fabrixe
install -d -m 755 /etc/fabrixe/certs
success "Directories created."

# ── Install binary ────────────────────────────────────────────────────────────
info "Installing Fabrixe binary..."
install -m 755 "$SCRIPT_DIR/bin/fabrixe" /opt/fabrixe/bin/fabrixe
ln -sf /opt/fabrixe/bin/fabrixe /usr/local/bin/fabrixe
success "Binary installed at /opt/fabrixe/bin/fabrixe"

# ── Install web assets ────────────────────────────────────────────────────────
info "Installing web dashboard..."
cp -r "$SCRIPT_DIR/web/." /opt/fabrixe/web/
success "Web assets installed at /opt/fabrixe/web/"

# ── Install config ────────────────────────────────────────────────────────────
if [[ ! -f /etc/fabrixe/config.yaml ]]; then
  info "Installing default configuration..."
  install -m 640 "$SCRIPT_DIR/config.yaml" /etc/fabrixe/config.yaml
  chown root:fabrixe /etc/fabrixe/config.yaml
  success "Config installed at /etc/fabrixe/config.yaml"
else
  warn "Config already exists at /etc/fabrixe/config.yaml — skipping (your config preserved)."
fi

# ── Set ownership ─────────────────────────────────────────────────────────────
chown -R fabrixe:fabrixe /var/lib/fabrixe /etc/fabrixe/certs
# Config is owned root:fabrixe with mode 640 so the service user can read it
chown root:fabrixe /etc/fabrixe/config.yaml
chmod 640 /etc/fabrixe/config.yaml

# ── Install systemd service ───────────────────────────────────────────────────
info "Installing systemd service..."
install -m 644 "$SCRIPT_DIR/fabrixe.service" /etc/systemd/system/fabrixe.service
systemctl daemon-reload
success "Service installed."

# ── Configure mDNS ───────────────────────────────────────────────────────────
info "Configuring mDNS (for fabrixe.local)..."
case "$PKG_MGR" in
  apt)
    if ! grep -q "mdns4_minimal" /etc/nsswitch.conf 2>/dev/null; then
      sed -i 's/^hosts:.*/hosts:          files mdns4_minimal [NOTFOUND=return] dns myhostname/' /etc/nsswitch.conf
    fi
    systemctl enable --now avahi-daemon 2>/dev/null || true
    ;;
  dnf|yum)
    systemctl enable --now avahi-daemon 2>/dev/null || true
    ;;
esac
success "mDNS configured."

# ── Initialize Fabrixe ────────────────────────────────────────────────────────
info "Initializing Fabrixe database and admin account..."
/opt/fabrixe/bin/fabrixe -config /etc/fabrixe/config.yaml -init

# The -init run as root creates files in /var/lib/fabrixe; re-chown so the
# fabrixe service user owns them.
chown -R fabrixe:fabrixe /var/lib/fabrixe

# ── Enable and start service ──────────────────────────────────────────────────
info "Enabling and starting Fabrixe service..."
systemctl enable fabrixe
systemctl start fabrixe

# Wait for it to come up
sleep 3
if systemctl is-active --quiet fabrixe; then
  success "Fabrixe is running!"
else
  warn "Fabrixe may not have started. Check: journalctl -u fabrixe -n 50"
fi

# ── Detect LAN IP ─────────────────────────────────────────────────────────────
LAN_IP=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K[\d.]+' || echo "your-server-ip")

# ── Done ─────────────────────────────────────────────────────────────────────
echo ""
banner "════════════════════════════════════════════"
echo -e "${GREEN}${BOLD}Fabrixe installed successfully!${RESET}"
banner "════════════════════════════════════════════"
echo ""
echo -e "  Dashboard (LAN):   ${CYAN}https://fabrixe.local${RESET}"
echo -e "  Dashboard (IP):    ${CYAN}https://${LAN_IP}:8443${RESET}"
echo ""
echo -e "  Default login:     ${BOLD}admin${RESET} / ${BOLD}FabrixeAdmin@2024${RESET}"
echo -e "  ${RED}${BOLD}Change this password immediately after first login!${RESET}"
echo ""
echo "  TLS certificate:   Self-signed — accept in your browser or import"
echo "                     /etc/fabrixe/certs/fabrixe.crt as a trusted CA."
echo ""
echo "  Service commands:"
echo "    systemctl status fabrixe"
echo "    systemctl restart fabrixe"
echo "    journalctl -u fabrixe -f"
echo ""
echo "  Config file:       /etc/fabrixe/config.yaml"
echo "  Data directory:    /var/lib/fabrixe/"
echo "  Certs directory:   /etc/fabrixe/certs/"
echo ""
