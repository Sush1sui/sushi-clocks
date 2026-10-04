package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/service"
)

type PayrollHandler struct {
	service *service.PayrollService
}

func NewPayrollHandler(service *service.PayrollService) *PayrollHandler {
	return &PayrollHandler{service: service}
}

// resolveCompanyAndDates validates tenant isolation and parses the period date range.
func (h *PayrollHandler) resolveCompanyAndDates(r *http.Request) (string, time.Time, time.Time, error) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("unauthorized")
	}

	companyID := r.PathValue("id")
	if companyID == "" {
		companyID = r.URL.Query().Get("company_id")
	}
	if companyID == "" {
		companyID = claims.CompanyID
	}

	if claims.SystemRole != domain.RoleSuperAdmin && claims.CompanyID != companyID {
		return "", time.Time{}, time.Time{}, fmt.Errorf("forbidden")
	}

	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	if startDateStr == "" || endDateStr == "" {
		return "", time.Time{}, time.Time{}, fmt.Errorf("start_date and end_date are required (YYYY-MM-DD)")
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("invalid start_date format (must be YYYY-MM-DD)")
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("invalid end_date format (must be YYYY-MM-DD)")
	}

	if endDate.Before(startDate) {
		return "", time.Time{}, time.Time{}, fmt.Errorf("end_date cannot be before start_date")
	}

	if endDate.Sub(startDate) > 366*24*time.Hour {
		return "", time.Time{}, time.Time{}, fmt.Errorf("date range exceeds maximum limit of 366 days")
	}

	return companyID, startDate, endDate, nil
}

// Calculate handles GET /api/v1/payroll/calculate and GET /api/v1/companies/{id}/payroll/calculate
func (h *PayrollHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	companyID, startDate, endDate, err := h.resolveCompanyAndDates(r)
	if err != nil {
		switch err.Error() {
		case "unauthorized":
			RespondError(w, http.StatusUnauthorized, "unauthorized")
		case "forbidden":
			RespondError(w, http.StatusForbidden, "forbidden")
		default:
			RespondError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	preview, err := h.service.Calculate(r.Context(), companyID, startDate, endDate)
	if err != nil {
		log.Printf("payroll calculation error for company %s: %v", companyID, err)
		RespondError(w, http.StatusInternalServerError, "failed to calculate payroll")
		return
	}

	RespondOK(w, http.StatusOK, preview)
}

// Export handles GET /api/v1/payroll/export and GET /api/v1/companies/{id}/payroll/export
func (h *PayrollHandler) Export(w http.ResponseWriter, r *http.Request) {
	companyID, startDate, endDate, err := h.resolveCompanyAndDates(r)
	if err != nil {
		switch err.Error() {
		case "unauthorized":
			RespondError(w, http.StatusUnauthorized, "unauthorized")
		case "forbidden":
			RespondError(w, http.StatusForbidden, "forbidden")
		default:
			RespondError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	preview, err := h.service.Calculate(r.Context(), companyID, startDate, endDate)
	if err != nil {
		log.Printf("payroll export calculation error for company %s: %v", companyID, err)
		RespondError(w, http.StatusInternalServerError, "failed to calculate payroll for export")
		return
	}

	filename := service.PayrollFilename(startDate)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	if err := service.BuildPayrollCSV(w, preview); err != nil {
		log.Printf("streaming payroll csv error: %v", err)
	}
}
