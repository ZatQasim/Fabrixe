# Fabrixe — Installation Guide

## Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| OS | Linux (kernel 4.15+) | Ubuntu 22.04 LTS / Debian 12 |
| RAM | 256 MB | 512 MB+ |
| Disk | 500 MB | 2 GB+ |
| Network | LAN connection | Gigabit LAN |
| CPU | Any x86_64 | x86_64 |

Supported distributions:
- Ubuntu 20.04, 22.04, 24.04
- Debian 11 (Bullseye), 12 (Bookworm)
- RHEL 8/9, AlmaLinux 8/9, Rocky Linux 8/9
- Fedora 36+

---

## Option 1: Install from Release Package (Recommended)

### Step 1: Download

```bash
wget https://github.com/fabrixe/fabrixe/releases/download/v1.0.0/Fabrixe-v1.0.0.tar.gz
```

Or copy the package to your server:
```bash
scp Fabrixe-v1.0.0.tar.gz admin@192.168.1.50:/tmp/
```

### Step 2: Extract

```bash
cd /tmp
tar -xzf Fabrixe-v1.0.0.tar.gz
cd fabrixe-1.0.0
```

### Step 3: Run the installer

```bash
sudo bash install.sh
```

The installer will:
1. Install runtime dependencies (SQLite, avahi-daemon for mDNS)
2. Create the `fabrixe` system user
3. Install the binary to `/opt/fabrixe/bin/`
4. Install the web dashboard to `/opt/fabrixe/web/`
5. Install the configuration to `/etc/fabrixe/config.yaml`
6. Install and start the systemd service
7. Generate a self-signed TLS certificate
8. Create the default admin account

### Step 4: First login

Open your browser and go to:

```
https://fabrixe.local
```

> **Note:** Your browser will warn about the self-signed certificate. This is expected.
> Accept the certificate or add it as a trusted CA (see below).

Default credentials:
- Username: `admin`
- Password: `FabrixeAdmin@2024`

**Change this password immediately** in the dashboard under Internal Security → Users.

---

## Option 2: Docker / Container

### Single container

```bash
# First-time initialization
docker run --rm \
  -v fabrixe-data:/var/lib/fabrixe \
  -v fabrixe-config:/etc/fabrixe \
  fabrixe:1.0.0 \
  -config /etc/fabrixe/config.yaml -init

# Start the service
docker run -d \
  --name fabrixe \
  --network host \
  -v fabrixe-data:/var/lib/fabrixe \
  -v fabrixe-config:/etc/fabrixe \
  --restart unless-stopped \
  fabrixe:1.0.0
```

### Docker Compose

```bash
# Initialize
docker compose run --rm fabrixe -config /etc/fabrixe/config.yaml -init

# Start
docker compose up -d
```

> **Important:** `--network host` is required for mDNS (`fabrixe.local`) to work.

---

## Option 3: Build from Source

### Prerequisites

```bash
# Install Go 1.21+
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install Node.js 20+
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install SQLite development headers (for CGO)
sudo apt-get install -y gcc libsqlite3-dev
```

### Build

```bash
git clone https://github.com/fabrixe/fabrixe.git
cd fabrixe

# Download Go dependencies
cd backend && go mod download && cd ..

# Build everything
make build

# Create the release package
make package
# → release/Fabrixe-v1.0.0.tar.gz
```

---

## TLS Certificate

Fabrixe generates a self-signed ECDSA P-256 certificate on first start, covering:
- `fabrixe.local`
- `localhost`
- All detected LAN IP addresses

The certificate lives at `/etc/fabrixe/certs/fabrixe.crt`.

### Trusting the certificate (so the browser doesn't warn)

**Ubuntu/Debian:**
```bash
sudo cp /etc/fabrixe/certs/fabrixe.crt /usr/local/share/ca-certificates/fabrixe.crt
sudo update-ca-certificates
```

**RHEL/Fedora:**
```bash
sudo cp /etc/fabrixe/certs/fabrixe.crt /etc/pki/ca-trust/source/anchors/fabrixe.crt
sudo update-ca-trust
```

**Browser (Chrome/Firefox):**
Navigate to `https://fabrixe.local`, click "Advanced" → "Proceed" (Chrome) or "Accept Risk" (Firefox). Alternatively, import the cert via browser settings → Certificates.

---

## Local DNS Resolution (fabrixe.local)

Fabrixe uses mDNS (Multicast DNS / Bonjour / Avahi) to advertise `fabrixe.local` on your LAN.

### Linux clients
Ensure `avahi-daemon` and `libnss-mdns` are installed:
```bash
sudo apt-get install -y avahi-daemon libnss-mdns
```

### macOS clients
Works out of the box — mDNS is built into macOS.

### Windows clients
Install Bonjour (included with iTunes) or use the IP address directly:
```
https://192.168.1.50:8443
```

---

## Service Management

```bash
# Status
systemctl status fabrixe

# View live logs
journalctl -u fabrixe -f

# Restart
systemctl restart fabrixe

# Stop
systemctl stop fabrixe

# Disable autostart
systemctl disable fabrixe
```

---

## Firewall Configuration

If you run `ufw` or `firewalld`, allow the Fabrixe ports:

**UFW:**
```bash
sudo ufw allow 443/tcp comment "Fabrixe HTTPS"
sudo ufw allow 8443/tcp comment "Fabrixe HTTPS alt"
sudo ufw allow 80/tcp comment "Fabrixe HTTP redirect"
```

**firewalld:**
```bash
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --reload
```

---

## Configuration Reference

All settings are in `/etc/fabrixe/config.yaml`. Key settings:

| Setting | Default | Description |
|---------|---------|-------------|
| `server.host` | `0.0.0.0` | Bind address |
| `server.port` | `8443` | HTTPS port |
| `server.http_redirect_port` | `80` | HTTP→HTTPS redirect |
| `database.path` | `/var/lib/fabrixe/fabrixe.db` | SQLite DB path |
| `security.max_login_attempts` | `5` | Failed logins before lockout |
| `security.lockout_minutes` | `15` | Lockout duration |
| `mdns.hostname` | `fabrixe` | → `fabrixe.local` |

After editing the config, restart the service:
```bash
systemctl restart fabrixe
```

---

## Directories

| Path | Purpose |
|------|---------|
| `/opt/fabrixe/bin/` | Binary |
| `/opt/fabrixe/web/` | Dashboard web assets |
| `/etc/fabrixe/config.yaml` | Configuration |
| `/etc/fabrixe/certs/` | TLS certificates |
| `/var/lib/fabrixe/fabrixe.db` | Database |
| `/var/lib/fabrixe/.jwt_secret` | Auto-generated JWT secret |
| `/var/lib/fabrixe/.node_id` | Node identity UUID |
| `/var/lib/fabrixe/node_key.pem` | Node private key |
| `/var/lib/fabrixe/node_pub.pem` | Node public key |

---

## Uninstallation

```bash
sudo bash /tmp/fabrixe-1.0.0/uninstall.sh
```

To also remove all data:
```bash
sudo rm -rf /var/lib/fabrixe /etc/fabrixe
```
