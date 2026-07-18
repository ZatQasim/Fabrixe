# Fabrixe — Developer Guide

## Architecture

```
fabrixe/
├── backend/                    Go backend service
│   ├── cmd/fabrixe/main.go     Entry point
│   ├── internal/
│   │   ├── api/                HTTP router + request routing
│   │   │   ├── router.go       gorilla/mux router, auth middleware
│   │   │   └── handlers/       One file per module
│   │   ├── auth/               JWT issuance & validation
│   │   ├── config/             YAML config loading
│   │   ├── db/                 SQLite setup, migrations, bootstrap
│   │   ├── mdns/               mDNS advertisement (fabrixe.local)
│   │   ├── modules/
│   │   │   └── system/         Real /proc reader (CPU, memory, disk, net)
│   │   └── tls/                Self-signed certificate generator
│   └── pkg/
│       ├── middleware/         HTTP middleware (auth, CORS, rate limit, logging)
│       └── models/             Shared data types (JSON-serialisable)
│
├── frontend/                   React + TypeScript dashboard
│   ├── src/
│   │   ├── context/            React contexts (Auth)
│   │   ├── components/
│   │   │   ├── layout/         Sidebar, Header, Layout wrapper
│   │   │   └── ui/             Shared UI components
│   │   ├── hooks/              useApi, useWebSocket, useAuth
│   │   ├── lib/                API client, WebSocket client, formatters
│   │   ├── pages/              One file per route (Dashboard, System, …)
│   │   └── types/              TypeScript interfaces mirroring Go models
│   ├── vite.config.ts          Build config; proxies /api to localhost:8443
│   └── tailwind.config.js      Design system tokens
│
├── deploy/                     Deployment artifacts
│   ├── config.yaml             Default configuration
│   ├── fabrixe.service         systemd unit file
│   ├── install.sh              Universal installer
│   └── uninstall.sh            Removal script
│
├── docs/                       Documentation
├── Makefile                    Build, package, install targets
└── Dockerfile                  Multi-stage container build
```

---

## Development Setup

### Prerequisites

```bash
# Go 1.21+
go version

# Node.js 20+
node --version

# C compiler (for CGO / SQLite)
gcc --version

# SQLite headers
sudo apt-get install -y libsqlite3-dev    # Debian/Ubuntu
sudo dnf install -y sqlite-devel          # RHEL/Fedora
```

### Running locally

**Backend (terminal 1):**
```bash
cd fabrixe/backend
go mod download
mkdir -p /tmp/fabrixe-dev/certs /tmp/fabrixe-dev/db

# Create a minimal dev config
cat > /tmp/fabrixe-dev.yaml <<EOF
server:
  host: "0.0.0.0"
  port: 8443
  static_dir: "/tmp/fabrixe-web"
database:
  path: "/tmp/fabrixe-dev/db/fabrixe.db"
tls:
  cert_dir: "/tmp/fabrixe-dev/certs"
  hostnames: ["localhost"]
jwt:
  access_token_ttl_minutes: 60
  refresh_token_ttl_days: 7
security:
  rate_limit_per_minute: 300
  max_login_attempts: 10
  lockout_minutes: 1
mdns:
  hostname: "fabrixe"
EOF

go run ./cmd/fabrixe -config /tmp/fabrixe-dev.yaml -init
go run ./cmd/fabrixe -config /tmp/fabrixe-dev.yaml
```

**Frontend (terminal 2):**
```bash
cd fabrixe/frontend
npm install
npm run dev
# Vite starts at http://localhost:5173 and proxies /api to https://localhost:8443
```

---

## API Reference

### Authentication

All endpoints except `POST /api/auth/login` and `GET /api/health` require:

```
Authorization: Bearer <access_token>
```

Tokens expire after 60 minutes by default. Use `POST /api/auth/refresh` with the `refresh_token` to get a new pair.

### Response envelope

All API responses use a consistent envelope:

```json
{
  "success": true,
  "data": { ... }
}
```

On error:
```json
{
  "success": false,
  "error": "description"
}
```

