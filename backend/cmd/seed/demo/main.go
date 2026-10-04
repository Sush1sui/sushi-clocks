package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/config"
	"github.com/sushi-clocks/backend/internal/db"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
)

/*
How to run the seeders:

  # 1. Super Admin only (run once on first setup)
  go run ./cmd/seed

  # 2. Demo company + Company Admin + HR + 3 Employees
  go run ./cmd/seed/demo

Run from: d:\VSC FILES\sushi-clocks\backend
*/

func main() {
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required to run seeder")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	companyRepo := repository.NewCompanyRepository(pool)
	_ = companyRepo

	// ── 1. Create Demo Company ───────────────────────────────────────────────
	const demoCompanyName = "Sushi Demo Corp"

	demoCompany, err := userRepo.GetCompanyByName(ctx, demoCompanyName)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			log.Fatalf("failed to query demo company: %v", err)
		}

		log.Printf("creating demo company: %s...", demoCompanyName)
		demoCompany = &domain.Company{
			Name:         demoCompanyName,
			CurrencyCode: "PHP",
			Timezone:     "Asia/Manila",
		}
		if err := userRepo.CreateCompany(ctx, demoCompany); err != nil {
			log.Fatalf("failed to create demo company: %v", err)
		}
		log.Printf("demo company created — ID: %s", demoCompany.ID)
	} else {
		log.Printf("demo company already exists — %s (ID: %s)", demoCompany.Name, demoCompany.ID)
	}

	// ── 2. Seed helpers ──────────────────────────────────────────────────────
	seedUser := func(firstName, lastName, email, password, role string) {
		existing, err := userRepo.GetUserByEmail(ctx, email)
		if err == nil {
			log.Printf("  [skip] %s already exists (ID: %s, Role: %s)", email, existing.ID, existing.SystemRole)
			return
		}
		if !errors.Is(err, repository.ErrNotFound) {
			log.Fatalf("  failed to query %s: %v", email, err)
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			log.Fatalf("  failed to hash password for %s: %v", email, err)
		}

		u := &domain.User{
			CompanyID:    demoCompany.ID,
			FirstName:    firstName,
			LastName:     lastName,
			Email:        email,
			PasswordHash: hash,
			SystemRole:   role,
		}
		if err := userRepo.CreateUser(ctx, u); err != nil {
			log.Fatalf("  failed to create %s: %v", email, err)
		}
		log.Printf("  [ok] %-32s role=%-10s id=%s", email, role, u.ID)
	}

	// ── 3. Seed Company Admin ────────────────────────────────────────────────
	log.Println("\n── Company Admin ──")
	seedUser(
		"Isagi", "Yoichi",
		"admin@sushidemo.com",
		"Admin1234!",
		domain.RoleAdmin,
	)

	// ── 4. Seed HR Manager ───────────────────────────────────────────────────
	log.Println("\n── HR Manager ──")
	seedUser(
		"Chigiri", "Hyoma",
		"hr@sushidemo.com",
		"Hr1234567!",
		domain.RoleHR,
	)

	// ── 5. Seed Employees ────────────────────────────────────────────────────
	log.Println("\n── Employees ──")
	seedUser("Nagito", "Igarashi", "nagito@sushidemo.com", "Employee123!", domain.RoleEmployee)
	seedUser("Eita", "Nagi", "nagi@sushidemo.com", "Employee123!", domain.RoleEmployee)
	seedUser("Reo", "Mikage", "reo@sushidemo.com", "Employee123!", domain.RoleEmployee)

	// ── 6. Update Company Leave Policy ───────────────────────────────────────
	log.Println("\n── Company Leave Policy ──")
	_, err = pool.Exec(ctx, `
		UPDATE companies
		SET leave_reset_month = 1,
		    leave_reset_day = 1,
		    allow_leave_carryover = TRUE,
		    max_carryover_days = 5
		WHERE id = $1
	`, demoCompany.ID)
	if err != nil {
		log.Printf("warning: failed to update company leave policy: %v", err)
	} else {
		log.Println("  [ok] Leave policy set: Reset Jan 1, Carryover Enabled (Max 5 days)")
	}

	// ── 7. Seed Company Roles ────────────────────────────────────────────────
	log.Println("\n── Company Roles ──")
	seedRole := func(name, wageType string, wage float64) string {
		var roleID string
		err := pool.QueryRow(ctx, `
			SELECT id FROM company_roles WHERE company_id = $1 AND name = $2
		`, demoCompany.ID, name).Scan(&roleID)
		if err == nil {
			log.Printf("  [skip] role %s already exists (ID: %s)", name, roleID)
			return roleID
		}
		err = pool.QueryRow(ctx, `
			INSERT INTO company_roles (company_id, name, default_wage, wage_type)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, demoCompany.ID, name, wage, wageType).Scan(&roleID)
		if err != nil {
			log.Fatalf("failed to create role %s: %v", name, err)
		}
		log.Printf("  [ok] role %-20s wage=%.2f/%s id=%s", name, wage, wageType, roleID)
		return roleID
	}

	devRoleID := seedRole("Software Engineer", "daily", 800.00)
	hrRoleID := seedRole("HR Specialist", "daily", 750.00)

	// ── 8. Assign User Roles ─────────────────────────────────────────────────
	log.Println("\n── Assign User Roles ──")
	assignRole := func(email, roleID string) {
		u, err := userRepo.GetUserByEmail(ctx, email)
		if err != nil {
			return
		}
		_, _ = pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			VALUES ($1, $2)
			ON CONFLICT (user_id) DO UPDATE SET role_id = $2
		`, u.ID, roleID)
		log.Printf("  [ok] assigned %s to role %s", email, roleID)
	}

	assignRole("admin@sushidemo.com", devRoleID) // Admin needs a company role for leave quota cascade
	assignRole("hr@sushidemo.com", hrRoleID)
	assignRole("nagito@sushidemo.com", devRoleID)
	assignRole("nagi@sushidemo.com", devRoleID)
	assignRole("reo@sushidemo.com", devRoleID)

	// ── 9. Seed Leave Types ──────────────────────────────────────────────────
	log.Println("\n── Leave Categories ──")
	seedLeaveType := func(name string, isPaid, allowCarryover bool, maxCarryover int) string {
		var ltID string
		err := pool.QueryRow(ctx, `
			SELECT id FROM leave_types WHERE company_id = $1 AND name = $2
		`, demoCompany.ID, name).Scan(&ltID)
		if err == nil {
			// Update carryover flags if existing
			_, _ = pool.Exec(ctx, `
				UPDATE leave_types
				SET allow_carryover = $1, max_carryover_days = $2
				WHERE id = $3
			`, allowCarryover, maxCarryover, ltID)
			log.Printf("  [skip] leave type %s already exists (ID: %s)", name, ltID)
			return ltID
		}
		err = pool.QueryRow(ctx, `
			INSERT INTO leave_types (company_id, name, is_paid, allow_carryover, max_carryover_days)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, demoCompany.ID, name, isPaid, allowCarryover, maxCarryover).Scan(&ltID)
		if err != nil {
			log.Fatalf("failed to create leave type %s: %v", name, err)
		}
		log.Printf("  [ok] leave type %-18s is_paid=%-5v carryover=%-5v (max %d) id=%s", name, isPaid, allowCarryover, maxCarryover, ltID)
		return ltID
	}

	vacationID := seedLeaveType("Vacation Leave", true, true, 5)
	sickID := seedLeaveType("Sick Leave", true, false, 0)
	emergencyID := seedLeaveType("Emergency Leave", true, false, 0)
	unpaidID := seedLeaveType("Unpaid Leave", false, false, 0)

	// ── 10. Seed Role Leave Quotas ───────────────────────────────────────────
	log.Println("\n── Role Leave Quotas ──")
	seedRoleQuota := func(rID, ltID string, days int) {
		_, _ = pool.Exec(ctx, `
			INSERT INTO role_leave_configs (role_id, leave_type_id, max_days)
			VALUES ($1, $2, $3)
			ON CONFLICT (role_id, leave_type_id) DO UPDATE SET max_days = $3
		`, rID, ltID, days)
	}

	seedRoleQuota(devRoleID, vacationID, 15)
	seedRoleQuota(devRoleID, sickID, 10)
	seedRoleQuota(devRoleID, emergencyID, 5)
	seedRoleQuota(devRoleID, unpaidID, 0) // Option C: 0 quota = flexible

	seedRoleQuota(hrRoleID, vacationID, 15)
	seedRoleQuota(hrRoleID, sickID, 10)
	seedRoleQuota(hrRoleID, emergencyID, 5)
	seedRoleQuota(hrRoleID, unpaidID, 0)
	log.Println("  [ok] Dev & HR quotas configured: Vacation: 15d, Sick: 10d, Emergency: 5d, Unpaid: 0d (flexible)")

	// ── 11. Seed User Leave Override for Eita Nagi (20 Days Vacation) ───────
	log.Println("\n── Individual Leave Override ──")
	nagiUser, err := userRepo.GetUserByEmail(ctx, "nagi@sushidemo.com")
	if err == nil {
		_, _ = pool.Exec(ctx, `
			INSERT INTO user_leave_overrides (user_id, leave_type_id, is_active, max_days)
			VALUES ($1, $2, TRUE, 20)
			ON CONFLICT (user_id, leave_type_id) DO UPDATE SET is_active = TRUE, max_days = 20
		`, nagiUser.ID, vacationID)
		log.Printf("  [ok] nagi@sushidemo.com: Vacation Leave override = 20 days (Active)")
	}

	// ── Summary ──────────────────────────────────────────────────────────────
	log.Println("\n── Seed Summary ──────────────────────────────────────────")
	log.Printf("Company  : %s (PHP / Asia/Manila)", demoCompany.Name)
	log.Println("Accounts :")
	log.Println("  admin@sushidemo.com   Admin1234!   → Company Admin (role: Software Engineer — Vacation 15d, Sick 10d, Emergency 5d)")
	log.Println("  hr@sushidemo.com      Hr1234567!   → HR Specialist (role: HR Specialist — Vacation 15d, Sick 10d, Emergency 5d)")
	log.Println("  nagito@sushidemo.com  Employee123! → Software Engineer (Standard 15d Vacation)")
	log.Println("  nagi@sushidemo.com    Employee123! → Software Engineer (Override 20d Vacation)")
	log.Println("  reo@sushidemo.com     Employee123! → Software Engineer (Standard 15d Vacation)")
	log.Println("──────────────────────────────────────────────────────────")
}
