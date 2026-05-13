package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

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
	dsn := cfg.MigrationDatabaseURL
	if dsn == "" {
		dsn = cfg.DatabaseURL
	}

	db, err := sql.Open("pgx", dsn)
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
		slog.Error("run migration", "command", command, "err", err)
		os.Exit(1)
	}

	fmt.Printf("migration command %q completed\n", command)
}
