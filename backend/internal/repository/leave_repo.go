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

type LeaveRepository struct {
	pool *pgxpool.Pool
}

func NewLeaveRepository(pool *pgxpool.Pool) *LeaveRepository {
	return &LeaveRepository{pool: pool}
}

// CalculateLeavePeriod computes the active and previous leave cycles given a company's reset month and day.
func CalculateLeavePeriod(resetMonth, resetDay int, now time.Time) domain.LeavePeriod {
	if resetMonth < 1 || resetMonth > 12 {
		resetMonth = 1
	}
	if resetDay < 1 || resetDay > 31 {
		resetDay = 1
	}

	loc := now.Location()
	thisYearReset := time.Date(now.Year(), time.Month(resetMonth), resetDay, 0, 0, 0, 0, loc)

	var periodStart, periodEnd, prevPeriodStart, prevPeriodEnd time.Time

	if now.Before(thisYearReset) {
		periodStart = time.Date(now.Year()-1, time.Month(resetMonth), resetDay, 0, 0, 0, 0, loc)
		periodEnd = thisYearReset.AddDate(0, 0, -1)
		prevPeriodStart = time.Date(now.Year()-2, time.Month(resetMonth), resetDay, 0, 0, 0, 0, loc)
		prevPeriodEnd = periodStart.AddDate(0, 0, -1)
	} else {
		periodStart = thisYearReset
		periodEnd = time.Date(now.Year()+1, time.Month(resetMonth), resetDay, 0, 0, 0, 0, loc).AddDate(0, 0, -1)
		prevPeriodStart = time.Date(now.Year()-1, time.Month(resetMonth), resetDay, 0, 0, 0, 0, loc)
		prevPeriodEnd = periodStart.AddDate(0, 0, -1)
	}

	return domain.LeavePeriod{
		StartDate:     periodStart,
		EndDate:       periodEnd,
		PrevStartDate: prevPeriodStart,
		PrevEndDate:   prevPeriodEnd,
	}
}

func (r *LeaveRepository) GetCompanyLeavePolicy(ctx context.Context, companyID string) (*domain.CompanyLeavePolicy, error) {
	query := `
		SELECT id, leave_reset_month, leave_reset_day, allow_leave_carryover, max_carryover_days
		FROM companies
		WHERE id = $1
	`
	var p domain.CompanyLeavePolicy
	err := r.pool.QueryRow(ctx, query, companyID).Scan(
		&p.CompanyID,
		&p.LeaveResetMonth,
		&p.LeaveResetDay,
		&p.AllowLeaveCarryover,
		&p.MaxCarryoverDays,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query leave policy: %w", err)
	}
	return &p, nil
}