### Endpoint map

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/health` | None | Service health check |
| POST | `/api/auth/login` | None | Login → token pair |
| POST | `/api/auth/logout` | Any | Invalidate refresh token |
| POST | `/api/auth/refresh` | None | Refresh token pair |
| GET | `/api/auth/me` | Any | Current user info |
| POST | `/api/auth/change-password` | Any | Change own password |
| GET | `/api/system/snapshot` | Any | Full system metrics snapshot |
| GET | `/api/system/hardware` | Any | Hardware information |
| GET | `/api/system/services` | Any | systemd service list |
| POST | `/api/system/services/{name}/{action}` | Operator | start/stop/restart/status |
| GET | `/api/system/alerts` | Any | Alert list |
| POST | `/api/system/alerts/{id}/resolve` | Operator | Resolve an alert |
| WS | `/api/system/ws?token=<jwt>` | Any | Live snapshot stream (2s interval) |
| GET | `/api/deployment` | Any | List deployments |
| POST | `/api/deployment` | Operator | Create deployment |
| GET | `/api/deployment/{id}` | Any | Get deployment |
| PUT | `/api/deployment/{id}` | Operator | Update deployment |
| DELETE | `/api/deployment/{id}` | Admin | Delete deployment |
| POST | `/api/deployment/{id}/run` | Operator | Trigger deployment |
| GET | `/api/deployment/tasks` | Any | List tasks |
| POST | `/api/deployment/tasks` | Operator | Create task |
| PUT | `/api/deployment/tasks/{id}` | Operator | Update task |
| DELETE | `/api/deployment/tasks/{id}` | Admin | Delete task |
| POST | `/api/deployment/tasks/{id}/run` | Operator | Run task now |
| GET | `/api/security/users` | Admin | List users |
| POST | `/api/security/users` | Admin | Create user |
| PUT | `/api/security/users/{id}` | Admin | Update user |
| DELETE | `/api/security/users/{id}` | Admin | Delete user |
| POST | `/api/security/users/{id}/unlock` | Admin | Unlock user |
| GET | `/api/security/audit-logs` | Operator | Audit log query |
| GET | `/api/security/devices` | Any | Device list |
| POST | `/api/security/devices/{id}/trust` | Admin | Trust device |
| POST | `/api/security/devices/{id}/revoke` | Admin | Revoke device |
| DELETE | `/api/security/devices/{id}` | Admin | Delete device |
| GET | `/api/security/sessions` | Admin | Session list |
| DELETE | `/api/security/sessions/{id}` | Admin | Revoke session |
| GET | `/api/security/settings` | Admin | Get settings |
| PUT | `/api/security/settings` | Admin | Update settings |
| GET | `/api/comm/identity` | Any | This node's identity |
| GET | `/api/comm/nodes` | Any | List nodes |
| POST | `/api/comm/nodes` | Operator | Add node |
| GET | `/api/comm/nodes/{id}` | Any | Get node |
| POST | `/api/comm/nodes/{id}/trust` | Admin | Trust node |
| POST | `/api/comm/nodes/{id}/revoke` | Admin | Revoke node |
| DELETE | `/api/comm/nodes/{id}` | Admin | Delete node |
| POST | `/api/comm/ping/{id}` | Operator | Ping node |

---

## Database Schema

Fabrixe uses SQLite with WAL mode for high read concurrency. Tables:

| Table | Purpose |
|-------|---------|
| `users` | User accounts and credentials |
| `sessions` | Active refresh token sessions |
| `devices` | Tracked client devices |
| `audit_logs` | Immutable event log |
| `scheduled_tasks` | Cron-style automation tasks |
| `deployments` | Named deployment configurations |
| `communication_nodes` | Peer Fabrixe node registry |
| `settings` | Key-value configuration store |
| `alerts` | System alerts |

Migrations run automatically on startup. To view the current schema:
```bash
sqlite3 /var/lib/fabrixe/fabrixe.db .schema
```

---

## Adding a New API Endpoint

1. **Add handler method** to the appropriate handler file in `internal/api/handlers/`
2. **Register the route** in `internal/api/router.go`
3. **Add types** to `pkg/models/models.go` if needed
4. **Update frontend API client** in `frontend/src/lib/api.ts`
5. **Update TypeScript types** in `frontend/src/types/index.ts`
6. **Add UI** in the appropriate page component

---

## Adding a New Module

1. Create `internal/modules/<name>/` with the module's business logic
2. Create `internal/api/handlers/<name>.go` with HTTP handlers
3. Register routes in `router.go` under `/api/<name>/`
4. Add navigation item in `frontend/src/components/layout/Sidebar.tsx`
5. Create `frontend/src/pages/<Name>.tsx`
6. Add route to `frontend/src/App.tsx`

---

## Building a Release Package

```bash
cd fabrixe

# Full build + package
make release

# Output: release/Fabrixe-v1.0.0.tar.gz
```

The package includes:
```
fabrixe-1.0.0/
├── bin/fabrixe          Linux x86_64 binary (statically linked)
├── web/                 Pre-built React dashboard
├── config.yaml          Default configuration
├── fabrixe.service      systemd unit
├── install.sh           Installer
├── uninstall.sh         Uninstaller
└── docs/
    ├── installation.md
    ├── user-guide.md
    └── developer-guide.md
```

---

## Security Design Decisions

| Decision | Rationale |
|----------|-----------|
| ECDSA P-256 TLS + node keys | Modern, compact, hardware-accelerable |
| JWT HS256 for API auth | Stateless verification; sessions tracked via refresh token |
| bcrypt for passwords | Industry standard, adjustable cost factor |
| SQLite WAL mode | ACID, no separate DB server, portable |
| Rate limiting per IP | Defense against brute-force login |
| Account lockout | Limits credential stuffing attacks |
| LAN-only by default | Reduces internet attack surface |
| Self-signed cert | No dependency on external CA or internet |

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes, add tests
4. Run `make test` to verify
5. Submit a pull request

### Code style

- **Go:** `gofmt`, `golangci-lint`
- **TypeScript:** Prettier + ESLint (config in `frontend/`)
- Commit messages: `<module>: <present-tense description>`

---

## License

Fabrixe is open-source software licensed under the MIT License.
