package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sushi-clocks/backend/internal/domain"
)

type contextKey string

const ClaimsContextKey = contextKey("jwt_claims")

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func respondJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error:   message,
	})
}

// GetClaims retrieves JWT claims from context
func GetClaims(ctx context.Context) *domain.JWTClaims {
	claims, ok := ctx.Value(ClaimsContextKey).(*domain.JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

// ExtractToken extracts JWT token from Authorization header or cookie
func ExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to cookie
	if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// RequireAuth middleware verifies access token and injects claims into request context
func RequireAuth(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := ExtractToken(r)
			if tokenString == "" {
				respondJSONError(w, http.StatusUnauthorized, "authorization token required")
				return
			}

			claims, err := jwtManager.ValidateToken(tokenString, "access")
			if err != nil {
				respondJSONError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSuperAdmin middleware restricts endpoint to platform super admin
func RequireSuperAdmin(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		authMiddleware := RequireAuth(jwtManager)
		return authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil || claims.SystemRole != domain.RoleSuperAdmin {
				respondJSONError(w, http.StatusForbidden, "super admin privilege required")
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

// RequireRoles middleware restricts endpoint to users matching any of the specified roles
func RequireRoles(jwtManager *JWTManager, allowedRoles ...string) func(http.Handler) http.Handler {
	allowedMap := make(map[string]bool)
	for _, role := range allowedRoles {
		allowedMap[role] = true
	}

	return func(next http.Handler) http.Handler {
		authMiddleware := RequireAuth(jwtManager)
		return authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				respondJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			// Super Admin can access all role-restricted routes
			if claims.SystemRole == domain.RoleSuperAdmin || allowedMap[claims.SystemRole] {
				next.ServeHTTP(w, r)
				return
			}

			respondJSONError(w, http.StatusForbidden, "insufficient role permissions")
		}))
	}
}
