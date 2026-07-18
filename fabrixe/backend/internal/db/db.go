package db

import (
	"database/sql"
	"fmt"
	"time"

	fabcrypto "github.com/fabrixe/fabrixe/internal/crypto"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// New opens (or creates) the SQLite database at the given path.
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	conn.SetMaxOpenConns(1) // SQLite: single writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Conn returns the underlying sql.DB (for use in handlers).
func (d *DB) Conn() *sql.DB {
	return d.conn
}

// Migrate runs all schema migrations in order.
func (d *DB) Migrate() error {
	migrations := []string{
		sqlCreateUsers,
		sqlCreateDevices,
		sqlCreateAuditLogs,
		sqlCreateSessions,
		sqlCreateScheduledTasks,
		sqlCreateDeployments,
		sqlCreateCommunicationNodes,
		sqlCreateSettings,
		sqlCreateAlerts,
	}
	for _, m := range migrations {
		if _, err := d.conn.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

// Bootstrap creates the default admin user on first run.
func (d *DB) Bootstrap() error {
	var count int
	_ = d.conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count > 0 {
		fmt.Println("[INFO] Users already exist, skipping bootstrap.")
		return nil
	}

	hash, err := fabcrypto.HashPassword("FabrixeAdmin@2024")
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	_, err = d.conn.Exec(`
		INSERT INTO users (username, email, password_hash, role, full_name, created_at, updated_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"admin",
		"admin@fabrixe.local",
		hash,
		"administrator",
		"Fabrixe Administrator",
		time.Now().UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
		1,
	)
	if err != nil {
		return fmt.Errorf("inserting admin user: %w", err)
	}

	// Insert default settings
	defaults := map[string]string{
		"fabrixe.node_name":          "Fabrixe Node",
		"fabrixe.organization":       "My Organization",
		"fabrixe.allow_registration": "false",
		"fabrixe.maintenance_mode":   "false",
	}
	for k, v := range defaults {
		_, _ = d.conn.Exec(`INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES (?, ?, ?)`,
			k, v, time.Now().UTC().Format(time.RFC3339))
	}

	fmt.Println("[INFO] Bootstrap complete. Admin user created.")
	return nil
}

// ─────────────────────────────────────────────
// Schema DDL
// ─────────────────────────────────────────────

const sqlCreateUsers = `
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    role          TEXT    NOT NULL DEFAULT 'viewer', -- administrator, operator, viewer
    full_name     TEXT    NOT NULL DEFAULT '',
    last_login    TEXT,
    failed_logins INTEGER NOT NULL DEFAULT 0,
    locked_until  TEXT,
    is_active     INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
);`

const sqlCreateSessions = `
CREATE TABLE IF NOT EXISTS sessions (
    id            TEXT    PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token TEXT    NOT NULL UNIQUE,
    ip_address    TEXT    NOT NULL,
    user_agent    TEXT    NOT NULL,
    created_at    TEXT    NOT NULL,
    expires_at    TEXT    NOT NULL,
    last_seen     TEXT    NOT NULL
);`

const sqlCreateDevices = `
CREATE TABLE IF NOT EXISTS devices (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT    NOT NULL,
    fingerprint   TEXT    NOT NULL UNIQUE,
    ip_address    TEXT,
    mac_address   TEXT,
    device_type   TEXT    NOT NULL DEFAULT 'workstation',
    is_trusted    INTEGER NOT NULL DEFAULT 0,
    first_seen    TEXT    NOT NULL,
    last_seen     TEXT    NOT NULL,
    added_by      INTEGER REFERENCES users(id),
    notes         TEXT    DEFAULT ''
);`

const sqlCreateAuditLogs = `
CREATE TABLE IF NOT EXISTS audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type  TEXT    NOT NULL,
    description TEXT    NOT NULL,
    user_id     INTEGER REFERENCES users(id),
    username    TEXT,
    ip_address  TEXT,
    resource    TEXT,
    outcome     TEXT    NOT NULL DEFAULT 'success', -- success, failure, warning
    metadata    TEXT    DEFAULT '{}',
    created_at  TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id    ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);`

const sqlCreateScheduledTasks = `
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    command      TEXT    NOT NULL,
    schedule     TEXT    NOT NULL, -- cron expression
    is_active    INTEGER NOT NULL DEFAULT 1,
    last_run     TEXT,
    last_status  TEXT    DEFAULT 'pending', -- pending, running, success, failed
    last_output  TEXT    DEFAULT '',
    next_run     TEXT,
    created_by   INTEGER REFERENCES users(id),
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL
);`

const sqlCreateDeployments = `
CREATE TABLE IF NOT EXISTS deployments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    deploy_type  TEXT    NOT NULL, -- script, docker, systemd
    config       TEXT    NOT NULL DEFAULT '{}',
    status       TEXT    NOT NULL DEFAULT 'idle', -- idle, running, success, failed
    last_run     TEXT,
    last_output  TEXT    DEFAULT '',
    created_by   INTEGER REFERENCES users(id),
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL
);`

const sqlCreateCommunicationNodes = `
CREATE TABLE IF NOT EXISTS communication_nodes (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id       TEXT    NOT NULL UNIQUE,
    display_name  TEXT    NOT NULL,
    endpoint      TEXT    NOT NULL,
    public_key    TEXT    NOT NULL,
    fingerprint   TEXT    NOT NULL UNIQUE,
    is_trusted    INTEGER NOT NULL DEFAULT 0,
    status        TEXT    NOT NULL DEFAULT 'unknown', -- online, offline, unknown
    last_seen     TEXT,
    added_by      INTEGER REFERENCES users(id),
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
);`

const sqlCreateSettings = `
CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);`

const sqlCreateAlerts = `
CREATE TABLE IF NOT EXISTS alerts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    level       TEXT    NOT NULL DEFAULT 'info', -- info, warning, critical
    source      TEXT    NOT NULL,
    message     TEXT    NOT NULL,
    is_resolved INTEGER NOT NULL DEFAULT 0,
    resolved_by INTEGER REFERENCES users(id),
    resolved_at TEXT,
    created_at  TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at   ON alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_is_resolved  ON alerts(is_resolved);`
