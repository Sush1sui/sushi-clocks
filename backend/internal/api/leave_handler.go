package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
	"github.com/sushi-clocks/backend/internal/sse"
)

type LeaveHandler struct {
	leaveRepo *repository.LeaveRepository
	auditRepo *repository.AuditRepository
	hub       *sse.Hub
}

func NewLeaveHandler(
	leaveRepo *repository.LeaveRepository,
	auditRepo *repository.AuditRepository,
	hub *sse.Hub,
) *LeaveHandler {
	return &LeaveHandler{
		leaveRepo: leaveRepo,
		auditRepo: auditRepo,
		hub:       hub,
	}
}

// GetLeaveTypes handles GET /api/v1/leave/types
func (h *LeaveHandler) GetLeaveTypes(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	types, err := h.leaveRepo.GetLeaveTypesByCompany(r.Context(), claims.CompanyID)
	if err != nil {
		log.Printf("error fetching leave types: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to query leave types")
		return
	}

	RespondOK(w, http.StatusOK, map[string]any{"types": types})
}

// GetLeaveBalances handles GET /api/v1/leave/balances
func (h *LeaveHandler) GetLeaveBalances(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	targetUserID := claims.UserID
	requestedUser := r.URL.Query().Get("user_id")

	// Only Admin, HR, or SuperAdmin can view another user's balance
	if requestedUser != "" && requestedUser != claims.UserID {
		if claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleHR && claims.SystemRole != domain.RoleSuperAdmin {
			RespondError(w, http.StatusForbidden, "not authorized to view another user's balance")
			return
		}
		targetUserID = requestedUser
	}

	balances, period, err := h.leaveRepo.GetUserLeaveBalances(r.Context(), targetUserID, claims.CompanyID, time.Now().UTC())
	if err != nil {
		log.Printf("error fetching leave balances: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to calculate leave balances")
		return
	}

	RespondOK(w, http.StatusOK, map[string]any{
		"balances": balances,
		"period":   period,
	})
}

// SubmitLeaveRequest handles POST /api/v1/leave/requests
func (h *LeaveHandler) SubmitLeaveRequest(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var payload domain.SubmitLeaveRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if payload.LeaveTypeID == "" || payload.StartDate == "" || payload.EndDate == "" {
		RespondError(w, http.StatusBadRequest, "leave_type_id, start_date, and end_date are required")
		return
	}

	startDate, err := time.Parse("2006-01-02", payload.StartDate)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", payload.EndDate)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "invalid end_date format, expected YYYY-MM-DD")
		return
	}

	if endDate.Before(startDate) {
		RespondError(w, http.StatusBadRequest, "end_date cannot be earlier than start_date")
		return
	}

	// Calendar days calculation: (end_date - start_date) + 1
	durationDays := int(endDate.Sub(startDate).Hours()/24) + 1

	// Check for conflicting overlapping leave requests
	hasOverlap, err := h.leaveRepo.CheckOverlap(r.Context(), claims.UserID, startDate, endDate, nil)
	if err != nil {
		log.Printf("overlap check error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to check date overlap")
		return
	}
	if hasOverlap {
		RespondError(w, http.StatusConflict, "an active or pending leave request already covers these dates")
		return
	}

	// Option C Quota Enforcement
	balances, _, err := h.leaveRepo.GetUserLeaveBalances(r.Context(), claims.UserID, claims.CompanyID, time.Now().UTC())
	if err != nil {
		log.Printf("balance lookup error: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to calculate remaining allowance")
		return
	}

	var targetBalance *domain.LeaveBalance
	for _, b := range balances {
		if b.LeaveTypeID == payload.LeaveTypeID {
			targetBalance = &b
			break
		}
	}

	if targetBalance == nil {
		RespondError(w, http.StatusNotFound, "leave category not found or inactive")
		return
	}

	// Quota evaluation:
	// If paid: must have remaining >= duration
	// If unpaid and has positive allocated quota: must have remaining >= duration
	// If unpaid and 0 allocated quota: allowed (Option C flexible)
	if targetBalance.IsPaid {
		if durationDays > targetBalance.RemainingDays {
			RespondError(w, http.StatusBadRequest, fmt.Sprintf("insufficient paid leave balance: requested %d days, remaining %d days", durationDays, targetBalance.RemainingDays))
			return
		}
	} else if targetBalance.TotalAllocatedDays > 0 {
		if durationDays > targetBalance.RemainingDays {
			RespondError(w, http.StatusBadRequest, fmt.Sprintf("insufficient unpaid leave balance: requested %d days, remaining %d days", durationDays, targetBalance.RemainingDays))
			return
		}
	}

	var reasonPtr *string
	if payload.Reason != "" {
		reasonPtr = &payload.Reason
	}

	req := domain.LeaveRequest{
		CompanyID:   claims.CompanyID,
		UserID:      claims.UserID,
		LeaveTypeID: payload.LeaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		Status:      domain.LeaveStatusPending,
		Reason:      reasonPtr,
	}

	if err := h.leaveRepo.CreateLeaveRequest(r.Context(), &req); err != nil {
		log.Printf("failed to create leave request: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to submit leave request")
		return
	}

	// Record audit log asynchronously in MongoDB Atlas
	if h.auditRepo != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = h.auditRepo.Insert(ctx, domain.AuditLog{
				CompanyID:  claims.CompanyID,
				Collection: "leave_requests",
				Action:     "leave_requested",
				ActorID:    claims.UserID,
				TargetID:   req.ID,
				After:      req,
				Reason:     payload.Reason,
				Timestamp:  time.Now().UTC(),
			})
		}()
	}

	// Broadcast SSE real-time event
	if h.hub != nil {
		h.hub.Broadcast(claims.CompanyID, "leave_requested", map[string]any{
			"request_id":      req.ID,
			"user_id":         claims.UserID,
			"leave_type_name": targetBalance.LeaveTypeName,
			"start_date":      payload.StartDate,
			"end_date":        payload.EndDate,
			"calendar_days":   durationDays,
			"status":          domain.LeaveStatusPending,
		})
	}

	RespondOK(w, http.StatusCreated, map[string]any{
		"message":       "Leave request submitted successfully",
		"leave_request": req,
		"calendar_days": durationDays,
	})
}

