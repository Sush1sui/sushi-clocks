package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
)

// allowedStaffRoles defines which system_role values a Company Admin may assign.
// Escalation to "admin" or "super_admin" is explicitly blocked.
var allowedStaffRoles = map[string]bool{
	domain.RoleHR:       true,
	domain.RoleEmployee: true,
}

type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// GetCompanyUsers handles GET /api/v1/companies/{id}/users
// Access: Admin, HR, or Super Admin — cross-tenant blocked at handler level.
func (h *UserHandler) GetCompanyUsers(w http.ResponseWriter, r *http.Request) {
	companyID := r.PathValue("id")
	if companyID == "" {
		RespondError(w, http.StatusBadRequest, "company id is required")
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// OWASP A01: Super Admin may inspect any tenant; others are locked to their own.
	if claims.SystemRole != domain.RoleSuperAdmin && claims.CompanyID != companyID {
		RespondError(w, http.StatusForbidden, "access forbidden to other tenant companies")
		return
	}

	users, err := h.userRepo.GetUsersByCompanyID(r.Context(), companyID)
	if err != nil {
		log.Printf("get company users error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to retrieve staff list")
		return
	}

	// Convert to response slice (strips password_hash, token_version)
	responses := make([]domain.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"users": responses,
	})
}

// CreateCompanyUser handles POST /api/v1/companies/{id}/users
// Access: Admin only (HR cannot create accounts).
// OWASP A01: company_id sourced from JWT — never from request body.
// OWASP A04: role escalation to admin/super_admin explicitly rejected.
func (h *UserHandler) CreateCompanyUser(w http.ResponseWriter, r *http.Request) {
	// OWASP A04/A05: cap request body at 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	companyID := r.PathValue("id")
	if companyID == "" {
		RespondError(w, http.StatusBadRequest, "company id is required")
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// OWASP A01: Only Admin may create staff; HR and Employee are forbidden.
	if claims.SystemRole != domain.RoleAdmin {
		RespondError(w, http.StatusForbidden, "only company admins can create staff accounts")
		return
	}

	// OWASP A01: Admin can only create users in their own company.
	if claims.CompanyID != companyID {
		RespondError(w, http.StatusForbidden, "access forbidden to other tenant companies")
		return
	}

	var req domain.CreateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Sanitize input
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(req.Email)
	req.SystemRole = strings.TrimSpace(strings.ToLower(req.SystemRole))

	// Validate required fields
	if req.FirstName == "" {
		RespondError(w, http.StatusBadRequest, "first name is required")
		return
	}
	if req.LastName == "" {
		RespondError(w, http.StatusBadRequest, "last name is required")
		return
	}
	if req.Email == "" {
		RespondError(w, http.StatusBadRequest, "email is required")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid email address format")
		return
	}
	if len(req.Password) < 6 {
		RespondError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	// OWASP A04: Block role escalation — only "hr" and "employee" are permitted.
	if !allowedStaffRoles[req.SystemRole] {
		RespondError(w, http.StatusBadRequest, "system_role must be 'hr' or 'employee'")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("hash staff password error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to process credentials")
		return
	}

	// OWASP A01: company_id comes from JWT — not from the request path or body.
	user := &domain.User{
		CompanyID:           claims.CompanyID,
		FirstName:           req.FirstName,
		LastName:            req.LastName,
		Email:               req.Email,
		PasswordHash:        passwordHash,
		MobileNumber:        req.MobileNumber,
		SystemRole:          req.SystemRole,
		ReceiveAuditArchive: req.ReceiveAuditArchive,
	}

	if err := h.userRepo.CreateCompanyUser(r.Context(), user); err != nil {
		// Surface duplicate email as a user-facing 409 conflict
		if strings.Contains(err.Error(), "uq_users_email") || strings.Contains(err.Error(), "unique") {
			RespondError(w, http.StatusConflict, "an account with this email already exists")
			return
		}
		log.Printf("create company user error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to create staff account")
		return
	}

	RespondOK(w, http.StatusCreated, map[string]interface{}{
		"user": user.ToResponse(),
	})
}

// UpdateArchivePreference handles PATCH /api/v1/users/archive-preference
// Allows HR or Admin users to toggle whether they receive yearly audit archive emails.
func (h *UserHandler) UpdateArchivePreference(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		ReceiveArchive bool `json:"receive_audit_archive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.userRepo.UpdateUserArchivePreference(r.Context(), claims.UserID, claims.CompanyID, req.ReceiveArchive); err != nil {
		log.Printf("update archive preference error for %s: %v", claims.UserID, err)
		RespondError(w, http.StatusInternalServerError, "failed to update preference")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"receive_audit_archive": req.ReceiveArchive,
		"message":               "archive email preference updated",
	})
}

