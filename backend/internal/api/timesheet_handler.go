package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
)

type TimesheetHandler struct {
	timesheetRepo *repository.TimesheetRepository
}

func NewTimesheetHandler(timesheetRepo *repository.TimesheetRepository) *TimesheetHandler {
	return &TimesheetHandler{timesheetRepo: timesheetRepo}
}

// ClockIn handles POST /api/v1/timesheets/clock-in
func (h *TimesheetHandler) ClockIn(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	timesheet, err := h.timesheetRepo.ClockIn(r.Context(), claims.UserID, claims.CompanyID)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyClockedIn) {
			RespondError(w, http.StatusConflict, "you already have an active clock-in shift")
			return
		}
		log.Printf("clock-in error for user %s: %v", claims.UserID, err)
		RespondError(w, http.StatusInternalServerError, "failed to clock in")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"message":   "clocked in successfully",
		"timesheet": timesheet,
	})
}

// ClockOut handles POST /api/v1/timesheets/clock-out
func (h *TimesheetHandler) ClockOut(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	timesheet, err := h.timesheetRepo.ClockOut(r.Context(), claims.UserID, claims.CompanyID)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveShift) {
			RespondError(w, http.StatusNotFound, "no active shift found to clock out")
			return
		}
		log.Printf("clock-out error for user %s: %v", claims.UserID, err)
		RespondError(w, http.StatusInternalServerError, "failed to clock out")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"message":   "clocked out successfully",
		"timesheet": timesheet,
	})
}

// GetStatus handles GET /api/v1/timesheets/status
func (h *TimesheetHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	timesheet, err := h.timesheetRepo.GetActiveShift(r.Context(), claims.UserID, claims.CompanyID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondOK(w, http.StatusOK, map[string]interface{}{
				"is_clocked_in": false,
				"shift":         nil,
			})
			return
		}
		log.Printf("get active shift error for user %s: %v", claims.UserID, err)
		RespondError(w, http.StatusInternalServerError, "failed to query shift status")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"is_clocked_in": true,
		"shift":         timesheet,
	})
}

// GetCompanySummary handles GET /api/v1/companies/{id}/attendance/summary
func (h *TimesheetHandler) GetCompanySummary(w http.ResponseWriter, r *http.Request) {
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

	// Super admin or members of the company with admin/hr role
	if claims.SystemRole != domain.RoleSuperAdmin && claims.CompanyID != companyID {
		RespondError(w, http.StatusForbidden, "forbidden")
		return
	}

	summary, err := h.timesheetRepo.GetCompanyAttendanceSummary(r.Context(), companyID)
	if err != nil {
		log.Printf("get company attendance summary error for %s: %v", companyID, err)
		RespondError(w, http.StatusInternalServerError, "failed to get attendance summary")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"summary": summary,
	})
}
