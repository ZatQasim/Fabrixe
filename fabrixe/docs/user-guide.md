# Fabrixe — User Guide

## Overview

Fabrixe is a self-hosted, browser-based infrastructure management platform. After installing it on your Linux server, every authorized user on your local network can open `https://fabrixe.local` and manage the server from their browser — no installed software required.

---

## Accessing Fabrixe

| URL | When to use |
|-----|-------------|
| `https://fabrixe.local` | Standard LAN access (requires mDNS) |
| `https://<server-ip>:8443` | Direct IP access / Windows clients |

Accept the TLS certificate warning on first visit (or import the CA cert — see the Installation Guide).

---

## Roles & Permissions

| Role | Description |
|------|-------------|
| **Administrator** | Full access: all modules, user management, settings |
| **Operator** | Can view everything and perform operational actions (start/stop services, run deployments, manage tasks). Cannot manage users or change security settings. |
| **Viewer** | Read-only access to monitoring data and dashboards. |

---

## Module 1: Dashboard

The main dashboard gives you an at-a-glance health summary:

- **Live CPU & Memory charts** — updates every 2 seconds via WebSocket
- **Uptime & load averages**
- **Storage usage** — all mounted filesystems
- **Network interfaces** — bytes transferred, errors
- **Per-core CPU breakdown**
- **Active alerts**

The live indicator (top-right) shows whether the real-time feed is connected.

---

## Module 2: System Management

### Overview Tab
Detailed real-time metrics:
- CPU usage with per-core breakdown and frequency
- Full memory breakdown (used, free, cached, buffers, swap)
- All mounted filesystems with usage bars
- Kernel, OS, architecture, process count

### Services Tab
Lists all systemd services on the server.

**Available actions** (Operator+ required):
- **▶ Start** — start a stopped service
- **■ Stop** — stop a running service
- **↺ Restart** — restart a service

Use the filter box to find a specific service. Output from service actions is displayed inline.

### Alerts Tab
View and resolve system alerts. Alerts are generated automatically when thresholds are crossed.

- **Info** (blue) — informational events
- **Warning** (amber) — potential issues requiring attention
- **Critical** (red) — immediate action required

Click **Resolve** (Operator+) to mark an alert as resolved.

### Hardware Tab
Reads hardware information from `/sys` and `/proc`:
- Motherboard vendor and product
- CPU model
- Installed RAM
- Detected storage devices

---

## Module 3: Deployment & Automation

### Deployments

A Deployment is a named, runnable action tied to a specific mechanism:

| Type | What it does |
|------|-------------|
| **script** | Runs an arbitrary shell script |
| **docker** | Runs `docker <args>` (Docker must be installed) |
| **systemd** | Runs `systemctl restart <service>` |

**Creating a Deployment:**
1. Click **New Deployment**
2. Choose the type
3. Enter the command/script content
4. Click **Create**

**Running a Deployment:**
- Click **Run** — the deployment executes in the background
- Click **View output** to see stdout/stderr from the last run
- Status updates automatically

### Scheduled Tasks

A Scheduled Task runs a shell command on a cron schedule.

**Schedule format:** Standard 5-field cron: `minute hour day month weekday`

Common examples:
| Schedule | When |
|----------|------|
| `0 2 * * *` | Every day at 2:00 AM |
| `*/15 * * * *` | Every 15 minutes |
| `0 9 * * 1` | Every Monday at 9:00 AM |
| `0 0 1 * *` | First day of every month |

**Run now:** Click **Run now** to execute a task immediately, outside of its schedule.

---

## Module 4: Internal Security

### Users

Manage Fabrixe user accounts.

**Creating a user:**
1. Click **New User**
2. Fill in username, email, password (min 10 characters)
3. Select a role
4. Click **Create User**

**Disabling a user:** Click **Disable** — the user cannot log in but their data is preserved.

**Unlocking a user:** If a user has been locked out after too many failed login attempts, an Administrator can click **Unlock** to restore access immediately.

**Deleting a user:** Permanently removes the account. You cannot delete your own account.

### Audit Logs

Every significant action is recorded with:
- Timestamp
- Event type (e.g., `auth.login`, `security.user_created`)
- Description
- Username and IP address
- Outcome (success/failure)

Use the filter box to search by event type.

**Common event types:**
| Event | Description |
|-------|-------------|
| `auth.login` | Successful login |
| `auth.login_failed` | Failed login attempt |
| `auth.logout` | User logout |
| `security.user_created` | New user created |
| `security.password_changed` | Password change |
| `security.session_revoked` | Session terminated |
| `system.service_restart` | Service restarted |
| `deployment.run_started` | Deployment triggered |
| `comm.node_trusted` | Communication node trusted |

### Devices

Fabrixe tracks devices that have logged in. Each login records the device's browser fingerprint and IP address.

- **Trust** — mark a device as authorized
- **Revoke** — remove trusted status
- **Delete** — remove the device record

> Note: Fabrixe does not enforce device-level access control by default. The device list is informational. Future versions will support mandatory device trust.

### Sessions

Lists all active login sessions. Administrators can revoke any session, immediately invalidating the user's tokens.

---

## Module 5: Protected Communication

### How it works

Each Fabrixe installation has a unique identity consisting of:
- A **Node ID** (UUID)
- An **ECDSA P-256 key pair**
- A **fingerprint** (first 16 bytes of SHA-256 of the public key)

To establish a trusted connection between two Fabrixe nodes:

1. Go to **Protected Communication → Node Identity** on **Node A**
2. Copy Node A's public key
3. On **Node B**, go to **Protected Communication → Nodes → Add Node**
4. Enter Node A's endpoint (e.g., `https://192.168.1.50:8443`) and paste the public key
5. After adding, an Administrator on Node B must click **Trust** to authorize the node

Repeat the process in reverse for bidirectional trust.

### Pinging a node

Click **Ping** to test connectivity to a remote node. Fabrixe will make an HTTPS request to the node's `/api/health` endpoint and report:
- Status (online/offline)
- Round-trip latency in milliseconds

### Node Identity page

Displays this node's:
- Node ID and hostname
- Fingerprint (share this verbally to verify the public key)
- Full public key (copy to paste into peer nodes)

---

## Security Best Practices

1. **Change the admin password** on first login
2. **Use strong passwords** — minimum 10 characters, mix of letters, numbers, symbols
3. **Review audit logs** weekly
4. **Use the Operator role** for day-to-day operations — reserve Administrator for sensitive changes
5. **Revoke inactive sessions** regularly
6. **Keep Fabrixe updated** — watch for new releases
7. **Restrict LAN access** — Fabrixe is LAN-only by default; do not expose port 8443 to the internet
8. **Review device list** — remove devices you don't recognize

---

## Troubleshooting

### Cannot access https://fabrixe.local

1. Verify Fabrixe is running: `systemctl status fabrixe`
2. Verify avahi-daemon is running: `systemctl status avahi-daemon`
3. Check your client has `libnss-mdns`: `getent hosts fabrixe.local`
4. Use the IP address instead: `https://<server-ip>:8443`

### Browser shows "Connection refused"

1. Check the service is running: `systemctl status fabrixe`
2. Check logs: `journalctl -u fabrixe -n 50`
3. Verify the port is open in your firewall

### Account locked

Contact an Administrator — they can unlock your account from Internal Security → Users → Unlock.

### TLS certificate error

The self-signed certificate is expected. Accept it in your browser, or install it as a trusted CA (see Installation Guide).

### Dashboard shows "Reconnecting…"

The WebSocket connection to the server was lost. Fabrixe will reconnect automatically. If it persists, check `systemctl status fabrixe`.
