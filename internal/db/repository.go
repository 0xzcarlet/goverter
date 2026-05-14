package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDailyLimitReached = errors.New("daily conversion limit reached")
	ErrDownloadNotFound  = errors.New("download not found")
	ErrJobNotReady       = errors.New("job is not ready for download")
)

const (
	JobStatusQueued     = "queued"
	JobStatusProcessing = "processing"
	JobStatusDone       = "done"
	JobStatusFailed     = "failed"

	QuotaStatusReserved  = "reserved"
	QuotaStatusCompleted = "completed"
	QuotaStatusRefunded  = "refunded"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
