package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"depo/internal/crypto"
	"depo/internal/db"
	"github.com/pquerna/otp/totp"
)

type SetupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type TOTPSetupResponse struct {
	Secret string `json:"secret"`
	URL    string `json:"url"`
}

type TOTPVerifyRequest struct {
	Code string `json:"code"`
}

type InviteRequest struct {
	Email string      `json:"email"`
	Role  db.UserRole `json:"role"`
}

type AcceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.db.CountUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to count users")
		return
	}

	setupRequired := count == 0
	user, _ := GetUserFromContext(r.Context())

	writeJSON(w, http.StatusOK, map[string]any{
		"setup_required": setupRequired,
		"authenticated":  user != nil,
		"user":           user,
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	count, err := s.db.CountUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if count > 0 {
		writeError(w, http.StatusBadRequest, "Platform setup has already been completed")
		return
	}

	var req SetupRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "Valid email address is required")
		return
	}
	if len(req.Password) < 10 {
		writeError(w, http.StatusBadRequest, "Password must be at least 10 characters")
		return
	}

	passHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user, err := s.db.CreateUser(email, passHash, db.RoleOwner)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create owner account")
		return
	}

	// Create initial session
	rawToken, tokenHash, _ := crypto.GenerateSessionToken()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	_, _ = s.db.CreateSession(user.ID, tokenHash, getClientIP(r), r.UserAgent(), expiresAt)

	http.SetCookie(w, &http.Cookie{
		Name:     "depo_session",
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set to true when deployed over HTTPS
	})

	_ = s.db.RecordAuditLog(user.ID, user.Email, "setup_platform", "user", user.ID, `{"role":"owner"}`, getClientIP(r))

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":  user,
		"token": rawToken,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	if !s.rateLimiter.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "Too many login attempts. Please wait a moment.")
		return
	}

	var req LoginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	ok, err := crypto.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Check 2FA
	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"totp_required": true,
			})
			return
		}

		valid := totp.Validate(req.TOTPCode, user.TOTPSecret)
		if !valid {
			// Check recovery codes
			var recoveryCodes []string
			_ = json.Unmarshal([]byte(user.RecoveryCodes), &recoveryCodes)
			inputCodeHash := crypto.SHA256String(strings.TrimSpace(req.TOTPCode))
			matchedIdx := -1
			for idx, codeHash := range recoveryCodes {
				if codeHash == inputCodeHash {
					matchedIdx = idx
					break
				}
			}
			if matchedIdx == -1 {
				writeError(w, http.StatusUnauthorized, "Invalid 2FA code or recovery code")
				return
			}
			// Burn used recovery code
			recoveryCodes = append(recoveryCodes[:matchedIdx], recoveryCodes[matchedIdx+1:]...)
			burnedBytes, _ := json.Marshal(recoveryCodes)
			_ = s.db.UpdateUserTOTP(user.ID, user.TOTPSecret, string(burnedBytes), true)
		}
	}

	// Create session
	rawToken, tokenHash, _ := crypto.GenerateSessionToken()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	_, _ = s.db.CreateSession(user.ID, tokenHash, ip, r.UserAgent(), expiresAt)

	http.SetCookie(w, &http.Cookie{
		Name:     "depo_session",
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	_ = s.db.RecordAuditLog(user.ID, user.Email, "login", "user", user.ID, `{}`, ip)

	writeJSON(w, http.StatusOK, map[string]any{
		"user":  user,
		"token": rawToken,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if session, ok := r.Context().Value(contextKeySession).(*db.Session); ok && session != nil {
		_ = s.db.DeleteSession(session.ID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "depo_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := GetUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	user, err := GetUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "DEPO",
		AccountName: user.Email,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate TOTP secret")
		return
	}

	// Temporarily save secret until verified
	_ = s.db.UpdateUserTOTP(user.ID, key.Secret(), user.RecoveryCodes, false)

	writeJSON(w, http.StatusOK, TOTPSetupResponse{
		Secret: key.Secret(),
		URL:    key.URL(),
	})
}

func (s *Server) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	user, err := GetUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var req TOTPVerifyRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	dbUser, err := s.db.GetUserByID(user.ID)
	if err != nil || dbUser.TOTPSecret == "" {
		writeError(w, http.StatusBadRequest, "No pending TOTP setup found")
		return
	}

	if !totp.Validate(req.Code, dbUser.TOTPSecret) {
		writeError(w, http.StatusBadRequest, "Invalid authentication code")
		return
	}

	// Generate 8 random recovery codes
	var rawCodes []string
	var hashedCodes []string
	for i := 0; i < 8; i++ {
		b := make([]byte, 5)
		_, _ = rand.Read(b)
		code := fmt.Sprintf("%x-%x", b[:2], b[2:])
		rawCodes = append(rawCodes, code)
		hashedCodes = append(hashedCodes, crypto.SHA256String(code))
	}
	recoveryJSON, _ := json.Marshal(hashedCodes)

	if err := s.db.UpdateUserTOTP(user.ID, dbUser.TOTPSecret, string(recoveryJSON), true); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to activate 2FA")
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "enable_totp", "user", user.ID, `{}`, getClientIP(r))

	writeJSON(w, http.StatusOK, map[string]any{
		"message":        "2FA enabled successfully",
		"recovery_codes": rawCodes,
	})
}

func (s *Server) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	var req InviteRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "Valid email required")
		return
	}

	if req.Role != db.RoleAdmin && req.Role != db.RoleMember {
		req.Role = db.RoleMember
	}

	rawToken, tokenHash, err := crypto.GenerateInvitationToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	inv, err := s.db.CreateInvitation(email, req.Role, tokenHash, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create invitation")
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "create_invitation", "invitation", inv.ID, fmt.Sprintf(`{"email":"%s","role":"%s"}`, email, req.Role), getClientIP(r))

	writeJSON(w, http.StatusCreated, map[string]any{
		"invitation":       inv,
		"invitation_token": rawToken,
		"invitation_url":   fmt.Sprintf("/accept-invite?token=%s", rawToken),
	})
}

func (s *Server) handleListInvitations(w http.ResponseWriter, r *http.Request) {
	invs, err := s.db.ListInvitations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list invitations")
		return
	}
	writeJSON(w, http.StatusOK, invs)
}

func (s *Server) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	var req AcceptInviteRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	tokenHash := crypto.SHA256String(req.Token)
	inv, err := s.db.GetInvitationByTokenHash(tokenHash)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid or expired invitation token")
		return
	}

	if len(req.Password) < 10 {
		writeError(w, http.StatusBadRequest, "Password must be at least 10 characters")
		return
	}

	passHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user, err := s.db.CreateUser(inv.Email, passHash, inv.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user account")
		return
	}

	_ = s.db.MarkInvitationAccepted(inv.ID)

	rawToken, sessionHash, _ := crypto.GenerateSessionToken()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	_, _ = s.db.CreateSession(user.ID, sessionHash, getClientIP(r), r.UserAgent(), expiresAt)

	http.SetCookie(w, &http.Cookie{
		Name:     "depo_session",
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	_ = s.db.RecordAuditLog(user.ID, user.Email, "accept_invitation", "user", user.ID, `{}`, getClientIP(r))

	writeJSON(w, http.StatusOK, map[string]any{
		"user":  user,
		"token": rawToken,
	})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.db.ListAuditLogs(100, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list audit logs")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// GenerateRandomHex Helper
func generateHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