func (r *LeaveRepository) UpdateCompanyLeavePolicy(ctx context.Context, policy *domain.CompanyLeavePolicy) error {
	query := `
		UPDATE companies
		SET leave_reset_month = $1,
		    leave_reset_day = $2,
		    allow_leave_carryover = $3,
		    max_carryover_days = $4
		WHERE id = $5
	`
	cmd, err := r.pool.Exec(ctx, query,
		policy.LeaveResetMonth,
		policy.LeaveResetDay,
		policy.AllowLeaveCarryover,
		policy.MaxCarryoverDays,
		policy.CompanyID,
	)
	if err != nil {
		return fmt.Errorf("failed to update leave policy: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *LeaveRepository) GetLeaveTypesByCompany(ctx context.Context, companyID string) ([]domain.LeaveType, error) {
	query := `
		SELECT id, company_id, name, is_paid, allow_carryover, max_carryover_days, created_at
		FROM leave_types
		WHERE company_id = $1
		ORDER BY is_paid DESC, name ASC
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query leave types: %w", err)
	}
	defer rows.Close()

	types := make([]domain.LeaveType, 0)
	for rows.Next() {
		var lt domain.LeaveType
		if err := rows.Scan(
			&lt.ID,
			&lt.CompanyID,
			&lt.Name,
			&lt.IsPaid,
			&lt.AllowCarryover,
			&lt.MaxCarryoverDays,
			&lt.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan leave type: %w", err)
		}
		types = append(types, lt)
	}
	return types, nil
}

func (r *LeaveRepository) CreateLeaveType(ctx context.Context, lt *domain.LeaveType) error {
	query := `
		INSERT INTO leave_types (company_id, name, is_paid, allow_carryover, max_carryover_days)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return r.pool.QueryRow(ctx, query,
		lt.CompanyID,
		lt.Name,
		lt.IsPaid,
		lt.AllowCarryover,
		lt.MaxCarryoverDays,
	).Scan(&lt.ID, &lt.CreatedAt)
}

func (r *LeaveRepository) UpdateLeaveType(ctx context.Context, lt *domain.LeaveType) error {
	query := `
		UPDATE leave_types
		SET name = $1, is_paid = $2, allow_carryover = $3, max_carryover_days = $4
		WHERE id = $5 AND company_id = $6
	`
	cmd, err := r.pool.Exec(ctx, query,
		lt.Name,
		lt.IsPaid,
		lt.AllowCarryover,
		lt.MaxCarryoverDays,
		lt.ID,
		lt.CompanyID,
	)
	if err != nil {
		return fmt.Errorf("failed to update leave type: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetUserLeaveBalances implements the cascading quota fallback algorithm with dynamic cycle dates and rollover.
func (r *LeaveRepository) GetUserLeaveBalances(
	ctx context.Context,
	userID string,
	companyID string,
	now time.Time,
) ([]domain.LeaveBalance, *domain.LeavePeriod, error) {
	policy, err := r.GetCompanyLeavePolicy(ctx, companyID)
	if err != nil {
		return nil, nil, err
	}

	period := CalculateLeavePeriod(policy.LeaveResetMonth, policy.LeaveResetDay, now)

	// 1. Find user's company role if assigned
	var roleID *string
	roleQuery := `SELECT role_id FROM user_roles WHERE user_id = $1 LIMIT 1`
	var tempRole string
	if err := r.pool.QueryRow(ctx, roleQuery, userID).Scan(&tempRole); err == nil {
		roleID = &tempRole
	}

	// 2. Fetch all leave types
	leaveTypes, err := r.GetLeaveTypesByCompany(ctx, companyID)
	if err != nil {
		return nil, nil, err
	}

	balances := make([]domain.LeaveBalance, 0, len(leaveTypes))

	for _, lt := range leaveTypes {
		// A. Cascading resolution for Base Allocated Days
		baseDays := 0

		// Step 1: Check user_leave_overrides
		var overrideDays int
		var overrideActive bool
		overrideQuery := `
			SELECT max_days, is_active 
			FROM user_leave_overrides 
			WHERE user_id = $1 AND leave_type_id = $2
		`
		if err := r.pool.QueryRow(ctx, overrideQuery, userID, lt.ID).Scan(&overrideDays, &overrideActive); err == nil {
			if overrideActive {
				baseDays = overrideDays
			}
		}

		// Step 2: Fall back to role_leave_configs if no active override
		if baseDays == 0 && roleID != nil {
			var roleDays int
			roleConfigQuery := `
				SELECT max_days 
				FROM role_leave_configs 
				WHERE role_id = $1 AND leave_type_id = $2
			`
			if err := r.pool.QueryRow(ctx, roleConfigQuery, *roleID, lt.ID).Scan(&roleDays); err == nil {
				baseDays = roleDays
			}
		}

		// B. Rollover / Carryover calculation
		carriedOverDays := 0
		canCarryover := policy.AllowLeaveCarryover || lt.AllowCarryover

		if canCarryover && baseDays > 0 {
			// Count approved days in previous period
			var prevUsedDays int
			prevUsedQuery := `
				SELECT COALESCE(SUM((end_date - start_date) + 1), 0)
				FROM leave_requests
				WHERE user_id = $1 AND leave_type_id = $2 AND status = $3
				  AND start_date >= $4 AND start_date <= $5
			`
			_ = r.pool.QueryRow(ctx, prevUsedQuery,
				userID,
				lt.ID,
				domain.LeaveStatusApproved,
				period.PrevStartDate.Format("2006-01-02"),
				period.PrevEndDate.Format("2006-01-02"),
			).Scan(&prevUsedDays)

			prevUnused := baseDays - prevUsedDays
			if prevUnused > 0 {
				carriedOverDays = prevUnused
				// Apply cap if defined
				if lt.MaxCarryoverDays > 0 && carriedOverDays > lt.MaxCarryoverDays {
					carriedOverDays = lt.MaxCarryoverDays
				} else if policy.MaxCarryoverDays > 0 && carriedOverDays > policy.MaxCarryoverDays {
					carriedOverDays = policy.MaxCarryoverDays
				}
			}
		}

		totalAllocated := baseDays + carriedOverDays

		// C. Current period used days
		var usedDays int
		usedQuery := `
			SELECT COALESCE(SUM((end_date - start_date) + 1), 0)
			FROM leave_requests
			WHERE user_id = $1 AND leave_type_id = $2 AND status = $3
			  AND start_date >= $4 AND start_date <= $5
		`
		_ = r.pool.QueryRow(ctx, usedQuery,
			userID,
			lt.ID,
			domain.LeaveStatusApproved,
			period.StartDate.Format("2006-01-02"),
			period.EndDate.Format("2006-01-02"),
		).Scan(&usedDays)

		// D. Current period pending days
		var pendingDays int
		pendingQuery := `
			SELECT COALESCE(SUM((end_date - start_date) + 1), 0)
			FROM leave_requests
			WHERE user_id = $1 AND leave_type_id = $2 AND status = $3
			  AND start_date >= $4 AND start_date <= $5
		`
		_ = r.pool.QueryRow(ctx, pendingQuery,
			userID,
			lt.ID,
			domain.LeaveStatusPending,
			period.StartDate.Format("2006-01-02"),
			period.EndDate.Format("2006-01-02"),
		).Scan(&pendingDays)

		remaining := totalAllocated - usedDays
		if remaining < 0 {
			remaining = 0
		}

		isFlexible := !lt.IsPaid && baseDays == 0

		balances = append(balances, domain.LeaveBalance{
			LeaveTypeID:        lt.ID,
			LeaveTypeName:      lt.Name,
			IsPaid:             lt.IsPaid,
			BaseAllocatedDays:  baseDays,
			CarriedOverDays:    carriedOverDays,
			TotalAllocatedDays: totalAllocated,
			UsedDays:           usedDays,
			PendingDays:        pendingDays,
			RemainingDays:      remaining,
			IsFlexible:         isFlexible,
		})
	}

	return balances, &period, nil
}

// CheckOverlap ensures an employee does not have conflicting pending or approved leave requests.
func (r *LeaveRepository) CheckOverlap(
	ctx context.Context,
	userID string,
	startDate, endDate time.Time,
	excludeID *string,
) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM leave_requests
		WHERE user_id = $1
		  AND status IN ($2, $3)
		  AND start_date <= $4
		  AND end_date >= $5
	`
	args := []any{
		userID,
		domain.LeaveStatusPending,
		domain.LeaveStatusApproved,
		endDate.Format("2006-01-02"),
		startDate.Format("2006-01-02"),
	}

	if excludeID != nil && *excludeID != "" {
		query += " AND id != $6"
		args = append(args, *excludeID)
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to check leave overlap: %w", err)
	}
	return count > 0, nil
}

func (r *LeaveRepository) CreateLeaveRequest(ctx context.Context, req *domain.LeaveRequest) error {
	query := `
		INSERT INTO leave_requests (company_id, user_id, leave_type_id, start_date, end_date, status, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	return r.pool.QueryRow(ctx, query,
		req.CompanyID,
		req.UserID,
		req.LeaveTypeID,
		req.StartDate.Format("2006-01-02"),
		req.EndDate.Format("2006-01-02"),
		req.Status,
		req.Reason,
	).Scan(&req.ID, &req.CreatedAt)
}

func (r *LeaveRepository) GetLeaveRequestsByUser(
	ctx context.Context,
	userID, companyID string,
	limit, offset int,
) ([]domain.LeaveRequestWithDetails, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM leave_requests
		WHERE user_id = $1 AND company_id = $2
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID, companyID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count user leave requests: %w", err)
	}

	query := `
		SELECT lr.id, lr.company_id, lr.user_id, lr.leave_type_id, lr.start_date, lr.end_date,
		       lr.status, lr.reason, lr.reviewed_by_user_id, lr.reviewed_at, lr.review_notes, lr.created_at,
		       u.first_name, u.last_name, u.email,
		       COALESCE(cr.name, u.system_role) AS role_name,
		       lt.name AS leave_type_name, lt.is_paid,
		       ((lr.end_date - lr.start_date) + 1) AS calendar_days
		FROM leave_requests lr
		JOIN users u ON u.id = lr.user_id
		JOIN leave_types lt ON lt.id = lr.leave_type_id
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN company_roles cr ON cr.id = ur.role_id
		WHERE lr.user_id = $1 AND lr.company_id = $2
		ORDER BY lr.created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, userID, companyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user leave requests: %w", err)
	}
	defer rows.Close()

	list := make([]domain.LeaveRequestWithDetails, 0)
	for rows.Next() {
		var item domain.LeaveRequestWithDetails
		if err := rows.Scan(
			&item.ID,
			&item.CompanyID,
			&item.UserID,
			&item.LeaveTypeID,
			&item.StartDate,
			&item.EndDate,
			&item.Status,
			&item.Reason,
			&item.ReviewedByUserID,
			&item.ReviewedAt,
			&item.ReviewNotes,
			&item.CreatedAt,
			&item.UserFirstName,
			&item.UserLastName,
			&item.UserEmail,
			&item.RoleName,
			&item.LeaveTypeName,
			&item.IsPaid,
			&item.CalendarDays,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user leave request: %w", err)
		}
		list = append(list, item)
	}

	return list, total, nil
}

func (r *LeaveRepository) GetLeaveRequestsByCompany(
	ctx context.Context,
	companyID string,
	statusFilter string,
	limit, offset int,
) ([]domain.LeaveRequestWithDetails, int, error) {
	whereClause := "WHERE lr.company_id = $1"
	args := []any{companyID}
	argIdx := 2

	if statusFilter != "" && statusFilter != "all" {
		whereClause += fmt.Sprintf(" AND lr.status = $%d", argIdx)
		args = append(args, statusFilter)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM leave_requests lr %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count company leave requests: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT lr.id, lr.company_id, lr.user_id, lr.leave_type_id, lr.start_date, lr.end_date,
		       lr.status, lr.reason, lr.reviewed_by_user_id, lr.reviewed_at, lr.review_notes, lr.created_at,
		       u.first_name, u.last_name, u.email,
		       COALESCE(cr.name, u.system_role) AS role_name,
		       lt.name AS leave_type_name, lt.is_paid,
		       ((lr.end_date - lr.start_date) + 1) AS calendar_days
		FROM leave_requests lr
		JOIN users u ON u.id = lr.user_id
		JOIN leave_types lt ON lt.id = lr.leave_type_id
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN company_roles cr ON cr.id = ur.role_id
		%s
		ORDER BY CASE WHEN lr.status = 'pending' THEN 0 ELSE 1 END, lr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query company leave requests: %w", err)
	}
	defer rows.Close()

	list := make([]domain.LeaveRequestWithDetails, 0)
	for rows.Next() {
		var item domain.LeaveRequestWithDetails
		if err := rows.Scan(
			&item.ID,
			&item.CompanyID,
			&item.UserID,
			&item.LeaveTypeID,
			&item.StartDate,
			&item.EndDate,
			&item.Status,
			&item.Reason,
			&item.ReviewedByUserID,
			&item.ReviewedAt,
			&item.ReviewNotes,
			&item.CreatedAt,
			&item.UserFirstName,
			&item.UserLastName,
			&item.UserEmail,
			&item.RoleName,
			&item.LeaveTypeName,
			&item.IsPaid,
			&item.CalendarDays,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan company leave request: %w", err)
		}
		list = append(list, item)
	}

	return list, total, nil
}

func (r *LeaveRepository) GetLeaveRequestByID(ctx context.Context, id, companyID string) (*domain.LeaveRequestWithDetails, error) {
	query := `
		SELECT lr.id, lr.company_id, lr.user_id, lr.leave_type_id, lr.start_date, lr.end_date,
		       lr.status, lr.reason, lr.reviewed_by_user_id, lr.reviewed_at, lr.review_notes, lr.created_at,
		       u.first_name, u.last_name, u.email,
		       COALESCE(cr.name, u.system_role) AS role_name,
		       lt.name AS leave_type_name, lt.is_paid,
		       ((lr.end_date - lr.start_date) + 1) AS calendar_days
		FROM leave_requests lr
		JOIN users u ON u.id = lr.user_id
		JOIN leave_types lt ON lt.id = lr.leave_type_id
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN company_roles cr ON cr.id = ur.role_id
		WHERE lr.id = $1 AND lr.company_id = $2
	`
	var item domain.LeaveRequestWithDetails
	err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&item.ID,
		&item.CompanyID,
		&item.UserID,
		&item.LeaveTypeID,
		&item.StartDate,
		&item.EndDate,
		&item.Status,
		&item.Reason,
		&item.ReviewedByUserID,
		&item.ReviewedAt,
		&item.ReviewNotes,
		&item.CreatedAt,
		&item.UserFirstName,
		&item.UserLastName,
		&item.UserEmail,
		&item.RoleName,
		&item.LeaveTypeName,
		&item.IsPaid,
		&item.CalendarDays,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get leave request by ID: %w", err)
	}
	return &item, nil
}

func (r *LeaveRepository) ResolveLeaveRequest(
	ctx context.Context,
	id, companyID, reviewerID, status string,
	reviewNotes *string,
) (*domain.LeaveRequestWithDetails, error) {
	now := time.Now().UTC()
	query := `
		UPDATE leave_requests
		SET status = $1,
		    reviewed_by_user_id = $2,
		    reviewed_at = $3,
		    review_notes = $4
		WHERE id = $5 AND company_id = $6
	`
	cmd, err := r.pool.Exec(ctx, query, status, reviewerID, now, reviewNotes, id, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve leave request: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return r.GetLeaveRequestByID(ctx, id, companyID)
}

// GetCompanyApprovedLeavesForPayroll returns all approved leave requests overlapping [start, end]
// grouped by user ID.
func (r *LeaveRepository) GetCompanyApprovedLeavesForPayroll(ctx context.Context, companyID string, start, end time.Time) (map[string][]domain.LeaveRequestWithDetails, error) {
	query := `
		SELECT lr.id, lr.company_id, lr.user_id, lr.leave_type_id, lr.start_date, lr.end_date,
		       lr.status, lr.reason, lr.reviewed_by_user_id, lr.reviewed_at, lr.review_notes, lr.created_at,
		       u.first_name, u.last_name, u.email,
		       COALESCE(cr.name, u.system_role) AS role_name,
		       lt.name AS leave_type_name, lt.is_paid,
		       ((lr.end_date - lr.start_date) + 1) AS calendar_days
		FROM leave_requests lr
		JOIN users u ON u.id = lr.user_id
		JOIN leave_types lt ON lt.id = lr.leave_type_id
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN company_roles cr ON cr.id = ur.role_id
		WHERE lr.company_id = $1
		  AND lr.status = 'approved'
		  AND lr.start_date <= $3
		  AND lr.end_date >= $2
		ORDER BY lr.user_id, lr.start_date ASC
	`
	rows, err := r.pool.Query(ctx, query, companyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query company leaves for payroll error: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]domain.LeaveRequestWithDetails)
	for rows.Next() {
		var item domain.LeaveRequestWithDetails
		if err := rows.Scan(
			&item.ID,
			&item.CompanyID,
			&item.UserID,
			&item.LeaveTypeID,
			&item.StartDate,
			&item.EndDate,
			&item.Status,
			&item.Reason,
			&item.ReviewedByUserID,
			&item.ReviewedAt,
			&item.ReviewNotes,
			&item.CreatedAt,
			&item.UserFirstName,
			&item.UserLastName,
			&item.UserEmail,
			&item.RoleName,
			&item.LeaveTypeName,
			&item.IsPaid,
			&item.CalendarDays,
		); err != nil {
			return nil, fmt.Errorf("scan leave for payroll error: %w", err)
		}
		result[item.UserID] = append(result[item.UserID], item)
	}

	return result, nil
}
