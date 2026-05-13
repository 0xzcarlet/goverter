package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDailyLimitReached = errors.New("daily conversion limit reached")
	ErrDownloadNotFound  = errors.New("download not found")
	ErrJobNotReady       = errors.New("job is not ready for download")
)

type Repository struct {
	pool *pgxpool.Pool
}

type CreateStoredFileParams struct {
	UserID       string
	OriginalName string
	StorageKey   string
	MimeType     string
	SizeBytes    int64
	ChecksumSHA  string
}

type StoredFile struct {
	ID           string
	UserID       string
	OriginalName string
	StorageKey   string
	MimeType     string
	SizeBytes    int64
	ChecksumSHA  *string
	CreatedAt    time.Time
}

type DailyUsage struct {
	UserID         string
	QuotaDate      time.Time
	ReservedCount  int
	CompletedCount int
}

type ConversionJob struct {
	ID               string
	UserID           string
	SourceFileID     string
	TargetFormat     string
	Status           string
	OutputFileID     *string
	OutputStorageKey *string
	ErrorMessage     *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
	SourceFileName   string
}

type CreateQueuedConversionParams struct {
	UserID       string
	TargetFormat string
	QuotaDate    time.Time
	Limit        int
	SourceFile   CreateStoredFileParams
}

type ClaimedJob struct {
	ConversionJob
	QuotaDate        time.Time
	QuotaStatus      string
	SourceStorageKey string
	SourceMIMEType   string
}

type DownloadableFile struct {
	StorageKey   string
	MimeType     string
	TargetFormat string
	SourceName   string
	OutputName   string
	OutputFileID string
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repository) CreateQueuedConversion(ctx context.Context, params CreateQueuedConversionParams) (ConversionJob, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ConversionJob{}, err
	}
	defer tx.Rollback(ctx)

	if err := reserveDailySlot(ctx, tx, params.UserID, params.QuotaDate, params.Limit); err != nil {
		return ConversionJob{}, err
	}

	storedFile, err := createStoredFile(ctx, tx, params.SourceFile)
	if err != nil {
		return ConversionJob{}, err
	}

	job, err := createJob(ctx, tx, params.UserID, storedFile.ID, params.TargetFormat, params.QuotaDate)
	if err != nil {
		return ConversionJob{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ConversionJob{}, err
	}
	return job, nil
}

func (r *Repository) DailyUsageSummary(ctx context.Context, userID string, quotaDate time.Time) (DailyUsage, error) {
	const query = `
		select user_id, quota_date, reserved_count, completed_count
		from app.daily_conversion_usage
		where user_id = $1 and quota_date = $2
	`

	var usage DailyUsage
	err := r.pool.QueryRow(ctx, query, userID, quotaDateValue(quotaDate)).Scan(
		&usage.UserID,
		&usage.QuotaDate,
		&usage.ReservedCount,
		&usage.CompletedCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DailyUsage{UserID: userID, QuotaDate: quotaDate}, nil
		}
		return DailyUsage{}, err
	}
	return usage, nil
}

