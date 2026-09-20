package domain

import "time"

const (
	LeaveStatusPending  = "pending"
	LeaveStatusApproved = "approved"
	LeaveStatusRejected = "rejected"
)

type LeaveType struct {
	ID               string    `json:"id"`
	CompanyID        string    `json:"company_id"`
	Name             string    `json:"name"`
	IsPaid           bool      `json:"is_paid"`
	AllowCarryover   bool      `json:"allow_carryover"`
	MaxCarryoverDays int       `json:"max_carryover_days"`
	CreatedAt        time.Time `json:"created_at"`
}

type LeaveRequest struct {
	ID               string     `json:"id"`
	CompanyID        string     `json:"company_id"`
	UserID           string     `json:"user_id"`
	LeaveTypeID      string     `json:"leave_type_id"`
	StartDate        time.Time  `json:"start_date"`
	EndDate          time.Time  `json:"end_date"`
	Status           string     `json:"status"`
	Reason           *string    `json:"reason,omitempty"`
	ReviewedByUserID *string    `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes      *string    `json:"review_notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type LeaveRequestWithDetails struct {
	LeaveRequest
	UserFirstName string `json:"user_first_name"`
	UserLastName  string `json:"user_last_name"`
	UserEmail     string `json:"user_email"`
	RoleName      string `json:"role_name"`
	LeaveTypeName string `json:"leave_type_name"`
	IsPaid        bool   `json:"is_paid"`
	CalendarDays  int    `json:"calendar_days"`
}

type LeaveBalance struct {
	LeaveTypeID        string `json:"leave_type_id"`
	LeaveTypeName      string `json:"leave_type_name"`
	IsPaid             bool   `json:"is_paid"`
	BaseAllocatedDays  int    `json:"base_allocated_days"`
	CarriedOverDays    int    `json:"carried_over_days"`
	TotalAllocatedDays int    `json:"total_allocated_days"`
	UsedDays           int    `json:"used_days"`
	PendingDays        int    `json:"pending_days"`
	RemainingDays      int    `json:"remaining_days"`
	IsFlexible         bool   `json:"is_flexible"` // true when unpaid with 0 quota (Option C)
}

type CompanyLeavePolicy struct {
	CompanyID           string `json:"company_id"`
	LeaveResetMonth     int    `json:"leave_reset_month"` // 1-12
	LeaveResetDay       int    `json:"leave_reset_day"`   // 1-31
	AllowLeaveCarryover bool   `json:"allow_leave_carryover"`
	MaxCarryoverDays    int    `json:"max_carryover_days"` // 0 = unlimited rollover
}

type LeavePeriod struct {
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	PrevStartDate time.Time `json:"prev_start_date"`
	PrevEndDate   time.Time `json:"prev_end_date"`
}

type SubmitLeaveRequestPayload struct {
	LeaveTypeID string `json:"leave_type_id"`
	StartDate   string `json:"start_date"` // YYYY-MM-DD
	EndDate     string `json:"end_date"`   // YYYY-MM-DD
	Reason      string `json:"reason"`
}

type ResolveLeaveRequestPayload struct {
	Action      string `json:"action"` // "approve" or "reject"
	ReviewNotes string `json:"review_notes,omitempty"`
}

type UpdateLeavePolicyPayload struct {
	LeaveResetMonth     int  `json:"leave_reset_month"`
	LeaveResetDay       int  `json:"leave_reset_day"`
	AllowLeaveCarryover bool `json:"allow_leave_carryover"`
	MaxCarryoverDays    int  `json:"max_carryover_days"`
}

type CreateLeaveTypePayload struct {
	Name             string `json:"name"`
	IsPaid           bool   `json:"is_paid"`
	AllowCarryover   bool   `json:"allow_carryover"`
	MaxCarryoverDays int    `json:"max_carryover_days"`
}
