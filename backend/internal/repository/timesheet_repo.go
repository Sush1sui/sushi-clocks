package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

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
		SELECT id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
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
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
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
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
	`
	var t domain.Timesheet
	err = r.pool.QueryRow(ctx, query, userID, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
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
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
	`
	var t domain.Timesheet
	err = r.pool.QueryRow(ctx, query, active.ID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update clock-out timesheet error: %w", err)
	}

	return &t, nil
}

// GetCompanyAttendanceSummary calculates live headcount for a company
func (r *TimesheetRepository) GetCompanyAttendanceSummary(ctx context.Context, companyID string) (*domain.AttendanceSummary, error) {
	var totalStaff int
	totalQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE company_id = $1
	`
	if err := r.pool.QueryRow(ctx, totalQuery, companyID).Scan(&totalStaff); err != nil {
		return nil, fmt.Errorf("count total staff error: %w", err)
	}

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

// GetTimesheetByID fetches a single timesheet scoped to company_id (OWASP A01 tenant isolation)
func (r *TimesheetRepository) GetTimesheetByID(ctx context.Context, id, companyID string) (*domain.Timesheet, error) {
	query := `
		SELECT id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
		FROM timesheets
		WHERE id = $1 AND company_id = $2
	`
	var t domain.Timesheet
	err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get timesheet by id error: %w", err)
	}
	return &t, nil
}

// RequestAdjustment flags an existing completed timesheet for review with an explanation
func (r *TimesheetRepository) RequestAdjustment(ctx context.Context, userID, timesheetID, companyID, reason string) (*domain.Timesheet, error) {
	query := `
		UPDATE timesheets
		SET status = 'flagged_for_review', adjustment_reason = $1
		WHERE id = $2 AND user_id = $3 AND company_id = $4
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
	`
	var t domain.Timesheet
	err := r.pool.QueryRow(ctx, query, reason, timesheetID, userID, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("request adjustment error: %w", err)
	}
	return &t, nil
}

// ApproveAdjustment sets timesheet status to completed with optional clock in/out updates
func (r *TimesheetRepository) ApproveAdjustment(ctx context.Context, reviewerID, timesheetID, companyID string, newClockIn, newClockOut *time.Time) (*domain.Timesheet, error) {
	query := `
		UPDATE timesheets
		SET status = 'completed',
		    clock_in_time = COALESCE($1, clock_in_time),
		    clock_out_time = COALESCE($2, clock_out_time),
		    reviewed_by = $3,
		    reviewed_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND company_id = $5
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
	`
	var t domain.Timesheet
	err := r.pool.QueryRow(ctx, query, newClockIn, newClockOut, reviewerID, timesheetID, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("approve adjustment error: %w", err)
	}
	return &t, nil
}

// RejectAdjustment marks timesheet as rejected with reviewer audit trail
func (r *TimesheetRepository) RejectAdjustment(ctx context.Context, reviewerID, timesheetID, companyID string) (*domain.Timesheet, error) {
	query := `
		UPDATE timesheets
		SET status = 'rejected',
		    reviewed_by = $1,
		    reviewed_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND company_id = $3
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
	`
	var t domain.Timesheet
	err := r.pool.QueryRow(ctx, query, reviewerID, timesheetID, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reject adjustment error: %w", err)
	}
	return &t, nil
}

// DirectOverride allows HR/Admin to directly overwrite times and status
func (r *TimesheetRepository) DirectOverride(ctx context.Context, reviewerID, timesheetID, companyID string, newClockIn time.Time, newClockOut *time.Time, reason string) (*domain.Timesheet, error) {
	query := `
		UPDATE timesheets
		SET clock_in_time = $1,
		    clock_out_time = $2,
		    adjustment_reason = $3,
		    status = 'completed',
		    reviewed_by = $4,
		    reviewed_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND company_id = $6
		RETURNING id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
	`
	var t domain.Timesheet
	err := r.pool.QueryRow(ctx, query, newClockIn, newClockOut, reason, reviewerID, timesheetID, companyID).Scan(
		&t.ID,
		&t.UserID,
		&t.CompanyID,
		&t.ClockInTime,
		&t.ClockOutTime,
		&t.Status,
		&t.AdjustmentReason,
		&t.ReviewedBy,
		&t.ReviewedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("direct override error: %w", err)
	}
	return &t, nil
}

// GetShiftHistory fetches paginated personal timesheets for a user
func (r *TimesheetRepository) GetShiftHistory(ctx context.Context, userID, companyID string, page, limit int) ([]domain.Timesheet, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM timesheets
		WHERE user_id = $1 AND company_id = $2
	`
	if err := r.pool.QueryRow(ctx, countQuery, userID, companyID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user shift history error: %w", err)
	}

	query := `
		SELECT id, user_id, company_id, clock_in_time, clock_out_time, status, adjustment_reason, reviewed_by, reviewed_at, created_at
		FROM timesheets
		WHERE user_id = $1 AND company_id = $2
		ORDER BY clock_in_time DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, userID, companyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query shift history error: %w", err)
	}
	defer rows.Close()

	list := make([]domain.Timesheet, 0)
	for rows.Next() {
		var t domain.Timesheet
		if err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.CompanyID,
			&t.ClockInTime,
			&t.ClockOutTime,
			&t.Status,
			&t.AdjustmentReason,
			&t.ReviewedBy,
			&t.ReviewedAt,
			&t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan shift history error: %w", err)
		}
		list = append(list, t)
	}

	return list, total, nil
}

// GetLiveRoster fetches all currently active shifts in the company with user details
func (r *TimesheetRepository) GetLiveRoster(ctx context.Context, companyID string) ([]domain.TimesheetWithUser, error) {
	query := `
		SELECT 
			t.id, t.user_id, t.company_id, t.clock_in_time, t.clock_out_time, t.status, 
			t.adjustment_reason, t.reviewed_by, t.reviewed_at, t.created_at,
			u.email, u.first_name, u.last_name, COALESCE(cr.name, u.system_role) as role_name
		FROM timesheets t
		JOIN users u ON t.user_id = u.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN company_roles cr ON ur.role_id = cr.id
		WHERE t.company_id = $1 AND t.status = 'active'
		ORDER BY t.clock_in_time ASC
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("query live roster error: %w", err)
	}
	defer rows.Close()

	roster := make([]domain.TimesheetWithUser, 0)
	for rows.Next() {
		var tu domain.TimesheetWithUser
		if err := rows.Scan(
			&tu.ID,
			&tu.UserID,
			&tu.CompanyID,
			&tu.ClockInTime,
			&tu.ClockOutTime,
			&tu.Status,
			&tu.AdjustmentReason,
			&tu.ReviewedBy,
			&tu.ReviewedAt,
			&tu.CreatedAt,
			&tu.UserEmail,
			&tu.UserFirstName,
			&tu.UserLastName,
			&tu.RoleName,
		); err != nil {
			return nil, fmt.Errorf("scan live roster error: %w", err)
		}
		roster = append(roster, tu)
	}

	return roster, nil
}

