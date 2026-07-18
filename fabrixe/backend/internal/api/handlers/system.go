package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	sysmod "github.com/fabrixe/fabrixe/internal/modules/system"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Origin check handled by our CORS middleware + auth
		return true
	},
}

// GET /api/system/snapshot
func (h *Handler) SystemSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := sysmod.GetSnapshot()
	if err != nil {
		internalError(w, err)
		return
	}
	ok(w, snap)
}

// GET /api/system/hardware
func (h *Handler) SystemHardware(w http.ResponseWriter, r *http.Request) {
	hw := sysmod.GetHardwareInfo()
	ok(w, hw)
}

// GET /api/system/services
func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := sysmod.GetServices()
	if err != nil {
		internalError(w, err)
		return
	}
	ok(w, services)
}

// POST /api/system/services/{name}/{action}
func (h *Handler) ServiceAction(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	name := paramStr(r, "name")
	action := paramStr(r, "action")

	out, err := sysmod.ServiceAction(name, action)
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}

	h.writeAuditLog(
		"system.service_"+action,
		fmt.Sprintf("service %q action %q by %s: %s", name, action, claims.Username, out),
		&claims.UserID, claims.Username, realIP(r), outcome,
	)

	if err != nil {
		fail(w, http.StatusInternalServerError, fmt.Sprintf("action failed: %v — output: %s", err, out))
		return
	}
	ok(w, map[string]string{"output": out})
}

// GET /api/system/alerts
func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	resolved := queryStr(r, "resolved")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	query := `SELECT id, level, source, message, is_resolved, resolved_by, resolved_at, created_at
		FROM alerts`
	args := []interface{}{}

	if resolved == "true" {
		query += ` WHERE is_resolved = 1`
	} else if resolved == "false" || resolved == "" {
		query += ` WHERE is_resolved = 0`
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.db.Conn().Query(query, args...)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()

	var alerts []map[string]interface{}
	for rows.Next() {
		var id int64
		var level, source, message string
		var isResolved int
		var resolvedBy sql.NullInt64
		var resolvedAt, createdAt sql.NullString
		if err := rows.Scan(&id, &level, &source, &message, &isResolved, &resolvedBy, &resolvedAt, &createdAt); err != nil {
			continue
		}
		alerts = append(alerts, map[string]interface{}{
			"id":          id,
			"level":       level,
			"source":      source,
			"message":     message,
			"is_resolved": isResolved == 1,
			"resolved_by": resolvedBy.Int64,
			"resolved_at": resolvedAt.String,
			"created_at":  createdAt.String,
		})
	}
	if alerts == nil {
		alerts = []map[string]interface{}{}
	}
	ok(w, alerts)
}

// POST /api/system/alerts/{id}/resolve
func (h *Handler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid alert id")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := h.db.Conn().Exec(`
		UPDATE alerts SET is_resolved = 1, resolved_by = ?, resolved_at = ? WHERE id = ? AND is_resolved = 0`,
		claims.UserID, now, id)
	if err != nil {
		internalError(w, err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		notFound(w)
		return
	}

	h.writeAuditLog("system.alert_resolved", fmt.Sprintf("alert %d resolved by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "alert resolved"})
}

// ─────────────────────────────────────────────
// WebSocket — live system metrics push
// ─────────────────────────────────────────────

func (h *Handler) SystemWebSocket(w http.ResponseWriter, r *http.Request) {
	// Auth via cookie or query param for WebSocket
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		if c, err := r.Cookie("fabrixe_token"); err == nil {
			tokenStr = c.Value
		}
	}
	if tokenStr == "" {
		if bearer := r.Header.Get("Authorization"); len(bearer) > 7 {
			tokenStr = bearer[7:]
		}
	}

	if _, err := h.auth.ValidateToken(tokenStr); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Send snapshots every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Handle client close
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			snap, err := sysmod.GetSnapshot()
			if err != nil {
				continue
			}
			if err := conn.WriteJSON(map[string]interface{}{
				"type":    "snapshot",
				"payload": snap,
			}); err != nil {
				return
			}
		}
	}
}

// CreateAlert creates a new alert (internal use).
func (h *Handler) createAlert(level, source, message string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`INSERT INTO alerts (level, source, message, created_at) VALUES (?, ?, ?, ?)`,
		level, source, message, now)
}

// helper not exported, only for query string
func (h *Handler) parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
