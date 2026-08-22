package domain

import "time"

const (
	TimesheetStatusActive    = "active"
	TimesheetStatusCompleted = "completed"
	TimesheetStatusFlagged   = "flagged_for_review"
)

type Timesheet struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	CompanyID    string     `json:"company_id"`
	ClockInTime  time.Time  `json:"clock_in_time"`
	ClockOutTime *time.Time `json:"clock_out_time,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
}

type AttendanceSummary struct {
	TotalStaff      int `json:"total_staff"`
	ClockedInCount  int `json:"clocked_in_count"`
	ClockedOutCount int `json:"clocked_out_count"`
}