func (r *Repository) ListJobs(ctx context.Context, userID string, limit int) ([]ConversionJob, error) {
	const query = `
		select
			j.id,
			j.user_id,
			j.source_file_id,
			j.target_format,
			j.status,
			j.output_file_id,
			of.storage_key,
			j.error_message,
			j.created_at,
			j.updated_at,
			j.started_at,
			j.finished_at,
			sf.original_name
		from app.conversion_jobs j
		join app.stored_files sf on sf.id = j.source_file_id
		left join app.stored_files of on of.id = j.output_file_id
		where j.user_id = $1
		order by j.created_at desc
		limit $2
	`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []ConversionJob
	for rows.Next() {
		var job ConversionJob
		if err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.SourceFileID,
			&job.TargetFormat,
			&job.Status,
			&job.OutputFileID,
			&job.OutputStorageKey,
			&job.ErrorMessage,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.StartedAt,
			&job.FinishedAt,
			&job.SourceFileName,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) FetchDownloadableFile(ctx context.Context, userID, jobID string) (DownloadableFile, error) {
	const query = `
		select
			of.id,
			of.storage_key,
			of.mime_type,
			of.original_name,
			sf.original_name,
			j.target_format,
			j.status
		from app.conversion_jobs j
		join app.stored_files sf on sf.id = j.source_file_id
		left join app.stored_files of on of.id = j.output_file_id
		where j.id = $1 and j.user_id = $2
	`

	var file DownloadableFile
	var status string
	var outputID *string
	var storageKey *string
	var mimeType *string
	var outputName *string
	err := r.pool.QueryRow(ctx, query, jobID, userID).Scan(
		&outputID,
		&storageKey,
		&mimeType,
		&outputName,
		&file.SourceName,
		&file.TargetFormat,
		&status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DownloadableFile{}, ErrDownloadNotFound
		}
		return DownloadableFile{}, err
	}
	if status != "done" || outputID == nil || storageKey == nil || mimeType == nil || outputName == nil {
		return DownloadableFile{}, ErrJobNotReady
	}
	file.OutputFileID = *outputID
	file.StorageKey = *storageKey
	file.MimeType = *mimeType
	file.OutputName = *outputName
	return file, nil
}

