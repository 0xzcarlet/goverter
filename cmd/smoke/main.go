package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alexanderzull/file-converter/internal/config"
	"github.com/alexanderzull/file-converter/internal/converter"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}

	conversion := converter.New("ebook-convert")
	if err := conversion.Check(); err != nil {
		slog.Error("ebook-convert not available", "err", err)
		os.Exit(1)
	}

	version, err := conversion.Version(ctx)
	if err != nil {
		slog.Error("ebook-convert version check failed", "err", err)
		os.Exit(1)
	}

	fmt.Printf("database=ok\nebook-convert=%s\n", string(version))
}
