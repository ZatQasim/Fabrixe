package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"github.com/fabrixe/fabrixe/internal/api/handlers"
	"github.com/fabrixe/fabrixe/internal/auth"
	"github.com/fabrixe/fabrixe/internal/config"
	"github.com/fabrixe/fabrixe/internal/db"
	"github.com/fabrixe/fabrixe/pkg/models"
)

type contextKey string

const (
	ctxClaims contextKey = "claims"
)

// NewRouter assembles the Fabrixe HTTP router.
func NewRouter(cfg *config.Config, database *db.DB) http.Handler {
	authSvc := auth.New(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL)
	h := handlers.New(cfg, database, authSvc)

	r := mux.NewRouter()

	// ── Health ──────────────────────────────────────────────────────────
	r.HandleFunc("/api/health", healthHandler).Methods("GET")

	// ── Auth ────────────────────────────────────────────────────────────
	a := r.PathPrefix("/api/auth").Subrouter()
	a.HandleFunc("/login", h.Login).Methods("POST")
	a.HandleFunc("/logout", h.Logout).Methods("POST")
	a.HandleFunc("/refresh", h.RefreshToken).Methods("POST")
	a.HandleFunc("/me", requireAuth(authSvc, h.Me)).Methods("GET")
	a.HandleFunc("/change-password", requireAuth(authSvc, h.ChangePassword)).Methods("POST")

	// ── System Management ──────────────────────────────────────────────
	sm := r.PathPrefix("/api/system").Subrouter()
	sm.HandleFunc("/snapshot", requireAuth(authSvc, h.SystemSnapshot)).Methods("GET")
	sm.HandleFunc("/hardware", requireAuth(authSvc, h.SystemHardware)).Methods("GET")
	sm.HandleFunc("/services", requireAuth(authSvc, h.ListServices)).Methods("GET")
	sm.HandleFunc("/services/{name}/{action}", requireRole(authSvc, auth.RoleOperator, h.ServiceAction)).Methods("POST")
	sm.HandleFunc("/alerts", requireAuth(authSvc, h.ListAlerts)).Methods("GET")
	sm.HandleFunc("/alerts/{id}/resolve", requireRole(authSvc, auth.RoleOperator, h.ResolveAlert)).Methods("POST")
	sm.HandleFunc("/ws", h.SystemWebSocket)

	// ── Deployment & Automation ──────────────────────────────────────
	dep := r.PathPrefix("/api/deployment").Subrouter()
	dep.HandleFunc("", requireAuth(authSvc, h.ListDeployments)).Methods("GET")
	dep.HandleFunc("", requireRole(authSvc, auth.RoleOperator, h.CreateDeployment)).Methods("POST")
	dep.HandleFunc("/tasks", requireAuth(authSvc, h.ListTasks)).Methods("GET")
	dep.HandleFunc("/tasks", requireRole(authSvc, auth.RoleOperator, h.CreateTask)).Methods("POST")
	dep.HandleFunc("/tasks/{id}", requireAuth(authSvc, h.GetTask)).Methods("GET")
	dep.HandleFunc("/tasks/{id}", requireRole(authSvc, auth.RoleOperator, h.UpdateTask)).Methods("PUT")
	dep.HandleFunc("/tasks/{id}", requireRole(authSvc, auth.RoleAdministrator, h.DeleteTask)).Methods("DELETE")
	dep.HandleFunc("/tasks/{id}/run", requireRole(authSvc, auth.RoleOperator, h.RunTask)).Methods("POST")
	dep.HandleFunc("/{id}", requireAuth(authSvc, h.GetDeployment)).Methods("GET")
	dep.HandleFunc("/{id}", requireRole(authSvc, auth.RoleOperator, h.UpdateDeployment)).Methods("PUT")
	dep.HandleFunc("/{id}", requireRole(authSvc, auth.RoleAdministrator, h.DeleteDeployment)).Methods("DELETE")
	dep.HandleFunc("/{id}/run", requireRole(authSvc, auth.RoleOperator, h.RunDeployment)).Methods("POST")

	// ── Internal Security ────────────────────────────────────────────
	sec := r.PathPrefix("/api/security").Subrouter()
	sec.HandleFunc("/users", requireRole(authSvc, auth.RoleAdministrator, h.ListUsers)).Methods("GET")
	sec.HandleFunc("/users", requireRole(authSvc, auth.RoleAdministrator, h.CreateUser)).Methods("POST")
	sec.HandleFunc("/users/{id}", requireRole(authSvc, auth.RoleAdministrator, h.GetUser)).Methods("GET")
	sec.HandleFunc("/users/{id}", requireRole(authSvc, auth.RoleAdministrator, h.UpdateUser)).Methods("PUT")
	sec.HandleFunc("/users/{id}", requireRole(authSvc, auth.RoleAdministrator, h.DeleteUser)).Methods("DELETE")
	sec.HandleFunc("/users/{id}/unlock", requireRole(authSvc, auth.RoleAdministrator, h.UnlockUser)).Methods("POST")
	sec.HandleFunc("/audit-logs", requireRole(authSvc, auth.RoleOperator, h.ListAuditLogs)).Methods("GET")
	sec.HandleFunc("/devices", requireAuth(authSvc, h.ListDevices)).Methods("GET")
	sec.HandleFunc("/devices/{id}/trust", requireRole(authSvc, auth.RoleAdministrator, h.TrustDevice)).Methods("POST")
	sec.HandleFunc("/devices/{id}/revoke", requireRole(authSvc, auth.RoleAdministrator, h.RevokeDevice)).Methods("POST")
	sec.HandleFunc("/devices/{id}", requireRole(authSvc, auth.RoleAdministrator, h.DeleteDevice)).Methods("DELETE")
	sec.HandleFunc("/sessions", requireRole(authSvc, auth.RoleAdministrator, h.ListSessions)).Methods("GET")
	sec.HandleFunc("/sessions/{id}", requireRole(authSvc, auth.RoleAdministrator, h.RevokeSession)).Methods("DELETE")
	sec.HandleFunc("/settings", requireRole(authSvc, auth.RoleAdministrator, h.GetSettings)).Methods("GET")
	sec.HandleFunc("/settings", requireRole(authSvc, auth.RoleAdministrator, h.UpdateSettings)).Methods("PUT")

	// ── Protected Communication ──────────────────────────────────────
	comm := r.PathPrefix("/api/comm").Subrouter()
	comm.HandleFunc("/nodes", requireAuth(authSvc, h.ListNodes)).Methods("GET")
	comm.HandleFunc("/nodes", requireRole(authSvc, auth.RoleOperator, h.AddNode)).Methods("POST")
	comm.HandleFunc("/identity", requireAuth(authSvc, h.GetNodeIdentity)).Methods("GET")
	comm.HandleFunc("/nodes/{id}", requireAuth(authSvc, h.GetNode)).Methods("GET")
	comm.HandleFunc("/nodes/{id}/trust", requireRole(authSvc, auth.RoleAdministrator, h.TrustNode)).Methods("POST")
	comm.HandleFunc("/nodes/{id}/revoke", requireRole(authSvc, auth.RoleAdministrator, h.RevokeNode)).Methods("POST")
	comm.HandleFunc("/nodes/{id}", requireRole(authSvc, auth.RoleAdministrator, h.DeleteNode)).Methods("DELETE")
	comm.HandleFunc("/ping/{id}", requireRole(authSvc, auth.RoleOperator, h.PingNode)).Methods("POST")

	// ── Frontend static files ────────────────────────────────────────
	staticDir := cfg.Server.StaticDir
	if staticDir != "" {
		r.PathPrefix("/").Handler(spaHandler(staticDir))
	}

	return r
}

// ─────────────────────────────────────────────
// Auth middleware
// ─────────────────────────────────────────────

func requireAuth(svc *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := extractClaims(svc, r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, models.Fail("authentication required"))
			return
		}
		ctx := context.WithValue(r.Context(), ctxClaims, claims)
		next(w, r.WithContext(ctx))
	}
}

func requireRole(svc *auth.Service, role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := extractClaims(svc, r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, models.Fail("authentication required"))
			return
		}
		if !auth.HasPermission(claims.Role, role) {
			writeJSON(w, http.StatusForbidden, models.Fail("insufficient permissions"))
			return
		}
		ctx := context.WithValue(r.Context(), ctxClaims, claims)
		next(w, r.WithContext(ctx))
	}
}

func extractClaims(svc *auth.Service, r *http.Request) (*auth.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return svc.ValidateToken(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if c, err := r.Cookie("fabrixe_token"); err == nil {
		return svc.ValidateToken(c.Value)
	}
	return nil, auth.ErrTokenInvalid
}

// ─────────────────────────────────────────────
// SPA handler — serves React app, falls back to index.html
// ─────────────────────────────────────────────

func spaHandler(staticDir string) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, filepath.Clean("/"+r.URL.Path))
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "1.0.0"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