// GetPendingAdjustments retrieves all timesheets flagged for review in a company
func (r *TimesheetRepository) GetPendingAdjustments(ctx context.Context, companyID string) ([]domain.TimesheetWithUser, error) {
	query := `
		SELECT 
			t.id, t.user_id, t.company_id, t.clock_in_time, t.clock_out_time, t.status, 
			t.adjustment_reason, t.reviewed_by, t.reviewed_at, t.created_at,
			u.email, u.first_name, u.last_name, COALESCE(cr.name, u.system_role) as role_name
		FROM timesheets t
		JOIN users u ON t.user_id = u.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN company_roles cr ON ur.role_id = cr.id
		WHERE t.company_id = $1 AND t.status = 'flagged_for_review'
		ORDER BY t.clock_in_time DESC
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("query pending adjustments error: %w", err)
	}
	defer rows.Close()

	adjustments := make([]domain.TimesheetWithUser, 0)
	for rows.Next() {
		var tu domain.TimesheetWithUser
		if err := rows.Scan(
			&tu.ID,
			&tu.UserID,
			&tu.CompanyID,
			&tu.ClockInTime,
			&tu.ClockOutTime,
			&tu.Status,
			&tu.AdjustmentReason,
			&tu.ReviewedBy,
			&tu.ReviewedAt,
			&tu.CreatedAt,
			&tu.UserEmail,
			&tu.UserFirstName,
			&tu.UserLastName,
			&tu.RoleName,
		); err != nil {
			return nil, fmt.Errorf("scan pending adjustments error: %w", err)
		}
		adjustments = append(adjustments, tu)
	}

	return adjustments, nil
}
