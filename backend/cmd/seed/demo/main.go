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

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	companyRepo := repository.NewCompanyRepository(pool)

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

	// ── Summary ──────────────────────────────────────────────────────────────
	log.Println("\n── Seed Summary ──────────────────────────────────────────")
	log.Printf("Company  : %s (PHP / Asia/Manila)", demoCompany.Name)
	log.Println("Accounts :")
	log.Println("  admin@sushidemo.com   Admin1234!   → Company Admin")
	log.Println("  hr@sushidemo.com      Hr1234567!   → HR Manager")
	log.Println("  nagito@sushidemo.com    Employee123! → Employee")
	log.Println("  nagi@sushidemo.com    Employee123! → Employee")
	log.Println("  reo@sushidemo.com    Employee123! → Employee")
	log.Println("──────────────────────────────────────────────────────────")

	// Suppress unused import if companyRepo is not used elsewhere
	_ = companyRepo
}
