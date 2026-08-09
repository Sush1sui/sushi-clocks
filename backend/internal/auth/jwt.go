package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sushi-clocks/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrInvalidType  = errors.New("token type mismatch")
)

type JWTManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (m *JWTManager) GenerateAccessToken(user *domain.User) (string, int64, error) {
	expiresAt := time.Now().Add(m.accessTTL)
	claims := &domain.JWTClaims{
		UserID:     user.ID,
		CompanyID:  user.CompanyID,
		Email:      user.Email,
		SystemRole: user.SystemRole,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign access token error: %w", err)
	}

	return tokenString, int64(m.accessTTL.Seconds()), nil
}

func (m *JWTManager) GenerateRefreshToken(user *domain.User) (string, int64, error) {
	expiresAt := time.Now().Add(m.refreshTTL)
	claims := &domain.JWTClaims{
		UserID:     user.ID,
		CompanyID:  user.CompanyID,
		Email:      user.Email,
		SystemRole: user.SystemRole,
		TokenType:  "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign refresh token error: %w", err)
	}

	return tokenString, int64(m.refreshTTL.Seconds()), nil
}

func (m *JWTManager) ValidateToken(tokenString string, expectedType string) (*domain.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &domain.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*domain.JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if expectedType != "" && claims.TokenType != expectedType {
		return nil, ErrInvalidType
	}

	return claims, nil
}

// HashPassword hashes plaintext password with bcrypt cost 12
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash error: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares password against bcrypt hash
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
