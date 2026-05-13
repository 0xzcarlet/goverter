package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/config"
	"github.com/alexanderzull/file-converter/internal/converter"
	"github.com/alexanderzull/file-converter/internal/db"
	"github.com/alexanderzull/file-converter/internal/jobs"
	"github.com/alexanderzull/file-converter/internal/storage"
	"github.com/alexanderzull/file-converter/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	fileStorage := storage.New(cfg.AppDataDir, cfg.UploadDir(), cfg.OutputDir(), cfg.TempDir())
	if err := fileStorage.EnsureDirs(); err != nil {
		logger.Error("prepare storage", "err", err)
		os.Exit(1)
	}

	repo := db.New(pool)
	authClient := auth.New(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	conversion := converter.New("ebook-convert")
	worker := jobs.NewWorker(logger.With("component", "worker"), repo, fileStorage, conversion, cfg.AppWorkerPoll)
	go worker.Run(ctx)

	server := web.New(cfg, logger, repo, authClient, fileStorage, conversion)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           server.Router(),
		ReadTimeout:       cfg.AppHTTPReadTimeout,
		WriteTimeout:      cfg.AppHTTPWriteTimeout,
		IdleTimeout:       cfg.AppHTTPIdleTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("starting http server", "addr", cfg.ListenAddr(), "env", cfg.AppEnv)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server stopped", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "err", err)
	}
}
