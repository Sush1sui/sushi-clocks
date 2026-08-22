package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sushi-clocks/backend/internal/domain"
)

var (
	ErrAlreadyClockedIn = errors.New("user already has an active shift")
	ErrNoActiveShift    = errors.New("no active shift found to clock out")
)

type TimesheetRepository struct {
	pool *pgxpool.Pool
}

func NewTimesheetRepository(pool *pgxpool.Pool) *TimesheetRepository {
	return &TimesheetRepository{pool: pool}
}

// GetActiveShift retrieves the user's ongoing active shift, if any
func (r *TimesheetRepository) GetActiveShift(ctx context.Context, userID, companyID string) (*domain.Timesheet, error) {
	query := `
		SELECT id, user_id, company_id, clock_in_time, clock_out_time, status, created_at
		FROM timesheets
		WHERE user_id = $1 AND company_id = $2 AND status = 'active'
		ORDER BY clock_in_time DESC
		LIMIT 1
	`
	var t domain.Timesheet
	err := r.pool.QueryRow(ctx, query, userID, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query active shift error: %w", err)
	}
	return &t, nil
}

// ClockIn starts a new shift for the user
func (r *TimesheetRepository) ClockIn(ctx context.Context, userID, companyID string) (*domain.Timesheet, error) {
	// Check if already active
	active, err := r.GetActiveShift(ctx, userID, companyID)
	if err == nil && active != nil {
		return nil, ErrAlreadyClockedIn
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("check active shift error: %w", err)
	}

	query := `
		INSERT INTO timesheets (user_id, company_id, clock_in_time, status)
		VALUES ($1, $2, CURRENT_TIMESTAMP, 'active')
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, created_at
	`
	var t domain.Timesheet
	err = r.pool.QueryRow(ctx, query, userID, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert clock-in timesheet error: %w", err)
	}

	return &t, nil
}

// ClockOut ends the user's ongoing active shift
func (r *TimesheetRepository) ClockOut(ctx context.Context, userID, companyID string) (*domain.Timesheet, error) {
	active, err := r.GetActiveShift(ctx, userID, companyID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNoActiveShift
		}
		return nil, fmt.Errorf("query active shift before clock-out error: %w", err)
	}

	query := `
		UPDATE timesheets
		SET clock_out_time = CURRENT_TIMESTAMP, status = 'completed'
		WHERE id = $1
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, created_at
	`
	var t domain.Timesheet
	err = r.pool.QueryRow(ctx, query, active.ID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update clock-out timesheet error: %w", err)
	}

	return &t, nil
}

// GetCompanyAttendanceSummary calculates live headcount for a company
func (r *TimesheetRepository) GetCompanyAttendanceSummary(ctx context.Context, companyID string) (*domain.AttendanceSummary, error) {
	// Total registered staff in company
	var totalStaff int
	totalQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE company_id = $1
	`
	if err := r.pool.QueryRow(ctx, totalQuery, companyID).Scan(&totalStaff); err != nil {
		return nil, fmt.Errorf("count total staff error: %w", err)
	}

	// Currently clocked-in distinct staff in company
	var clockedInCount int
	clockedInQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM timesheets
		WHERE company_id = $1 AND status = 'active'
	`
	if err := r.pool.QueryRow(ctx, clockedInQuery, companyID).Scan(&clockedInCount); err != nil {
		return nil, fmt.Errorf("count clocked-in staff error: %w", err)
	}

	clockedOutCount := totalStaff - clockedInCount
	if clockedOutCount < 0 {
		clockedOutCount = 0
	}

	return &domain.AttendanceSummary{
		TotalStaff:      totalStaff,
		ClockedInCount:  clockedInCount,
		ClockedOutCount: clockedOutCount,
	}, nil
}
