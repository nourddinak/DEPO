package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"depo/internal/crypto"
	"depo/internal/db"
	"github.com/google/uuid"
)

type contextKey string

const (
	contextKeyUser      contextKey = "depo_user"
	contextKeySession   contextKey = "depo_session"
	contextKeyRequestID contextKey = "depo_request_id"
)

// Standard JSON API response envelope
type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// RateLimiter tracks recent attempts by IP
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	window   time.Duration
	maxLimit int
}

func NewRateLimiter(maxLimit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		maxLimit: maxLimit,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.maxLimit {
		rl.attempts[ip] = valid
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

// RequestIDMiddleware injects a unique X-Request-ID
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), contextKeyRequestID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SecurityHeadersMiddleware adds modern security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware resolves session from cookie or Authorization: Bearer <token>
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Check Authorization header first
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		} else if cookie, err := r.Cookie("depo_session"); err == nil {
			token = cookie.Value
		}

		if token != "" {
			tokenHash := crypto.SHA256String(token)
			session, user, err := s.db.GetSessionByTokenHash(tokenHash)
			if err == nil && session != nil && user != nil {
				ctx := context.WithValue(r.Context(), contextKeyUser, user)
				ctx = context.WithValue(ctx, contextKeySession, session)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(contextKeyUser).(*db.User)
		if !ok || user == nil {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}
		next(w, r)
	}
}

func RequireRole(role db.UserRole, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(contextKeyUser).(*db.User)
		if !ok || user == nil {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}
		if user.Role != db.RoleOwner && user.Role != role {
			writeError(w, http.StatusForbidden, fmt.Sprintf("Action requires %s privileges", role))
			return
		}
		next(w, r)
	}
}

func GetUserFromContext(ctx context.Context) (*db.User, error) {
	user, ok := ctx.Value(contextKeyUser).(*db.User)
	if !ok || user == nil {
		return nil, errors.New("no user in context")
	}
	return user, nil
}
