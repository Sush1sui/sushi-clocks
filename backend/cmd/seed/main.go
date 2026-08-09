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
# seed super admin
go run ./cmd/seed
*/

const PlatformCompanyName = "__platform__"

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

	// 1. Ensure platform master company exists
	platformCompany, err := userRepo.GetCompanyByName(ctx, PlatformCompanyName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.Printf("creating platform master company (%s)...", PlatformCompanyName)
			platformCompany = &domain.Company{
				Name:         PlatformCompanyName,
				CurrencyCode: "USD",
				Timezone:     "UTC",
			}
			if err := userRepo.CreateCompany(ctx, platformCompany); err != nil {
				log.Fatalf("failed to create platform company: %v", err)
			}
			log.Printf("platform company created with ID: %s", platformCompany.ID)
		} else {
			log.Fatalf("failed to query platform company: %v", err)
		}
	} else {
		log.Printf("platform company already exists: %s (ID: %s)", platformCompany.Name, platformCompany.ID)
	}

	// 2. Prepare Super Admin info
	adminEmail := cfg.SuperAdminEmail
	if adminEmail == "" {
		adminEmail = "superadmin@sushi-clocks.dev"
	}

	adminPassword := cfg.SuperAdminPassword
	if adminPassword == "" {
		adminPassword = "SuperAdminPassword123!"
	}

	adminFirstName := cfg.SuperAdminFirstName
	if adminFirstName == "" {
		adminFirstName = "Platform"
	}

	adminLastName := cfg.SuperAdminLastName
	if adminLastName == "" {
		adminLastName = "SuperAdmin"
	}

	var mobilePtr *string
	if cfg.SuperAdminMobile != "" {
		mobilePtr = &cfg.SuperAdminMobile
	}

	// 3. Ensure Super Admin user exists
	existingAdmin, err := userRepo.GetUserByEmail(ctx, adminEmail)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.Printf("seeding super admin account: %s...", adminEmail)

			passwordHash, err := auth.HashPassword(adminPassword)
			if err != nil {
				log.Fatalf("failed to hash super admin password: %v", err)
			}

			newAdmin := &domain.User{
				CompanyID:    platformCompany.ID,
				FirstName:    adminFirstName,
				LastName:     adminLastName,
				Email:        adminEmail,
				PasswordHash: passwordHash,
				MobileNumber: mobilePtr,
				SystemRole:   domain.RoleSuperAdmin,
			}

			if err := userRepo.CreateUser(ctx, newAdmin); err != nil {
				log.Fatalf("failed to create super admin: %v", err)
			}

			log.Printf("super admin created successfully!")
			log.Printf("Email: %s", newAdmin.Email)
			log.Printf("Role:  %s", newAdmin.SystemRole)
			log.Printf("ID:    %s", newAdmin.ID)
		} else {
			log.Fatalf("failed to query super admin: %v", err)
		}
	} else {
		log.Printf("super admin account already exists (Email: %s, ID: %s, Role: %s)", existingAdmin.Email, existingAdmin.ID, existingAdmin.SystemRole)
	}

	log.Println("database seeding completed successfully")
}