// GetMyLeaveRequests handles GET /api/v1/leave/requests
func (h *LeaveHandler) GetMyLeaveRequests(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 1 {
			offset = (parsed - 1) * limit
		}
	}

	requests, total, err := h.leaveRepo.GetLeaveRequestsByUser(r.Context(), claims.UserID, claims.CompanyID, limit, offset)
	if err != nil {
		log.Printf("error fetching user leave requests: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to query leave history")
		return
	}

	RespondOK(w, http.StatusOK, map[string]any{
		"requests": requests,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetCompanyLeaveRequests handles GET /api/v1/companies/{id}/leave-requests
func (h *LeaveHandler) GetCompanyLeaveRequests(w http.ResponseWriter, r *http.Request) {
	companyID := r.PathValue("id")
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// OWASP A01 tenant check
	if claims.SystemRole != domain.RoleSuperAdmin && claims.CompanyID != companyID {
		RespondError(w, http.StatusForbidden, "cannot access another company's leave queue")
		return
	}

	statusFilter := r.URL.Query().Get("status")
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 1 {
			offset = (parsed - 1) * limit
		}
	}

	requests, total, err := h.leaveRepo.GetLeaveRequestsByCompany(r.Context(), companyID, statusFilter, limit, offset)
	if err != nil {
		log.Printf("error fetching company leave requests: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to query leave requests queue")
		return
	}

	RespondOK(w, http.StatusOK, map[string]any{
		"requests": requests,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// ResolveLeaveRequest handles PATCH /api/v1/leave/requests/{id}
func (h *LeaveHandler) ResolveLeaveRequest(w http.ResponseWriter, r *http.Request) {
	requestID := r.PathValue("id")
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var payload domain.ResolveLeaveRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if payload.Action != "approve" && payload.Action != "reject" {
		RespondError(w, http.StatusBadRequest, "action must be either 'approve' or 'reject'")
		return
	}

	targetStatus := domain.LeaveStatusApproved
	if payload.Action == "reject" {
		targetStatus = domain.LeaveStatusRejected
	}

	existing, err := h.leaveRepo.GetLeaveRequestByID(r.Context(), requestID, claims.CompanyID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "leave request not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to query leave request")
		return
	}

	// OWASP A04: Separation of duties — cannot review or resolve own leave request
	if existing.UserID == claims.UserID && claims.SystemRole != domain.RoleSuperAdmin {
		RespondError(w, http.StatusForbidden, "cannot review or resolve your own leave request")
		return
	}

	var notesPtr *string
	if payload.ReviewNotes != "" {
		notesPtr = &payload.ReviewNotes
	}

	updated, err := h.leaveRepo.ResolveLeaveRequest(r.Context(), requestID, claims.CompanyID, claims.UserID, targetStatus, notesPtr)
	if err != nil {
		log.Printf("error resolving leave request: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to update leave request status")
		return
	}

	// Record audit log asynchronously in MongoDB
	if h.auditRepo != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = h.auditRepo.Insert(ctx, domain.AuditLog{
				CompanyID:  claims.CompanyID,
				Collection: "leave_requests",
				Action:     "leave_resolved",
				ActorID:    claims.UserID,
				TargetID:   requestID,
				Before:     existing,
				After:      updated,
				Reason:     payload.ReviewNotes,
				Timestamp:  time.Now().UTC(),
			})
		}()
	}

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.Broadcast(claims.CompanyID, "leave_resolved", map[string]any{
			"request_id": requestID,
			"user_id":    updated.UserID,
			"action":     payload.Action,
			"status":     updated.Status,
		})
	}

	RespondOK(w, http.StatusOK, map[string]any{
		"message":       fmt.Sprintf("Leave request %sd successfully", payload.Action),
		"leave_request": updated,
	})
}