func (r *Repository) ClaimNextJob(ctx context.Context) (*ClaimedJob, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const query = `
		with next_job as (
			select j.id
			from app.conversion_jobs j
			where j.status = 'queued'
			order by j.created_at asc
			for update skip locked
			limit 1
		)
		update app.conversion_jobs j
		set status = 'processing',
			started_at = now(),
			updated_at = now()
		from next_job
		where j.id = next_job.id
		returning j.id, j.user_id, j.source_file_id, j.target_format, j.status, j.output_file_id, j.error_message, j.created_at, j.updated_at, j.started_at, j.finished_at, j.quota_date, j.quota_status
	`
	var job ClaimedJob
	err = tx.QueryRow(ctx, query).Scan(
		&job.ID,
		&job.UserID,
		&job.SourceFileID,
		&job.TargetFormat,
		&job.Status,
		&job.OutputFileID,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.StartedAt,
		&job.FinishedAt,
		&job.QuotaDate,
		&job.QuotaStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	err = tx.QueryRow(ctx, `
		select original_name, storage_key, mime_type
		from app.stored_files
		where id = $1
	`, job.SourceFileID).Scan(&job.SourceFileName, &job.SourceStorageKey, &job.SourceMIMEType)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *Repository) MarkJobFailed(ctx context.Context, jobID, message string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID string
	var quotaDate time.Time
	var quotaStatus string
	err = tx.QueryRow(ctx, `
		select user_id, quota_date, quota_status
		from app.conversion_jobs
		where id = $1
		for update
	`, jobID).Scan(&userID, &quotaDate, &quotaStatus)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		update app.conversion_jobs
		set status = 'failed',
			error_message = $2,
			quota_status = case when quota_status = 'reserved' then 'refunded' else quota_status end,
			finished_at = now(),
			updated_at = now()
		where id = $1
	`, jobID, message)
	if err != nil {
		return err
	}

	if quotaStatus == "reserved" {
		if err := adjustUsageCounts(ctx, tx, userID, quotaDate, -1, 0); err != nil {
			return fmt.Errorf("refund quota slot: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) RecordJobSuccess(ctx context.Context, jobID string, params CreateStoredFileParams) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID string
	var quotaDate time.Time
	var quotaStatus string
	err = tx.QueryRow(ctx, `
		select user_id, quota_date, quota_status
		from app.conversion_jobs
		where id = $1
		for update
	`, jobID).Scan(&userID, &quotaDate, &quotaStatus)
	if err != nil {
		return err
	}

	storedFile, err := createStoredFile(ctx, tx, params)
	if err != nil {
		return fmt.Errorf("insert output file: %w", err)
	}

	_, err = tx.Exec(ctx, `
		update app.conversion_jobs
		set status = 'done',
			output_file_id = $2,
			error_message = null,
			quota_status = case when quota_status = 'reserved' then 'completed' else quota_status end,
			finished_at = now(),
			updated_at = now()
		where id = $1
	`, jobID, storedFile.ID)
	if err != nil {
		return fmt.Errorf("mark job success: %w", err)
	}

	if quotaStatus == "reserved" {
		if err := adjustUsageCounts(ctx, tx, userID, quotaDate, -1, 1); err != nil {
			return fmt.Errorf("finalize quota slot: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func reserveDailySlot(ctx context.Context, tx pgx.Tx, userID string, quotaDate time.Time, limit int) error {
	quotaDateArg := quotaDateValue(quotaDate)
	_, err := tx.Exec(ctx, `
		insert into app.daily_conversion_usage (user_id, quota_date, reserved_count, completed_count)
		values ($1, $2, 0, 0)
		on conflict (user_id, quota_date) do nothing
	`, userID, quotaDateArg)
	if err != nil {
		return err
	}

	var reservedCount int
	var completedCount int
	err = tx.QueryRow(ctx, `
		select reserved_count, completed_count
		from app.daily_conversion_usage
		where user_id = $1 and quota_date = $2
		for update
	`, userID, quotaDateArg).Scan(&reservedCount, &completedCount)
	if err != nil {
		return err
	}

	if reservedCount+completedCount >= limit {
		return ErrDailyLimitReached
	}

	_, err = tx.Exec(ctx, `
		update app.daily_conversion_usage
		set reserved_count = reserved_count + 1,
			updated_at = now()
		where user_id = $1 and quota_date = $2
	`, userID, quotaDateArg)
	return err
}

func adjustUsageCounts(ctx context.Context, tx pgx.Tx, userID string, quotaDate time.Time, reservedDelta, completedDelta int) error {
	quotaDateArg := quotaDateValue(quotaDate)
	commandTag, err := tx.Exec(ctx, `
		update app.daily_conversion_usage
		set reserved_count = reserved_count + $3,
			completed_count = completed_count + $4,
			updated_at = now()
		where user_id = $1 and quota_date = $2
	`, userID, quotaDateArg, reservedDelta, completedDelta)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("daily usage row missing for user %s on %s", userID, quotaDate.Format("2006-01-02"))
	}
	return nil
}

func createStoredFile(ctx context.Context, tx pgx.Tx, params CreateStoredFileParams) (StoredFile, error) {
	const query = `
		insert into app.stored_files (
			id, user_id, original_name, storage_key, mime_type, size_bytes, checksum_sha256
		) values ($1, $2, $3, $4, $5, $6, $7)
		returning id, user_id, original_name, storage_key, mime_type, size_bytes, checksum_sha256, created_at
	`
	row := tx.QueryRow(ctx, query,
		uuid.NewString(),
		params.UserID,
		params.OriginalName,
		params.StorageKey,
		params.MimeType,
		params.SizeBytes,
		nullIfEmpty(params.ChecksumSHA),
	)
	var file StoredFile
	err := row.Scan(
		&file.ID,
		&file.UserID,
		&file.OriginalName,
		&file.StorageKey,
		&file.MimeType,
		&file.SizeBytes,
		&file.ChecksumSHA,
		&file.CreatedAt,
	)
	return file, err
}

func createJob(ctx context.Context, tx pgx.Tx, userID, sourceFileID, targetFormat string, quotaDate time.Time) (ConversionJob, error) {
	const query = `
		insert into app.conversion_jobs (
			id, user_id, source_file_id, target_format, status, quota_date, quota_status
		) values ($1, $2, $3, $4, 'queued', $5, 'reserved')
		returning id, user_id, source_file_id, target_format, status, output_file_id, error_message, created_at, updated_at, started_at, finished_at
	`
	row := tx.QueryRow(ctx, query, uuid.NewString(), userID, sourceFileID, targetFormat, quotaDateValue(quotaDate))
	var job ConversionJob
	err := row.Scan(
		&job.ID,
		&job.UserID,
		&job.SourceFileID,
		&job.TargetFormat,
		&job.Status,
		&job.OutputFileID,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.StartedAt,
		&job.FinishedAt,
	)
	return job, err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func quotaDateValue(value time.Time) string {
	return value.Format("2006-01-02")
}
