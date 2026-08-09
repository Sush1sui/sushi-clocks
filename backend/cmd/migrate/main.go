package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

/*
# apply all migrations
go run ./cmd/migrate -cmd=up

# fresh migration (drop all and re-apply)
go run ./cmd/migrate -cmd=fresh

# check version
go run ./cmd/migrate -cmd=version

# roll back 1 step
go run ./cmd/migrate -cmd=down -steps=1
*/

func main() {
	// Flags
	cmd := flag.String("cmd", "up", "migration command: up | down | fresh | version | force | drop")
	steps := flag.Int("steps", 0, "number of steps for down (0 = all)")
	forceVersion := flag.Int("force", -1, "force set version (use with -cmd=force)")
	migrationsPath := flag.String("path", "migrations", "path to migrations directory")
	flag.Parse()

	// Load env
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL env var is required")
	}

	// golang-migrate pgx/v5 driver registers as "pgx5" scheme, not "postgres"
	migrateURL := strings.Replace(databaseURL, "postgres://", "pgx5://", 1)
	migrateURL = strings.Replace(migrateURL, "postgresql://", "pgx5://", 1)
	sourceURL := fmt.Sprintf("file://%s", *migrationsPath)

	m, err := migrate.New(sourceURL, migrateURL)
	if err != nil {
		log.Fatalf("create migrate instance error: %v", err)
	}
	var alreadyClosed bool
	defer func() {
		if !alreadyClosed {
			srcErr, dbErr := m.Close()
			if srcErr != nil {
				log.Printf("migrate source close error: %v", srcErr)
			}
			if dbErr != nil {
				log.Printf("migrate db close error: %v", dbErr)
			}
		}
	}()

	switch *cmd {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate up error: %v", err)
		}
		version, dirty, _ := m.Version()
		log.Printf("migrate up complete — version: %d, dirty: %v", version, dirty)

	case "fresh":
		log.Println("dropping all tables and schema...")
		if err := m.Drop(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate drop error: %v", err)
		}
		log.Println("drop complete. Re-applying all migrations...")

		alreadyClosed = true
		_, _ = m.Close()

		mFresh, err := migrate.New(sourceURL, migrateURL)
		if err != nil {
			log.Fatalf("recreate migrate instance error: %v", err)
		}
		defer func() {
			_, _ = mFresh.Close()
		}()

		if err := mFresh.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate fresh up error: %v", err)
		}
		version, dirty, _ := mFresh.Version()
		log.Printf("migrate fresh complete — version: %d, dirty: %v", version, dirty)

	case "down":
		if *steps == 0 {
			if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("migrate down error: %v", err)
			}
			log.Println("migrate down complete — all migrations rolled back")
		} else {
			if err := m.Steps(-(*steps)); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("migrate down %d steps error: %v", *steps, err)
			}
			version, dirty, _ := m.Version()
			log.Printf("migrate down %d steps complete — version: %d, dirty: %v", *steps, version, dirty)
		}

	case "drop":
		if err := m.Drop(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate drop error: %v", err)
		}
		log.Println("migrate drop complete — all tables and schema dropped")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("get version error: %v", err)
		}
		log.Printf("current version: %d, dirty: %v", version, dirty)

	case "force":
		if *forceVersion < 0 {
			log.Fatal("force requires -force=<version>")
		}
		if err := m.Force(*forceVersion); err != nil {
			log.Fatalf("force version error: %v", err)
		}
		log.Printf("forced version to %d", *forceVersion)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\nvalid commands: up | down | fresh | drop | version | force\n", *cmd)
		os.Exit(1)
	}
}
