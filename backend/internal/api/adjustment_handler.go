package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
	"github.com/sushi-clocks/backend/internal/sse"
)

type AdjustmentHandler struct {
	timesheetRepo *repository.TimesheetRepository
	auditRepo     *repository.AuditRepository
	hub           *sse.Hub
}

func NewAdjustmentHandler(
	timesheetRepo *repository.TimesheetRepository,
	auditRepo *repository.AuditRepository,
	hub *sse.Hub,
) *AdjustmentHandler {
	return &AdjustmentHandler{
		timesheetRepo: timesheetRepo,
		auditRepo:     auditRepo,
		hub:           hub,
	}
}

type RequestAdjustmentPayload struct {
	Reason string `json:"reason"`
}

type ResolveAdjustmentPayload struct {
	Action      string     `json:"action"` // "approve" or "reject"
	NewClockIn  *time.Time `json:"new_clock_in,omitempty"`
	NewClockOut *time.Time `json:"new_clock_out,omitempty"`
}

type DirectOverridePayload struct {
	ClockInTime  time.Time  `json:"clock_in_time"`
	ClockOutTime *time.Time `json:"clock_out_time,omitempty"`
	Reason       string     `json:"reason"`
}

// RequestAdjustment handles POST /api/v1/timesheets/{id}/adjustment-request
func (h *AdjustmentHandler) RequestAdjustment(w http.ResponseWriter, r *http.Request) {
	timesheetID := r.PathValue("id")
	if timesheetID == "" {
		RespondError(w, http.StatusBadRequest, "timesheet id required")
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var payload RequestAdjustmentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if payload.Reason == "" {
		RespondError(w, http.StatusBadRequest, "reason for adjustment is required")
		return
	}

	// Fetch existing timesheet for audit diff & ownership validation
	existing, err := h.timesheetRepo.GetTimesheetByID(r.Context(), timesheetID, claims.CompanyID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "timesheet not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to query timesheet")
		return
	}

	if existing.UserID != claims.UserID && claims.SystemRole != domain.RoleSuperAdmin && claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleHR {
		RespondError(w, http.StatusForbidden, "cannot request adjustment for another user's timesheet")
		return
	}

	updated, err := h.timesheetRepo.RequestAdjustment(r.Context(), existing.UserID, timesheetID, claims.CompanyID, payload.Reason)
	if err != nil {
		log.Printf("request adjustment error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to record adjustment request")
		return
	}

	// Record audit log asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if h.auditRepo != nil {
			_ = h.auditRepo.Insert(ctx, domain.AuditLog{
				CompanyID:  claims.CompanyID,
				Collection: "timesheets",
				Action:     "adjust_request",
				ActorID:    claims.UserID,
				TargetID:   timesheetID,
				Before:     existing,
				After:      updated,
				Reason:     payload.Reason,
				Timestamp:  time.Now(),
			})
		}
	}()

	// Broadcast SSE update
	if h.hub != nil {
		h.hub.Broadcast(claims.CompanyID, "adjustment_request", map[string]interface{}{
			"timesheet_id": timesheetID,
			"user_id":      claims.UserID,
			"reason":       payload.Reason,
			"timesheet":    updated,
		})
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"message":   "adjustment requested successfully",
		"timesheet": updated,
	})
}

// GetCompanyAdjustments handles GET /api/v1/companies/{id}/adjustments (Admin & HR)
func (h *AdjustmentHandler) GetCompanyAdjustments(w http.ResponseWriter, r *http.Request) {
	companyID := r.PathValue("id")
	if companyID == "" {
		RespondError(w, http.StatusBadRequest, "company id required")
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

	adjustments, err := h.timesheetRepo.GetPendingAdjustments(r.Context(), companyID)
	if err != nil {
		log.Printf("get pending adjustments error for %s: %v", companyID, err)
		RespondError(w, http.StatusInternalServerError, "failed to query pending adjustments")
		return
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"adjustments": adjustments,
	})
}

