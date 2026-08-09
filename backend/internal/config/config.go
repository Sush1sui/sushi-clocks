package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Environment        string
	DatabaseURL        string
	CORSAllowedOrigins string
	JWTSecret          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration

	// Super Admin Seeder configs
	SuperAdminEmail     string
	SuperAdminPassword  string
	SuperAdminFirstName string
	SuperAdminLastName  string
	SuperAdminMobile    string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	databaseURL := os.Getenv("DATABASE_URL")

	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:5173"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "sushi-clocks-default-development-jwt-secret-key-32b"
	}

	jwtAccessTTLStr := os.Getenv("JWT_ACCESS_TTL")
	jwtAccessTTL, err := time.ParseDuration(jwtAccessTTLStr)
	if err != nil || jwtAccessTTL == 0 {
		jwtAccessTTL = 15 * time.Minute
	}

	jwtRefreshTTLStr := os.Getenv("JWT_REFRESH_TTL")
	jwtRefreshTTL, err := time.ParseDuration(jwtRefreshTTLStr)
	if err != nil || jwtRefreshTTL == 0 {
		jwtRefreshTTL = 7 * 24 * time.Hour
	}

	return &Config{
		Port:                port,
		Environment:         env,
		DatabaseURL:         databaseURL,
		CORSAllowedOrigins:  corsOrigins,
		JWTSecret:           jwtSecret,
		JWTAccessTTL:        jwtAccessTTL,
		JWTRefreshTTL:       jwtRefreshTTL,
		SuperAdminEmail:     os.Getenv("SUPER_ADMIN_EMAIL"),
		SuperAdminPassword:  os.Getenv("SUPER_ADMIN_PASSWORD"),
		SuperAdminFirstName: os.Getenv("SUPER_ADMIN_FIRST_NAME"),
		SuperAdminLastName:  os.Getenv("SUPER_ADMIN_LAST_NAME"),
		SuperAdminMobile:    os.Getenv("SUPER_ADMIN_MOBILE"),
	}
}
