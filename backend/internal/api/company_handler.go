package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
)

type CompanyHandler struct {
	companyRepo *repository.CompanyRepository
}

func NewCompanyHandler(companyRepo *repository.CompanyRepository) *CompanyHandler {
	return &CompanyHandler{companyRepo: companyRepo}
}

// GetCompanies handles GET /api/v1/companies (Super Admin Only)
func (h *CompanyHandler) GetCompanies(w http.ResponseWriter, r *http.Request) {
	companies, err := h.companyRepo.GetAllCompaniesWithStats(r.Context())
	if err != nil {
		log.Printf("get all companies error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to retrieve companies")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"companies": companies,
	})
}

// CreateCompany handles POST /api/v1/companies (Super Admin Only)
func (h *CompanyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.CurrencyCode = strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	req.Timezone = strings.TrimSpace(req.Timezone)
	req.AdminEmail = strings.TrimSpace(req.AdminEmail)

	if req.Name == "" {
		RespondError(w, http.StatusBadRequest, "company name is required")
		return
	}
	if req.CurrencyCode == "" || len(req.CurrencyCode) > 3 {
		RespondError(w, http.StatusBadRequest, "valid 3-letter currency code is required (e.g., USD, PHP, EUR)")
		return
	}
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	company := &domain.Company{
		Name:         req.Name,
		CurrencyCode: req.CurrencyCode,
		Timezone:     req.Timezone,
	}

	var admin *domain.User
	if req.AdminEmail != "" {
		if _, err := mail.ParseAddress(req.AdminEmail); err != nil {
			RespondError(w, http.StatusBadRequest, "invalid admin email address format")
			return
		}
		if req.AdminPassword == "" || len(req.AdminPassword) < 6 {
			RespondError(w, http.StatusBadRequest, "admin password must be at least 6 characters")
			return
		}
		if req.AdminFirstName == "" {
			req.AdminFirstName = "Admin"
		}
		if req.AdminLastName == "" {
			req.AdminLastName = "User"
		}

		passwordHash, err := auth.HashPassword(req.AdminPassword)
		if err != nil {
			log.Printf("hash admin password error: %v", err)
			RespondError(w, http.StatusInternalServerError, "failed to process admin credentials")
			return
		}

		admin = &domain.User{
			FirstName:    req.AdminFirstName,
			LastName:     req.AdminLastName,
			Email:        req.AdminEmail,
			PasswordHash: passwordHash,
			SystemRole:   domain.RoleAdmin,
		}
	}

	if err := h.companyRepo.CreateCompanyWithAdmin(r.Context(), company, admin); err != nil {
		log.Printf("create company with admin error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to create company")
		return
	}

	resp := map[string]interface{}{
		"company": company,
	}
	if admin != nil {
		resp["admin"] = admin.ToResponse()
	}

	RespondOK(w, http.StatusCreated, resp)
}

// GetCompanyByID handles GET /api/v1/companies/{id} (Super Admin or Company Member)
func (h *CompanyHandler) GetCompanyByID(w http.ResponseWriter, r *http.Request) {
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

	// RBAC Check: Super Admin OR caller belongs to this company
	if claims.SystemRole != domain.RoleSuperAdmin && claims.CompanyID != companyID {
		RespondError(w, http.StatusForbidden, "access forbidden to other tenant companies")
		return
	}

	company, err := h.companyRepo.GetCompanyByID(r.Context(), companyID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "company not found")
			return
		}
		log.Printf("get company by id error: %v", err)
		RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"company": company,
	})
}