// ResolveAdjustment handles PATCH /api/v1/timesheets/adjustments/{id} (Admin & HR)
func (h *AdjustmentHandler) ResolveAdjustment(w http.ResponseWriter, r *http.Request) {
	timesheetID := r.PathValue("id")
	if timesheetID == "" {
		RespondError(w, http.StatusBadRequest, "timesheet id required")
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var payload ResolveAdjustmentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.timesheetRepo.GetTimesheetByID(r.Context(), timesheetID, claims.CompanyID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "timesheet not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to query timesheet")
		return
	}

	// OWASP A04: Separation of duties — cannot review or resolve own timesheet adjustment
	if existing.UserID == claims.UserID && claims.SystemRole != domain.RoleSuperAdmin {
		RespondError(w, http.StatusForbidden, "cannot review or resolve your own timesheet adjustment")
		return
	}

	var updated *domain.Timesheet
	actionName := ""

	if payload.Action == "approve" {
		actionName = "adjust_approve"
		updated, err = h.timesheetRepo.ApproveAdjustment(r.Context(), claims.UserID, timesheetID, claims.CompanyID, payload.NewClockIn, payload.NewClockOut)
	} else if payload.Action == "reject" {
		actionName = "adjust_reject"
		updated, err = h.timesheetRepo.RejectAdjustment(r.Context(), claims.UserID, timesheetID, claims.CompanyID)
	} else {
		RespondError(w, http.StatusBadRequest, "action must be 'approve' or 'reject'")
		return
	}

	if err != nil {
		log.Printf("resolve adjustment error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to resolve adjustment")
		return
	}

	// Record audit log asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if h.auditRepo != nil {
			_ = h.auditRepo.Insert(ctx, domain.AuditLog{
				CompanyID:  claims.CompanyID,
				Collection: "timesheets",
				Action:     actionName,
				ActorID:    claims.UserID,
				TargetID:   timesheetID,
				Before:     existing,
				After:      updated,
				Timestamp:  time.Now(),
			})
		}
	}()

	// Broadcast SSE
	if h.hub != nil {
		h.hub.Broadcast(claims.CompanyID, "adjustment_resolved", map[string]interface{}{
			"timesheet_id": timesheetID,
			"action":       payload.Action,
			"reviewer_id":  claims.UserID,
			"timesheet":    updated,
		})
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"message":   "adjustment " + payload.Action + "d successfully",
		"timesheet": updated,
	})
}

// DirectOverride handles PUT /api/v1/timesheets/{id} (Admin & HR)
func (h *AdjustmentHandler) DirectOverride(w http.ResponseWriter, r *http.Request) {
	timesheetID := r.PathValue("id")
	if timesheetID == "" {
		RespondError(w, http.StatusBadRequest, "timesheet id required")
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var payload DirectOverridePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if payload.ClockInTime.IsZero() {
		RespondError(w, http.StatusBadRequest, "valid clock_in_time is required")
		return
	}

	if payload.Reason == "" {
		RespondError(w, http.StatusBadRequest, "reason for direct override is required")
		return
	}

	existing, err := h.timesheetRepo.GetTimesheetByID(r.Context(), timesheetID, claims.CompanyID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "timesheet not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to query timesheet")
		return
	}

	updated, err := h.timesheetRepo.DirectOverride(r.Context(), claims.UserID, timesheetID, claims.CompanyID, payload.ClockInTime, payload.ClockOutTime, payload.Reason)
	if err != nil {
		log.Printf("direct override error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to override timesheet")
		return
	}

	// Record audit log asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if h.auditRepo != nil {
			_ = h.auditRepo.Insert(ctx, domain.AuditLog{
				CompanyID:  claims.CompanyID,
				Collection: "timesheets",
				Action:     "direct_override",
				ActorID:    claims.UserID,
				TargetID:   timesheetID,
				Before:     existing,
				After:      updated,
				Reason:     payload.Reason,
				Timestamp:  time.Now(),
			})
		}
	}()

	// Broadcast SSE
	if h.hub != nil {
		h.hub.Broadcast(claims.CompanyID, "timesheet_updated", map[string]interface{}{
			"timesheet_id": timesheetID,
			"action":       "direct_override",
			"timesheet":    updated,
		})
	}

	RespondOK(w, http.StatusOK, map[string]interface{}{
		"message":   "timesheet updated successfully",
		"timesheet": updated,
	})
}
