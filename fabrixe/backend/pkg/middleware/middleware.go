package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	CtxRequestID contextKey = "request_id"
	CtxUserID    contextKey = "user_id"
	CtxUsername  contextKey = "username"
	CtxUserRole  contextKey = "user_role"
)

// ─────────────────────────────────────────────
// Chain — applies middleware in order (outermost first)
// ─────────────────────────────────────────────

func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// ─────────────────────────────────────────────
// RequestID
// ─────────────────────────────────────────────

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), CtxRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ─────────────────────────────────────────────
// Logger
// ─────────────────────────────────────────────

type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		fmt.Printf("[HTTP] %s %s %d %dB %s %s\n",
			r.Method, r.URL.Path, rec.status, rec.size,
			time.Since(start).Round(time.Millisecond),
			r.RemoteAddr,
		)
	})
}

// ─────────────────────────────────────────────
// SecurityHeaders
// ─────────────────────────────────────────────

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self' wss: ws:;")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────
// CORS
// ─────────────────────────────────────────────

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if originSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────
// RateLimit — simple in-memory token bucket per IP
// ─────────────────────────────────────────────

type ipBucket struct {
	tokens    float64
	lastRefil time.Time
}

type rateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*ipBucket
	ratePerSec float64
	burst     float64
}

var globalRL *rateLimiter
var rlOnce sync.Once

func RateLimit(perMinute int) func(http.Handler) http.Handler {
	rlOnce.Do(func() {
		globalRL = &rateLimiter{
			buckets:   make(map[string]*ipBucket),
			ratePerSec: float64(perMinute) / 60.0,
			burst:     float64(perMinute),
		}
		// Purge stale buckets every minute
		go func() {
			for range time.Tick(time.Minute) {
				globalRL.mu.Lock()
				for ip, b := range globalRL.buckets {
					if time.Since(b.lastRefil) > 5*time.Minute {
						delete(globalRL.buckets, ip)
					}
				}
				globalRL.mu.Unlock()
			}
		}()
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)

			globalRL.mu.Lock()
			b, ok := globalRL.buckets[ip]
			if !ok {
				b = &ipBucket{tokens: globalRL.burst, lastRefil: time.Now()}
				globalRL.buckets[ip] = b
			}
			elapsed := time.Since(b.lastRefil).Seconds()
			b.tokens = min(globalRL.burst, b.tokens+elapsed*globalRL.ratePerSec)
			b.lastRefil = time.Now()

			if b.tokens < 1 {
				globalRL.mu.Unlock()
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			b.tokens--
			globalRL.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	idx := strings.LastIndex(r.RemoteAddr, ":")
	if idx < 0 {
		return r.RemoteAddr
	}
	return r.RemoteAddr[:idx]
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
