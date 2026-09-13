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
	ErrNotFound = errors.New("record not found")
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, company_id, first_name, last_name, email, password_hash, mobile_number, system_role, token_version, created_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.CompanyID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.PasswordHash,
		&u.MobileNumber,
		&u.SystemRole,
		&u.TokenVersion,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query user by email error: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, company_id, first_name, last_name, email, password_hash, mobile_number, system_role, token_version, created_at
		FROM users
		WHERE id = $1
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.CompanyID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.PasswordHash,
		&u.MobileNumber,
		&u.SystemRole,
		&u.TokenVersion,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query user by id error: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) IncrementTokenVersion(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET token_version = token_version + 1
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("increment token_version error: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) CreateUser(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (company_id, first_name, last_name, email, password_hash, mobile_number, system_role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, token_version, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		u.CompanyID,
		u.FirstName,
		u.LastName,
		u.Email,
		u.PasswordHash,
		u.MobileNumber,
		u.SystemRole,
	).Scan(&u.ID, &u.TokenVersion, &u.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert user error: %w", err)
	}
	return nil
}

func (r *UserRepository) GetCompanyByName(ctx context.Context, name string) (*domain.Company, error) {
	query := `
		SELECT id, name, currency_code, timezone, created_at
		FROM companies
		WHERE name = $1
		LIMIT 1
	`
	var c domain.Company
	err := r.pool.QueryRow(ctx, query, name).Scan(
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
		return nil, fmt.Errorf("query company by name error: %w", err)
	}
	return &c, nil
}

func (r *UserRepository) CreateCompany(ctx context.Context, c *domain.Company) error {
	query := `
		INSERT INTO companies (name, currency_code, timezone)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		c.Name,
		c.CurrencyCode,
		c.Timezone,
	).Scan(&c.ID, &c.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert company error: %w", err)
	}
	return nil
}

// GetUsersByCompanyID returns all non-super-admin users in a tenant company.
func (r *UserRepository) GetUsersByCompanyID(ctx context.Context, companyID string) ([]domain.User, error) {
	query := `
		SELECT id, company_id, first_name, last_name, email, password_hash, mobile_number, system_role, token_version, created_at
		FROM users
		WHERE company_id = $1 AND system_role != 'super_admin'
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("query users by company error: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.CompanyID,
			&u.FirstName,
			&u.LastName,
			&u.Email,
			&u.PasswordHash,
			&u.MobileNumber,
			&u.SystemRole,
			&u.TokenVersion,
			&u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user error: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// CreateCompanyUser inserts a new HR or Employee user. The caller MUST set u.CompanyID
// from the authenticated JWT claims — never from the request body.
func (r *UserRepository) CreateCompanyUser(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (company_id, first_name, last_name, email, password_hash, mobile_number, system_role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, token_version, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		u.CompanyID,
		u.FirstName,
		u.LastName,
		u.Email,
		u.PasswordHash,
		u.MobileNumber,
		u.SystemRole,
	).Scan(&u.ID, &u.TokenVersion, &u.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert company user error: %w", err)
	}
	return nil
}

