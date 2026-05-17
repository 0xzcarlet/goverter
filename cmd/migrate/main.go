package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/alexanderzull/file-converter/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, source, err := openMigrationDatabase(cfg)
	if err != nil {
		slog.Error("open migration database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("set goose dialect", "err", err)
		os.Exit(1)
	}

	if err := goose.Run(command, db, "db/migrations"); err != nil {
		slog.Error("run migration", "command", command, "database", source, "err", err)
		os.Exit(1)
	}

	fmt.Printf("migration command %q completed using %s\n", command, source)
}

func openMigrationDatabase(cfg config.Config) (*sql.DB, string, error) {
	candidates := migrationCandidates(cfg)
	var attempts []string

	for idx, candidate := range candidates {
		db, err := sql.Open("pgx", candidate.dsn)
		if err != nil {
			attempts = append(attempts, fmt.Sprintf("%s: %v", candidate.name, err))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err != nil {
			_ = db.Close()
			attempts = append(attempts, fmt.Sprintf("%s: %v", candidate.name, err))
			continue
		}

		if idx > 0 {
			slog.Warn("primary migration database unavailable, falling back", "database", candidate.name)
		}
		return db, candidate.name, nil
	}

	if len(attempts) == 0 {
		return nil, "", errors.New("no database DSN configured for migrations")
	}

	return nil, "", fmt.Errorf("all migration database connection attempts failed:\n - %s", strings.Join(attempts, "\n - "))
}

func migrationCandidates(cfg config.Config) []struct {
	name string
	dsn  string
} {
	candidates := []struct {
		name string
		dsn  string
	}{}

	add := func(name, dsn string) {
		dsn = strings.TrimSpace(dsn)
		if dsn == "" {
			return
		}
		for _, existing := range candidates {
			if existing.dsn == dsn {
				return
			}
		}
		candidates = append(candidates, struct {
			name string
			dsn  string
		}{name: name, dsn: dsn})
	}

	add("MIGRATION_DATABASE_URL", cfg.MigrationDatabaseURL)
	add("DATABASE_URL", cfg.DatabaseURL)

	return candidates
}