// GetLeavePolicy handles GET /api/v1/companies/{id}/leave-policy
func (h *LeaveHandler) GetLeavePolicy(w http.ResponseWriter, r *http.Request) {
	companyID := r.PathValue("id")
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if claims.SystemRole != domain.RoleSuperAdmin && claims.CompanyID != companyID {
		RespondError(w, http.StatusForbidden, "cannot access another company's leave policy")
		return
	}

	policy, err := h.leaveRepo.GetCompanyLeavePolicy(r.Context(), companyID)
	if err != nil {
		log.Printf("failed to get leave policy: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to query leave policy")
		return
	}

	RespondOK(w, http.StatusOK, map[string]any{"policy": policy})
}

// UpdateLeavePolicy handles PUT /api/v1/companies/{id}/leave-policy
func (h *LeaveHandler) UpdateLeavePolicy(w http.ResponseWriter, r *http.Request) {
	companyID := r.PathValue("id")
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Only Company Admin or SuperAdmin can update policy
	if claims.SystemRole != domain.RoleSuperAdmin && (claims.CompanyID != companyID || claims.SystemRole != domain.RoleAdmin) {
		RespondError(w, http.StatusForbidden, "only company administrators can update leave policy")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var payload domain.UpdateLeavePolicyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if payload.LeaveResetMonth < 1 || payload.LeaveResetMonth > 12 {
		RespondError(w, http.StatusBadRequest, "leave_reset_month must be between 1 and 12")
		return
	}
	if payload.LeaveResetDay < 1 || payload.LeaveResetDay > 31 {
		RespondError(w, http.StatusBadRequest, "leave_reset_day must be between 1 and 31")
		return
	}

	policy := domain.CompanyLeavePolicy{
		CompanyID:           companyID,
		LeaveResetMonth:     payload.LeaveResetMonth,
		LeaveResetDay:       payload.LeaveResetDay,
		AllowLeaveCarryover: payload.AllowLeaveCarryover,
		MaxCarryoverDays:    payload.MaxCarryoverDays,
	}

	if err := h.leaveRepo.UpdateCompanyLeavePolicy(r.Context(), &policy); err != nil {
		log.Printf("failed to update leave policy: %v", err)
		RespondError(w, http.StatusInternalServerError, "failed to update leave policy")
		return
	}

	// Record audit log
	if h.auditRepo != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = h.auditRepo.Insert(ctx, domain.AuditLog{
				CompanyID:  companyID,
				Collection: "companies",
				Action:     "leave_policy_updated",
				ActorID:    claims.UserID,
				TargetID:   companyID,
				After:      policy,
				Timestamp:  time.Now().UTC(),
			})
		}()
	}

	// Broadcast SSE event
	if h.hub != nil {
		h.hub.Broadcast(companyID, "leave_policy_updated", policy)
	}

	RespondOK(w, http.StatusOK, map[string]any{
		"message": "Leave policy updated successfully",
		"policy":  policy,
	})
}
