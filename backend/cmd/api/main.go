package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sushi-clocks/backend/internal/api"
	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/config"
	"github.com/sushi-clocks/backend/internal/db"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
)

type HealthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database"`
}

func corsMiddleware(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthHandler(dbConnected bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		dbStatus := "disconnected"
		if dbConnected {
			dbStatus = "connected"
		}

		resp := HealthResponse{
			Status:   "ok",
			Service:  "sushi-clocks-api",
			Database: dbStatus,
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("error encoding response: %v", err)
		}
	}
}

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()

	var dbConnected bool
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	if cfg.DatabaseURL != "" {
		pool, err := db.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Printf("warning: database connection failed: %v", err)
		} else {
			defer pool.Close()
			dbConnected = true
			log.Println("connected to database successfully")

			userRepo := repository.NewUserRepository(pool)
			companyRepo := repository.NewCompanyRepository(pool)
			timesheetRepo := repository.NewTimesheetRepository(pool)

			authHandler := api.NewAuthHandler(cfg, userRepo, jwtMgr)
			companyHandler := api.NewCompanyHandler(companyRepo)
			timesheetHandler := api.NewTimesheetHandler(timesheetRepo)

			rateLimiter := api.NewIPRateLimiter(5.0, 15.0) // 5 req/sec with burst 15

			// Auth routes with rate limiting
			mux.HandleFunc("POST /api/v1/auth/login", rateLimiter.Middleware(authHandler.Login))
			mux.HandleFunc("POST /api/v1/auth/refresh", rateLimiter.Middleware(authHandler.Refresh))
			mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)

			// Protected routes
			authMiddleware := auth.RequireAuth(jwtMgr)
			superAdminMiddleware := auth.RequireSuperAdmin(jwtMgr)
			adminHrMiddleware := auth.RequireRoles(jwtMgr, domain.RoleAdmin, domain.RoleHR)

			mux.Handle("GET /api/v1/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

			// Super Admin Company Management routes
			mux.Handle("GET /api/v1/companies", superAdminMiddleware(http.HandlerFunc(companyHandler.GetCompanies)))
			mux.Handle("POST /api/v1/companies", superAdminMiddleware(http.HandlerFunc(companyHandler.CreateCompany)))

			// Tenant-scoped Company details route
			mux.Handle("GET /api/v1/companies/{id}", authMiddleware(http.HandlerFunc(companyHandler.GetCompanyByID)))

			// Timesheet & Attendance routes
			mux.Handle("POST /api/v1/timesheets/clock-in", authMiddleware(http.HandlerFunc(timesheetHandler.ClockIn)))
			mux.Handle("POST /api/v1/timesheets/clock-out", authMiddleware(http.HandlerFunc(timesheetHandler.ClockOut)))
			mux.Handle("GET /api/v1/timesheets/status", authMiddleware(http.HandlerFunc(timesheetHandler.GetStatus)))
			mux.Handle("GET /api/v1/companies/{id}/attendance/summary", adminHrMiddleware(http.HandlerFunc(timesheetHandler.GetCompanySummary)))
		}
	} else {
		log.Println("DATABASE_URL not set, database features disabled")
	}

	mux.HandleFunc("GET /", healthHandler(dbConnected))

	// Wrap entire mux with CORS
	handler := corsMiddleware(cfg.CORSAllowedOrigins, mux)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("sushi-clocks-api listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited cleanly")
}
