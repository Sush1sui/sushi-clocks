package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleHR         = "hr"
	RoleEmployee   = "employee"
)

type User struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"company_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	MobileNumber *string   `json:"mobile_number,omitempty"`
	SystemRole   string    `json:"system_role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Company struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CurrencyCode string    `json:"currency_code"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserResponse struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"company_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	MobileNumber *string   `json:"mobile_number,omitempty"`
	SystemRole   string    `json:"system_role"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:           u.ID,
		CompanyID:    u.CompanyID,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		Email:        u.Email,
		MobileNumber: u.MobileNumber,
		SystemRole:   u.SystemRole,
		CreatedAt:    u.CreatedAt,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User        UserResponse `json:"user"`
	AccessToken string       `json:"access_token"`
	ExpiresIn   int64        `json:"expires_in"` // seconds
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type JWTClaims struct {
	UserID     string `json:"user_id"`
	CompanyID  string `json:"company_id"`
	Email      string `json:"email"`
	SystemRole string `json:"system_role"`
	TokenType  string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}
