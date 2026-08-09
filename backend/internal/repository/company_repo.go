package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sushi-clocks/backend/internal/domain"
)

type CompanyRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

func (r *CompanyRepository) GetAllCompaniesWithStats(ctx context.Context) ([]domain.CompanyWithStats, error) {
	query := `
		SELECT 
			c.id, 
			c.name, 
			c.currency_code, 
			c.timezone, 
			c.created_at,
			COUNT(u.id) as total_users,
			(
				SELECT u2.email 
				FROM users u2 
				WHERE u2.company_id = c.id AND u2.system_role = 'admin' 
				ORDER BY u2.created_at ASC 
				LIMIT 1
			) as admin_email
		FROM companies c
		LEFT JOIN users u ON u.company_id = c.id
		WHERE c.name != '__platform__'
		GROUP BY c.id, c.name, c.currency_code, c.timezone, c.created_at
		ORDER BY c.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query companies error: %w", err)
	}
	defer rows.Close()

	companies := make([]domain.CompanyWithStats, 0)
	for rows.Next() {
		var cs domain.CompanyWithStats
		if err := rows.Scan(
			&cs.ID,
			&cs.Name,
			&cs.CurrencyCode,
			&cs.Timezone,
			&cs.CreatedAt,
			&cs.TotalUsers,
			&cs.AdminEmail,
		); err != nil {
			return nil, fmt.Errorf("scan company stats error: %w", err)
		}
		companies = append(companies, cs)
	}

	return companies, nil
}

func (r *CompanyRepository) GetCompanyByID(ctx context.Context, id string) (*domain.Company, error) {
	query := `
		SELECT id, name, currency_code, timezone, created_at
		FROM companies
		WHERE id = $1
		LIMIT 1
	`
	var c domain.Company
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.CurrencyCode,
		&c.Timezone,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query company by id error: %w", err)
	}
	return &c, nil
}

func (r *CompanyRepository) CreateCompanyWithAdmin(ctx context.Context, company *domain.Company, admin *domain.User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction error: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Insert Company
	companyQuery := `
		INSERT INTO companies (name, currency_code, timezone)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err = tx.QueryRow(ctx, companyQuery,
		company.Name,
		company.CurrencyCode,
		company.Timezone,
	).Scan(&company.ID, &company.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert company error: %w", err)
	}

	// 2. Insert Admin User if specified
	if admin != nil {
		admin.CompanyID = company.ID
		admin.SystemRole = domain.RoleAdmin

		userQuery := `
			INSERT INTO users (company_id, first_name, last_name, email, password_hash, mobile_number, system_role)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, created_at
		`
		err = tx.QueryRow(ctx, userQuery,
			admin.CompanyID,
			admin.FirstName,
			admin.LastName,
			admin.Email,
			admin.PasswordHash,
			admin.MobileNumber,
			admin.SystemRole,
		).Scan(&admin.ID, &admin.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert initial company admin error: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit company creation transaction error: %w", err)
	}

	return nil
}
