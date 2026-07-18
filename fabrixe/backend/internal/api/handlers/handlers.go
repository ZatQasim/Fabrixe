package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/fabrixe/fabrixe/internal/auth"
	"github.com/fabrixe/fabrixe/internal/config"
	"github.com/fabrixe/fabrixe/internal/db"
	"github.com/fabrixe/fabrixe/pkg/models"
)

type contextKey string

const ctxClaims contextKey = "claims"

// Handler holds shared dependencies for all API handlers.
type Handler struct {
	cfg  *config.Config
	db   *db.DB
	auth *auth.Service
}

// New creates a Handler with all dependencies wired.
func New(cfg *config.Config, database *db.DB, authSvc *auth.Service) *Handler {
	return &Handler{cfg: cfg, db: database, auth: authSvc}
}

// ─────────────────────────────────────────────
// Context helpers
// ─────────────────────────────────────────────

func claimsFromCtx(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(ctxClaims).(*auth.Claims)
	return c
}

// ─────────────────────────────────────────────
// Response helpers
// ─────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ok(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, models.OK(data))
}

func created(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusCreated, models.OK(data))
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, models.Fail(msg))
}

func badRequest(w http.ResponseWriter, msg string) {
	fail(w, http.StatusBadRequest, msg)
}

func notFound(w http.ResponseWriter) {
	fail(w, http.StatusNotFound, "not found")
}

func internalError(w http.ResponseWriter, err error) {
	fail(w, http.StatusInternalServerError, "internal server error: "+err.Error())
}

// ─────────────────────────────────────────────
// Request decoding
// ─────────────────────────────────────────────

func decode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// ─────────────────────────────────────────────
// URL parameter helpers
// ─────────────────────────────────────────────

func paramStr(r *http.Request, key string) string {
	return mux.Vars(r)[key]
}

func paramInt64(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)[key], 10, 64)
}

func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func queryStr(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}
