package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	fabcrypto "github.com/fabrixe/fabrixe/internal/crypto"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────
// Login
// ─────────────────────────────────────────────

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		badRequest(w, "username and password are required")
		return
	}

	ip := realIP(r)

	// Fetch user
	var (
		userID       int64
		passwordHash string
		role         string
		fullName     string
		isActive     bool
		failedLogins int
		lockedUntil  sql.NullString
	)
	err := h.db.Conn().QueryRow(`
		SELECT id, password_hash, role, full_name, is_active, failed_logins, locked_until
		FROM users WHERE username = ?`, req.Username).
		Scan(&userID, &passwordHash, &role, &fullName, &isActive, &failedLogins, &lockedUntil)

	if err == sql.ErrNoRows {
		h.writeAuditLog("auth.login_failed", fmt.Sprintf("login attempt for unknown user %q", req.Username), nil, "", ip, "failure")
		fail(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	if !isActive {
		fail(w, http.StatusForbidden, "account is disabled")
		return
	}

	// Check lockout
	if lockedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntil.String)
		if time.Now().Before(t) {
			fail(w, http.StatusForbidden, fmt.Sprintf("account locked until %s", t.Format(time.RFC3339)))
			return
		}
	}

	// Verify password
	if err := fabcrypto.CompareHashAndPassword(passwordHash, req.Password); err != nil {
		newFailed := failedLogins + 1
		var lockSQL string
		var lockArgs []interface{}

		if newFailed >= h.cfg.Security.MaxLoginFails {
			lockUntil := time.Now().Add(time.Duration(h.cfg.Security.LockoutMinutes) * time.Minute).Format(time.RFC3339)
			lockSQL = `UPDATE users SET failed_logins = ?, locked_until = ? WHERE id = ?`
			lockArgs = []interface{}{newFailed, lockUntil, userID}
		} else {
			lockSQL = `UPDATE users SET failed_logins = ? WHERE id = ?`
			lockArgs = []interface{}{newFailed, userID}
		}
		_, _ = h.db.Conn().Exec(lockSQL, lockArgs...)

		h.writeAuditLog("auth.login_failed", fmt.Sprintf("wrong password for user %q (attempt %d)", req.Username, newFailed), &userID, req.Username, ip, "failure")
		fail(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Issue tokens
	pair, err := h.auth.IssueTokenPair(userID, req.Username, role)
	if err != nil {
		internalError(w, err)
		return
	}

	// Persist session
	sessionID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	expiresAt := time.Now().Add(time.Duration(h.cfg.JWT.RefreshTokenTTL) * 24 * time.Hour).UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`
		INSERT INTO sessions (id, user_id, refresh_token, ip_address, user_agent, created_at, expires_at, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, userID, pair.RefreshToken, ip, r.UserAgent(), now, expiresAt, now)

	// Reset failed logins + update last_login
	_, _ = h.db.Conn().Exec(`UPDATE users SET failed_logins = 0, locked_until = NULL, last_login = ? WHERE id = ?`, now, userID)

	// Track device
	h.trackDevice(r, ip)

	h.writeAuditLog("auth.login", fmt.Sprintf("user %q logged in", req.Username), &userID, req.Username, ip, "success")

	ok(w, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_at":    pair.ExpiresAt,
		"token_type":    pair.TokenType,
		"user": map[string]interface{}{
			"id":        userID,
			"username":  req.Username,
			"full_name": fullName,
			"role":      role,
		},
	})
}

// ─────────────────────────────────────────────
// Logout
// ─────────────────────────────────────────────

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	_ = decode(r, &req)

	claims := claimsFromCtx(r.Context())
	if req.RefreshToken != "" {
		_, _ = h.db.Conn().Exec(`DELETE FROM sessions WHERE refresh_token = ?`, req.RefreshToken)
	}
	if claims != nil {
		h.writeAuditLog("auth.logout", fmt.Sprintf("user %q logged out", claims.Username), &claims.UserID, claims.Username, realIP(r), "success")
	}
	ok(w, map[string]string{"message": "logged out"})
}

// ─────────────────────────────────────────────
// Refresh token
// ─────────────────────────────────────────────

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decode(r, &req); err != nil || req.RefreshToken == "" {
		badRequest(w, "refresh_token is required")
		return
	}

	var userID int64
	var expiresAt string
	err := h.db.Conn().QueryRow(`
		SELECT user_id, expires_at FROM sessions WHERE refresh_token = ?`, req.RefreshToken).
		Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		fail(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	t, _ := time.Parse(time.RFC3339, expiresAt)
	if time.Now().After(t) {
		_, _ = h.db.Conn().Exec(`DELETE FROM sessions WHERE refresh_token = ?`, req.RefreshToken)
		fail(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	var username, role string
	_ = h.db.Conn().QueryRow(`SELECT username, role FROM users WHERE id = ?`, userID).Scan(&username, &role)

	pair, err := h.auth.IssueTokenPair(userID, username, role)
	if err != nil {
		internalError(w, err)
		return
	}

	// Rotate refresh token
	now := time.Now().UTC().Format(time.RFC3339)
	newExpires := time.Now().Add(time.Duration(h.cfg.JWT.RefreshTokenTTL) * 24 * time.Hour).UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`
		UPDATE sessions SET refresh_token = ?, expires_at = ?, last_seen = ? WHERE refresh_token = ?`,
		pair.RefreshToken, newExpires, now, req.RefreshToken)

	ok(w, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_at":    pair.ExpiresAt,
		"token_type":    pair.TokenType,
	})
}

// ─────────────────────────────────────────────
// Me
// ─────────────────────────────────────────────

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	if claims == nil {
		fail(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var email, fullName string
	var lastLogin sql.NullString
	_ = h.db.Conn().QueryRow(`SELECT email, full_name, last_login FROM users WHERE id = ?`, claims.UserID).
		Scan(&email, &fullName, &lastLogin)

	ok(w, map[string]interface{}{
		"id":         claims.UserID,
		"username":   claims.Username,
		"email":      email,
		"full_name":  fullName,
		"role":       claims.Role,
		"last_login": lastLogin.String,
	})
}

// ─────────────────────────────────────────────
// Change password
// ─────────────────────────────────────────────

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req changePasswordRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	if len(req.NewPassword) < 10 {
		badRequest(w, "new password must be at least 10 characters")
		return
	}

	var hash string
	_ = h.db.Conn().QueryRow(`SELECT password_hash FROM users WHERE id = ?`, claims.UserID).Scan(&hash)
	if err := fabcrypto.CompareHashAndPassword(hash, req.CurrentPassword); err != nil {
		fail(w, http.StatusForbidden, "current password is incorrect")
		return
	}

	newHash, err := fabcrypto.HashPassword(req.NewPassword)
	if err != nil {
		internalError(w, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		newHash, now, claims.UserID)

	// Invalidate all sessions
	_, _ = h.db.Conn().Exec(`DELETE FROM sessions WHERE user_id = ?`, claims.UserID)

	h.writeAuditLog("security.password_changed", fmt.Sprintf("user %q changed password", claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")

	ok(w, map[string]string{"message": "password changed successfully"})
}

// ─────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────

func (h *Handler) writeAuditLog(eventType, description string, userID *int64, username, ip, outcome string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`
		INSERT INTO audit_logs (event_type, description, user_id, username, ip_address, outcome, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		eventType, description, userID, username, ip, outcome, now)
}

func (h *Handler) trackDevice(r *http.Request, ip string) {
	fp := fingerprintDevice(r)
	now := time.Now().UTC().Format(time.RFC3339)
	name := "Browser @ " + ip

	_, _ = h.db.Conn().Exec(`
		INSERT INTO devices (name, fingerprint, ip_address, device_type, first_seen, last_seen)
		VALUES (?, ?, ?, 'workstation', ?, ?)
		ON CONFLICT(fingerprint) DO UPDATE SET ip_address = excluded.ip_address, last_seen = excluded.last_seen`,
		name, fp, ip, now, now)
}

func fingerprintDevice(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	ip := realIP(r)
	return fmt.Sprintf("%x", []byte(ua+":"+ip))
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[:i]
	}
	return addr
}
