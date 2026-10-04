package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sushi-clocks/backend/internal/api"
	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/config"
	"github.com/sushi-clocks/backend/internal/db"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
	"github.com/sushi-clocks/backend/internal/service"
	"github.com/sushi-clocks/backend/internal/sse"
)

type HealthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database"`
}

func securityHeadersMiddleware(isProduction bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OWASP A05: Security response headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if isProduction {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(allowedOriginsStr string, next http.Handler) http.Handler {
	rawOrigins := strings.Split(allowedOriginsStr, ",")
	allowedMap := make(map[string]bool)
	for _, o := range rawOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			allowedMap[trimmed] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedMap[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if len(allowedMap) == 0 {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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
		pool, err := db.NewPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
		if err != nil {
			log.Printf("warning: database connection failed: %v", err)
		} else {
			defer pool.Close()
			dbConnected = true
			log.Printf("connected to database successfully (pool max: %d, min: %d)", cfg.DBMaxConns, cfg.DBMinConns)

			userRepo := repository.NewUserRepository(pool)
			companyRepo := repository.NewCompanyRepository(pool)
			timesheetRepo := repository.NewTimesheetRepository(pool)
			leaveRepo := repository.NewLeaveRepository(pool)
			wageRepo := repository.NewWageRepository(pool)
			payrollSvc := service.NewPayrollService(timesheetRepo, leaveRepo, wageRepo, companyRepo, userRepo)
			emailSvc := service.NewEmailService(cfg)

			// SSE Real-Time Presence Hub
			sseHub := sse.NewHub()
			sseHandler := api.NewSSEHandler(sseHub)

			// MongoDB Atlas Telemetry & Audit Logs (Free Tier)
			var telemetryRepo *repository.TelemetryRepository
			var auditRepo *repository.AuditRepository
			if cfg.MongoURI != "" {
				mongoClient, err := db.NewMongoClient(ctx, cfg.MongoURI)
				if err != nil {
					log.Printf("warning: mongo connection failed: %v", err)
				} else {
					defer func() { _ = mongoClient.Disconnect(context.Background()) }()
					log.Println("connected to MongoDB Atlas successfully")
					mongoDB := mongoClient.Database(cfg.MongoDBName)
					telemetryRepo = repository.NewTelemetryRepository(mongoDB)
					auditRepo = repository.NewAuditRepository(mongoDB)

					// Ensure 366-day safety TTL indexes in MongoDB
					_ = telemetryRepo.EnsureIndexes(ctx)
					_ = auditRepo.EnsureIndexes(ctx)

					// Start automated 1-Year Archival & Email Dispatcher
					archiveSvc := service.NewArchiveService(telemetryRepo, auditRepo, userRepo, emailSvc)
					archiveSvc.StartScheduler(ctx, 24*time.Hour)
				}
			}

			authHandler := api.NewAuthHandler(cfg, userRepo, jwtMgr)
			companyHandler := api.NewCompanyHandler(companyRepo)
			timesheetHandler := api.NewTimesheetHandler(timesheetRepo, telemetryRepo, sseHub, cfg.BehindProxy)
			adjustmentHandler := api.NewAdjustmentHandler(timesheetRepo, auditRepo, sseHub)
			userHandler := api.NewUserHandler(userRepo)
			leaveHandler := api.NewLeaveHandler(leaveRepo, auditRepo, sseHub)
			payrollHandler := api.NewPayrollHandler(payrollSvc)

			rateLimiter := api.NewIPRateLimiter(5.0, 15.0, cfg.BehindProxy) // 5 req/sec with burst 15
			heavyQueue := api.NewHeavyQueue(2, 30*time.Second)              // Max 2 concurrent heavy queries

			// Auth routes with rate limiting
			mux.HandleFunc("POST /api/v1/auth/login", rateLimiter.Middleware(authHandler.Login))
			mux.HandleFunc("POST /api/v1/auth/refresh", rateLimiter.Middleware(authHandler.Refresh))
			mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)

			// Protected routes
			authMiddleware := auth.RequireAuth(jwtMgr)
			superAdminMiddleware := auth.RequireSuperAdmin(jwtMgr)
			adminHrMiddleware := auth.RequireRoles(jwtMgr, domain.RoleAdmin, domain.RoleHR)
			adminOnlyMiddleware := auth.RequireRoles(jwtMgr, domain.RoleAdmin)

			mux.Handle("GET /api/v1/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

			// Real-time SSE event stream
			mux.Handle("GET /api/v1/events", authMiddleware(http.HandlerFunc(sseHandler.Subscribe)))

			// Super Admin Company Management routes
			mux.Handle("GET /api/v1/companies", superAdminMiddleware(http.HandlerFunc(companyHandler.GetCompanies)))
			mux.Handle("POST /api/v1/companies", superAdminMiddleware(http.HandlerFunc(companyHandler.CreateCompany)))

			// Tenant-scoped Company details route
			mux.Handle("GET /api/v1/companies/{id}", authMiddleware(http.HandlerFunc(companyHandler.GetCompanyByID)))

			// Timesheet & Attendance routes
			mux.Handle("POST /api/v1/timesheets/clock-in", authMiddleware(http.HandlerFunc(timesheetHandler.ClockIn)))
			mux.Handle("POST /api/v1/timesheets/clock-out", authMiddleware(http.HandlerFunc(timesheetHandler.ClockOut)))
			mux.Handle("GET /api/v1/timesheets/status", authMiddleware(http.HandlerFunc(timesheetHandler.GetStatus)))
			mux.Handle("GET /api/v1/timesheets/history", authMiddleware(http.HandlerFunc(timesheetHandler.GetShiftHistory)))
			mux.Handle("GET /api/v1/companies/{id}/attendance/summary", adminHrMiddleware(http.HandlerFunc(timesheetHandler.GetCompanySummary)))
			mux.Handle("GET /api/v1/companies/{id}/attendance/roster", adminHrMiddleware(http.HandlerFunc(timesheetHandler.GetLiveRoster)))

			// Attendance Adjustments & Audit routes
			mux.Handle("POST /api/v1/timesheets/{id}/adjustment-request", authMiddleware(http.HandlerFunc(adjustmentHandler.RequestAdjustment)))
			mux.Handle("GET /api/v1/companies/{id}/adjustments", adminHrMiddleware(http.HandlerFunc(adjustmentHandler.GetCompanyAdjustments)))
			mux.Handle("PATCH /api/v1/timesheets/adjustments/{id}", adminHrMiddleware(http.HandlerFunc(adjustmentHandler.ResolveAdjustment)))
			mux.Handle("PUT /api/v1/timesheets/{id}", adminHrMiddleware(http.HandlerFunc(adjustmentHandler.DirectOverride)))

			// Tenant User Management routes
			mux.Handle("GET /api/v1/companies/{id}/users", adminHrMiddleware(http.HandlerFunc(userHandler.GetCompanyUsers)))
			mux.Handle("POST /api/v1/companies/{id}/users", adminOnlyMiddleware(http.HandlerFunc(userHandler.CreateCompanyUser)))
			mux.Handle("PATCH /api/v1/users/archive-preference", authMiddleware(http.HandlerFunc(userHandler.UpdateArchivePreference)))

			// Leave Management & Policy routes
			mux.Handle("GET /api/v1/leave/types", authMiddleware(http.HandlerFunc(leaveHandler.GetLeaveTypes)))
			mux.Handle("GET /api/v1/leave/balances", authMiddleware(http.HandlerFunc(leaveHandler.GetLeaveBalances)))
			mux.Handle("POST /api/v1/leave/requests", authMiddleware(http.HandlerFunc(leaveHandler.SubmitLeaveRequest)))
			mux.Handle("GET /api/v1/leave/requests", authMiddleware(http.HandlerFunc(leaveHandler.GetMyLeaveRequests)))
			mux.Handle("GET /api/v1/companies/{id}/leave-requests", adminHrMiddleware(http.HandlerFunc(leaveHandler.GetCompanyLeaveRequests)))
			mux.Handle("PATCH /api/v1/leave/requests/{id}", adminHrMiddleware(http.HandlerFunc(leaveHandler.ResolveLeaveRequest)))
			mux.Handle("GET /api/v1/companies/{id}/leave-policy", authMiddleware(http.HandlerFunc(leaveHandler.GetLeavePolicy)))
			mux.Handle("PUT /api/v1/companies/{id}/leave-policy", adminHrMiddleware(http.HandlerFunc(leaveHandler.UpdateLeavePolicy)))

			// Payroll calculation & CSV export routes (Queued & concurrency-gated for 0-cost DB safety)
			mux.Handle("GET /api/v1/payroll/calculate", adminHrMiddleware(http.HandlerFunc(heavyQueue.Middleware(payrollHandler.Calculate))))
			mux.Handle("GET /api/v1/payroll/export", adminHrMiddleware(http.HandlerFunc(heavyQueue.Middleware(payrollHandler.Export))))
			mux.Handle("GET /api/v1/companies/{id}/payroll/calculate", adminHrMiddleware(http.HandlerFunc(heavyQueue.Middleware(payrollHandler.Calculate))))
			mux.Handle("GET /api/v1/companies/{id}/payroll/export", adminHrMiddleware(http.HandlerFunc(heavyQueue.Middleware(payrollHandler.Export))))
		}
	} else {
		log.Println("DATABASE_URL not set, database features disabled")
	}

	mux.HandleFunc("GET /", healthHandler(dbConnected))

	// Wrap entire mux with security headers and CORS
	handler := securityHeadersMiddleware(cfg.Environment == "production", corsMiddleware(cfg.CORSAllowedOrigins, mux))

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
