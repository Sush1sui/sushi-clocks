package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
	"github.com/sushi-clocks/backend/internal/sse"
	"github.com/sushi-clocks/backend/internal/utils"
)

type TimesheetHandler struct {
	timesheetRepo *repository.TimesheetRepository
	telemetryRepo *repository.TelemetryRepository
	hub           *sse.Hub
	behindProxy   bool
}

func NewTimesheetHandler(
	timesheetRepo *repository.TimesheetRepository,
	telemetryRepo *repository.TelemetryRepository,
	hub *sse.Hub,
	behindProxy bool,
) *TimesheetHandler {
	return &TimesheetHandler{
		timesheetRepo: timesheetRepo,
		telemetryRepo: telemetryRepo,
		hub:           hub,
		behindProxy:   behindProxy,
	}
}

// recordTelemetry runs asynchronously to record immutable device & network details into MongoDB
func (h *TimesheetHandler) recordTelemetry(r *http.Request, userID, companyID, action string, punchTime time.Time) {
	if h.telemetryRepo == nil {
		return
	}

	ip := utils.GetClientIP(r, h.behindProxy)
	uaStr := r.Header.Get("User-Agent")
	device := utils.ParseUserAgent(uaStr)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		telemetry := domain.PunchTelemetry{
			CompanyID: companyID,
			UserID:    userID,
			Action:    action,
			Timestamp: punchTime,
			IP:        ip,
			Device:    device.DeviceType,
			OS:        device.OS,
			Browser:   device.Browser,
		}

		if err := h.telemetryRepo.Insert(ctx, telemetry); err != nil {
			log.Printf("warning: failed to record punch telemetry: %v", err)
		}
	}()
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

	// 1. Record async MongoDB Atlas telemetry
	h.recordTelemetry(r, claims.UserID, claims.CompanyID, "clock_in", timesheet.ClockInTime)

	// 2. Broadcast real-time SSE event to all connected company managers
	if h.hub != nil {
		h.hub.Broadcast(claims.CompanyID, "punch", map[string]interface{}{
			"action":    "clock_in",
			"user_id":   claims.UserID,
			"email":     claims.Email,
			"timesheet": timesheet,
		})
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

	punchTime := time.Now()
	if timesheet.ClockOutTime != nil {
		punchTime = *timesheet.ClockOutTime
	}

	// 1. Record async MongoDB Atlas telemetry
	h.recordTelemetry(r, claims.UserID, claims.CompanyID, "clock_out", punchTime)

	// 2. Broadcast real-time SSE event
	if h.hub != nil {
		h.hub.Broadcast(claims.CompanyID, "punch", map[string]interface{}{
			"action":    "clock_out",
			"user_id":   claims.UserID,
			"email":     claims.Email,
			"timesheet": timesheet,
		})
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

// GetLiveRoster handles GET /api/v1/companies/{id}/attendance/roster
func (h *TimesheetHandler) GetLiveRoster(w http.ResponseWriter, r *http.Request) {
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

	if claims.SystemRole != domain.RoleSuperAdmin && claims.CompanyID != companyID {
		RespondError(w, http.StatusForbidden, "forbidden")
		return
	}

	roster, err := h.timesheetRepo.GetLiveRoster(r.Context(), companyID)
	if err != nil {
		log.Printf("get live roster error for %s: %v", companyID, err)
		RespondError(w, http.StatusInternalServerError, "failed to get live roster")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"roster": roster,
	})
}

// GetShiftHistory handles GET /api/v1/timesheets/history
func (h *TimesheetHandler) GetShiftHistory(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page := 1
	limit := 10
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	history, total, err := h.timesheetRepo.GetShiftHistory(r.Context(), claims.UserID, claims.CompanyID, page, limit)
	if err != nil {
		log.Printf("get shift history error for %s: %v", claims.UserID, err)
		RespondError(w, http.StatusInternalServerError, "failed to get shift history")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"history": history,
		"page":    page,
		"limit":   limit,
		"total":   total,
	})
}
