package domain

import "time"

const (
	TimesheetStatusActive    = "active"
	TimesheetStatusCompleted = "completed"
	TimesheetStatusFlagged   = "flagged_for_review"
	TimesheetStatusRejected  = "rejected"
)

type Timesheet struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	CompanyID        string     `json:"company_id"`
	ClockInTime      time.Time  `json:"clock_in_time"`
	ClockOutTime     *time.Time `json:"clock_out_time,omitempty"`
	Status           string     `json:"status"`
	AdjustmentReason *string    `json:"adjustment_reason,omitempty"`
	ReviewedBy       *string    `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type TimesheetWithUser struct {
	Timesheet
	UserEmail     string `json:"user_email"`
	UserFirstName string `json:"user_first_name"`
	UserLastName  string `json:"user_last_name"`
	RoleName      string `json:"role_name"`
}

type AttendanceSummary struct {
	TotalStaff      int `json:"total_staff"`
	ClockedInCount  int `json:"clocked_in_count"`
	ClockedOutCount int `json:"clocked_out_count"`
}
