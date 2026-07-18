package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	fabcrypto "github.com/fabrixe/fabrixe/internal/crypto"
)

// ─────────────────────────────────────────────
// Users
// ─────────────────────────────────────────────

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Conn().Query(`
		SELECT id, username, email, role, full_name, last_login, is_active, failed_logins, locked_until, created_at, updated_at
		FROM users ORDER BY username`)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int64
		var username, email, role, fullName, createdAt, updatedAt string
		var isActive, failedLogins int
		var lastLogin, lockedUntil sql.NullString
		if err := rows.Scan(&id, &username, &email, &role, &fullName, &lastLogin, &isActive, &failedLogins, &lockedUntil, &createdAt, &updatedAt); err != nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"id":            id,
			"username":      username,
			"email":         email,
			"role":          role,
			"full_name":     fullName,
			"last_login":    lastLogin.String,
			"is_active":     isActive == 1,
			"failed_logins": failedLogins,
			"locked_until":  lockedUntil.String,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		})
	}
	if users == nil {
		users = []map[string]interface{}{}
	}
	ok(w, users)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid user id")
		return
	}
	var username, email, role, fullName, createdAt, updatedAt string
	var isActive, failedLogins int
	var lastLogin, lockedUntil sql.NullString
	err = h.db.Conn().QueryRow(`
		SELECT username, email, role, full_name, last_login, is_active, failed_logins, locked_until, created_at, updated_at
		FROM users WHERE id = ?`, id).
		Scan(&username, &email, &role, &fullName, &lastLogin, &isActive, &failedLogins, &lockedUntil, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		notFound(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	ok(w, map[string]interface{}{
		"id":            id,
		"username":      username,
		"email":         email,
		"role":          role,
		"full_name":     fullName,
		"last_login":    lastLogin.String,
		"is_active":     isActive == 1,
		"failed_logins": failedLogins,
		"locked_until":  lockedUntil.String,
		"created_at":    createdAt,
		"updated_at":    updatedAt,
	})
}

type createUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req createUserRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		badRequest(w, "username, email and password are required")
		return
	}
	if len(req.Password) < 10 {
		badRequest(w, "password must be at least 10 characters")
		return
	}
	validRoles := map[string]bool{"administrator": true, "operator": true, "viewer": true}
	if !validRoles[req.Role] {
		req.Role = "viewer"
	}

	hash, err := fabcrypto.HashPassword(req.Password)
	if err != nil {
		internalError(w, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := h.db.Conn().Exec(`
		INSERT INTO users (username, email, password_hash, role, full_name, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
		req.Username, req.Email, hash, req.Role, req.FullName, now, now)
	if err != nil {
		fail(w, http.StatusConflict, "username or email already exists")
		return
	}
	newID, _ := res.LastInsertId()

	h.writeAuditLog("security.user_created",
		fmt.Sprintf("user %q (role: %s) created by %s", req.Username, req.Role, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")

	created(w, map[string]interface{}{"id": newID, "username": req.Username, "role": req.Role})
}

type updateUserRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
	IsActive *bool  `json:"is_active"`
	Password string `json:"password"`
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid user id")
		return
	}
	var req updateUserRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)

	if req.Password != "" {
		if len(req.Password) < 10 {
			badRequest(w, "password must be at least 10 characters")
			return
		}
		hash, _ := fabcrypto.HashPassword(req.Password)
		_, _ = h.db.Conn().Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
			hash, now, id)
	}

	if req.Email != "" {
		_, _ = h.db.Conn().Exec(`UPDATE users SET email = ?, updated_at = ? WHERE id = ?`, req.Email, now, id)
	}
	if req.Role != "" {
		validRoles := map[string]bool{"administrator": true, "operator": true, "viewer": true}
		if validRoles[req.Role] {
			_, _ = h.db.Conn().Exec(`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, req.Role, now, id)
		}
	}
	if req.FullName != "" {
		_, _ = h.db.Conn().Exec(`UPDATE users SET full_name = ?, updated_at = ? WHERE id = ?`, req.FullName, now, id)
	}
	if req.IsActive != nil {
		v := 0
		if *req.IsActive {
			v = 1
		}
		_, _ = h.db.Conn().Exec(`UPDATE users SET is_active = ?, updated_at = ? WHERE id = ?`, v, now, id)
	}

	h.writeAuditLog("security.user_updated", fmt.Sprintf("user id=%d updated by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "user updated"})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid user id")
		return
	}
	if id == claims.UserID {
		badRequest(w, "cannot delete your own account")
		return
	}
	_, _ = h.db.Conn().Exec(`DELETE FROM users WHERE id = ?`, id)
	h.writeAuditLog("security.user_deleted", fmt.Sprintf("user id=%d deleted by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "user deleted"})
}

func (h *Handler) UnlockUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid user id")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE users SET locked_until = NULL, failed_logins = 0, updated_at = ? WHERE id = ?`, now, id)
	h.writeAuditLog("security.user_unlocked", fmt.Sprintf("user id=%d unlocked by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "user unlocked"})
}

// ─────────────────────────────────────────────
// Audit logs
// ─────────────────────────────────────────────

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 100)
	offset := queryInt(r, "offset", 0)
	eventType := queryStr(r, "event_type")
	username := queryStr(r, "username")

	query := `SELECT id, event_type, description, user_id, username, ip_address, resource, outcome, metadata, created_at FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	if eventType != "" {
		query += ` AND event_type LIKE ?`
		args = append(args, "%"+eventType+"%")
	}
	if username != "" {
		query += ` AND username = ?`
		args = append(args, username)
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.db.Conn().Query(query, args...)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int64
		var eventType, description, outcome, createdAt string
		var userID sql.NullInt64
		var username, ipAddress, resource, metadata sql.NullString
		if err := rows.Scan(&id, &eventType, &description, &userID, &username, &ipAddress, &resource, &outcome, &metadata, &createdAt); err != nil {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"id":          id,
			"event_type":  eventType,
			"description": description,
			"user_id":     userID.Int64,
			"username":    username.String,
			"ip_address":  ipAddress.String,
			"resource":    resource.String,
			"outcome":     outcome,
			"metadata":    metadata.String,
			"created_at":  createdAt,
		})
	}
	if logs == nil {
		logs = []map[string]interface{}{}
	}
	ok(w, logs)
}

// ─────────────────────────────────────────────
// Devices
// ─────────────────────────────────────────────

func (h *Handler) ListDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Conn().Query(`
		SELECT id, name, fingerprint, ip_address, mac_address, device_type, is_trusted, first_seen, last_seen, notes
		FROM devices ORDER BY last_seen DESC`)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()

	var devices []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, fingerprint, deviceType, firstSeen, lastSeen string
		var ip, mac, notes sql.NullString
		var isTrusted int
		if err := rows.Scan(&id, &name, &fingerprint, &ip, &mac, &deviceType, &isTrusted, &firstSeen, &lastSeen, &notes); err != nil {
			continue
		}
		devices = append(devices, map[string]interface{}{
			"id":          id,
			"name":        name,
			"fingerprint": fingerprint,
			"ip_address":  ip.String,
			"mac_address": mac.String,
			"device_type": deviceType,
			"is_trusted":  isTrusted == 1,
			"first_seen":  firstSeen,
			"last_seen":   lastSeen,
			"notes":       notes.String,
		})
	}
	if devices == nil {
		devices = []map[string]interface{}{}
	}
	ok(w, devices)
}

func (h *Handler) TrustDevice(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE devices SET is_trusted = 1, last_seen = ? WHERE id = ?`, now, id)
	h.writeAuditLog("security.device_trusted", fmt.Sprintf("device id=%d trusted by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "device trusted"})
}

func (h *Handler) RevokeDevice(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE devices SET is_trusted = 0, last_seen = ? WHERE id = ?`, now, id)
	h.writeAuditLog("security.device_revoked", fmt.Sprintf("device id=%d revoked by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "device access revoked"})
}

func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	_, _ = h.db.Conn().Exec(`DELETE FROM devices WHERE id = ?`, id)
	h.writeAuditLog("security.device_deleted", fmt.Sprintf("device id=%d deleted by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "device removed"})
}

// ─────────────────────────────────────────────
// Sessions
// ─────────────────────────────────────────────

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Conn().Query(`
		SELECT s.id, s.user_id, u.username, s.ip_address, s.user_agent, s.created_at, s.expires_at, s.last_seen
		FROM sessions s JOIN users u ON u.id = s.user_id
		ORDER BY s.last_seen DESC`)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var id, userAgent, createdAt, expiresAt, lastSeen, sessionID string
		var userID int64
		var username, ip string
		if err := rows.Scan(&sessionID, &userID, &username, &ip, &userAgent, &createdAt, &expiresAt, &lastSeen); err != nil {
			continue
		}
		_ = id
		sessions = append(sessions, map[string]interface{}{
			"id":         sessionID,
			"user_id":    userID,
			"username":   username,
			"ip_address": ip,
			"user_agent": userAgent,
			"created_at": createdAt,
			"expires_at": expiresAt,
			"last_seen":  lastSeen,
		})
	}
	if sessions == nil {
		sessions = []map[string]interface{}{}
	}
	ok(w, sessions)
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id := paramStr(r, "id")
	_, _ = h.db.Conn().Exec(`DELETE FROM sessions WHERE id = ?`, id)
	h.writeAuditLog("security.session_revoked", fmt.Sprintf("session %s revoked by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "session revoked"})
}

// ─────────────────────────────────────────────
// Settings
// ─────────────────────────────────────────────

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Conn().Query(`SELECT key, value, updated_at FROM settings ORDER BY key`)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()

	settings := map[string]interface{}{}
	for rows.Next() {
		var key, value, updatedAt string
		if err := rows.Scan(&key, &value, &updatedAt); err != nil {
			continue
		}
		settings[key] = map[string]string{"value": value, "updated_at": updatedAt}
	}
	ok(w, settings)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var body map[string]string
	if err := decode(r, &body); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for k, v := range body {
		_, _ = h.db.Conn().Exec(`
			INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, k, v, now)
	}
	h.writeAuditLog("security.settings_updated", fmt.Sprintf("settings updated by %s", claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "settings updated"})
}
