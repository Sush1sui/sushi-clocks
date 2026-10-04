package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sushi-clocks/backend/internal/domain"
)

type WageRepository struct {
	pool *pgxpool.Pool
}

func NewWageRepository(pool *pgxpool.Pool) *WageRepository {
	return &WageRepository{pool: pool}
}

// ResolveAllUserWages returns the effective wage for all non-super-admin users in the company.
// Active overrides take precedence over company role default wages.
func (r *WageRepository) ResolveAllUserWages(ctx context.Context, companyID string) (map[string]*domain.ResolvedWage, error) {
	query := `
		WITH combined AS (
			SELECT u.id AS user_id, uwo.wage_type, uwo.override_wage AS rate, 'override' AS source, 1 AS priority
			FROM users u
			JOIN user_wage_overrides uwo ON uwo.user_id = u.id AND uwo.is_active = true
			WHERE u.company_id = $1 AND u.system_role != 'super_admin'
			UNION ALL
			SELECT u.id AS user_id, cr.wage_type, cr.default_wage AS rate, 'role' AS source, 2 AS priority
			FROM users u
			JOIN user_roles ur ON ur.user_id = u.id
			JOIN company_roles cr ON cr.id = ur.role_id AND cr.company_id = $1
			WHERE u.company_id = $1 AND u.system_role != 'super_admin'
		)
		SELECT DISTINCT ON (user_id) user_id, wage_type, rate, source
		FROM combined
		ORDER BY user_id, priority ASC
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("resolve user wages error: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*domain.ResolvedWage)
	for rows.Next() {
		var w domain.ResolvedWage
		if err := rows.Scan(&w.UserID, &w.WageType, &w.Rate, &w.Source); err != nil {
			return nil, fmt.Errorf("scan resolved wage error: %w", err)
		}
		result[w.UserID] = &w
	}
	return result, nil
}

// GetUserModifiers returns all payroll modifiers assigned to a specific user within the company.
func (r *WageRepository) GetUserModifiers(ctx context.Context, userID, companyID string) ([]domain.PayrollModifier, error) {
	query := `
		SELECT pm.id, pm.name, pm.modifier_type, pm.calculation_method, pm.value
		FROM user_payroll_modifiers upm
		JOIN payroll_modifiers pm ON pm.id = upm.modifier_id
		WHERE upm.user_id = $1 AND pm.company_id = $2
		ORDER BY pm.modifier_type ASC, pm.name ASC
	`
	rows, err := r.pool.Query(ctx, query, userID, companyID)
	if err != nil {
		return nil, fmt.Errorf("get user modifiers error: %w", err)
	}
	defer rows.Close()

	var modifiers []domain.PayrollModifier
	for rows.Next() {
		var m domain.PayrollModifier
		if err := rows.Scan(&m.ID, &m.Name, &m.ModifierType, &m.CalculationMethod, &m.Value); err != nil {
			return nil, fmt.Errorf("scan user modifier error: %w", err)
		}
		modifiers = append(modifiers, m)
	}
	return modifiers, nil
}

// GetRoleNameForUser returns the role name for a given user in the company, or empty string if unassigned.
func (r *WageRepository) GetRoleNameForUser(ctx context.Context, userID, companyID string) (string, error) {
	query := `
		SELECT cr.name
		FROM user_roles ur
		JOIN company_roles cr ON cr.id = ur.role_id
		WHERE ur.user_id = $1 AND cr.company_id = $2
		LIMIT 1
	`
	var roleName string
	err := r.pool.QueryRow(ctx, query, userID, companyID).Scan(&roleName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get role name error: %w", err)
	}
	return roleName, nil
}

// GetAllCompanyUserModifiers batch fetches all user-assigned modifiers in the company (keyed by user_id).
func (r *WageRepository) GetAllCompanyUserModifiers(ctx context.Context, companyID string) (map[string][]domain.PayrollModifier, error) {
	query := `
		SELECT upm.user_id, pm.id, pm.name, pm.modifier_type, pm.calculation_method, pm.value
		FROM user_payroll_modifiers upm
		JOIN payroll_modifiers pm ON pm.id = upm.modifier_id
		WHERE pm.company_id = $1
		ORDER BY pm.modifier_type ASC, pm.name ASC
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("batch get company user modifiers error: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]domain.PayrollModifier)
	for rows.Next() {
		var userID string
		var m domain.PayrollModifier
		if err := rows.Scan(&userID, &m.ID, &m.Name, &m.ModifierType, &m.CalculationMethod, &m.Value); err != nil {
			return nil, fmt.Errorf("scan company user modifier error: %w", err)
		}
		result[userID] = append(result[userID], m)
	}
	return result, nil
}

// GetAllCompanyUserRoleNames batch fetches role names for all users in the company (keyed by user_id).
func (r *WageRepository) GetAllCompanyUserRoleNames(ctx context.Context, companyID string) (map[string]string, error) {
	query := `
		SELECT ur.user_id, cr.name
		FROM user_roles ur
		JOIN company_roles cr ON cr.id = ur.role_id
		WHERE cr.company_id = $1
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("batch get company role names error: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var userID, roleName string
		if err := rows.Scan(&userID, &roleName); err != nil {
			return nil, fmt.Errorf("scan company user role name error: %w", err)
		}
		result[userID] = roleName
	}
	return result, nil
}

