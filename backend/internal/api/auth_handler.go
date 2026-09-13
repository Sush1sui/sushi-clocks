package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/config"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
)

type AuthHandler struct {
	cfg      *config.Config
	userRepo *repository.UserRepository
	jwtMgr   *auth.JWTManager
}

func NewAuthHandler(cfg *config.Config, userRepo *repository.UserRepository, jwtMgr *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		cfg:      cfg,
		userRepo: userRepo,
		jwtMgr:   jwtMgr,
	}
}

// setAuthCookies sets secure http-only refresh and access cookies
func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, accessToken string, accessTTL int64, refreshToken string, refreshTTL int64) {
	isSecure := h.cfg.Environment == "production"

	// Refresh token cookie — restricted to /api/v1/auth
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth",
		MaxAge:   int(refreshTTL),
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Access token cookie — available across entire API
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   int(accessTTL),
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearAuthCookies clears auth cookies on logout
func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	isSecure := h.cfg.Environment == "production"

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Enforce 1MB request body limit to prevent memory exhaustion DoS
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		RespondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Validate email format
	if _, err := mail.ParseAddress(req.Email); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid email address format")
		return
	}

	user, err := h.userRepo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		log.Printf("login get user error: %v", err)
		RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		RespondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	accessToken, accessExpiresIn, err := h.jwtMgr.GenerateAccessToken(user)
	if err != nil {
		log.Printf("generate access token error: %v", err)
		RespondError(w, http.StatusInternalServerError, "could not generate access token")
		return
	}

	refreshToken, refreshExpiresIn, err := h.jwtMgr.GenerateRefreshToken(user)
	if err != nil {
		log.Printf("generate refresh token error: %v", err)
		RespondError(w, http.StatusInternalServerError, "could not generate refresh token")
		return
	}

	h.setAuthCookies(w, accessToken, accessExpiresIn, refreshToken, refreshExpiresIn)

	resp := domain.LoginResponse{
		User:        user.ToResponse(),
		AccessToken: accessToken,
		ExpiresIn:   accessExpiresIn,
	}

	RespondOK(w, http.StatusOK, resp)
}

// Refresh handles POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var refreshTokenStr string

	// Check cookie first
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		refreshTokenStr = cookie.Value
	}

	// Fallback to Authorization header or JSON body
	if refreshTokenStr == "" {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		refreshTokenStr = req.RefreshToken
	}

	if refreshTokenStr == "" {
		RespondError(w, http.StatusUnauthorized, "refresh token required")
		return
	}

	claims, err := h.jwtMgr.ValidateToken(refreshTokenStr, "refresh")
	if err != nil {
		RespondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	user, err := h.userRepo.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, "user account not found")
		return
	}

	// OWASP A07: Session revocation check via token_version
	if claims.TokenVersion != user.TokenVersion {
		h.clearAuthCookies(w)
		RespondError(w, http.StatusUnauthorized, "session has been revoked, please sign in again")
		return
	}

	newAccessToken, accessExpiresIn, err := h.jwtMgr.GenerateAccessToken(user)
	if err != nil {
		log.Printf("refresh generate access token error: %v", err)
		RespondError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	newRefreshToken, refreshExpiresIn, err := h.jwtMgr.GenerateRefreshToken(user)
	if err != nil {
		log.Printf("refresh generate refresh token error: %v", err)
		RespondError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	h.setAuthCookies(w, newAccessToken, accessExpiresIn, newRefreshToken, refreshExpiresIn)

	resp := domain.RefreshTokenResponse{
		AccessToken: newAccessToken,
		ExpiresIn:   accessExpiresIn,
	}

	RespondOK(w, http.StatusOK, resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Invalidate session by incrementing user's token_version in DB
	var userID string
	if claims := auth.GetClaims(r.Context()); claims != nil {
		userID = claims.UserID
	} else if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		if c, err := h.jwtMgr.ValidateToken(cookie.Value, "refresh"); err == nil {
			userID = c.UserID
		}
	} else if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		if c, err := h.jwtMgr.ValidateToken(cookie.Value, "access"); err == nil {
			userID = c.UserID
		}
	}

	if userID != "" {
		if err := h.userRepo.IncrementTokenVersion(r.Context(), userID); err != nil {
			log.Printf("increment token_version on logout error for %s: %v", userID, err)
		}
	}

	h.clearAuthCookies(w)
	RespondOK(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userRepo.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "user not found")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"user": user.ToResponse(),
	})
}
