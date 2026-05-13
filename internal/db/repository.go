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

type CreateJobParams struct {
	UserID       string
	SourceFileID string
	TargetFormat string
}

type ClaimedJob struct {
	ConversionJob
	SourceStorageKey string
	SourceMIMEType   string
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repository) CreateStoredFile(ctx context.Context, params CreateStoredFileParams) (StoredFile, error) {
	const query = `
		insert into app.stored_files (
			id, user_id, original_name, storage_key, mime_type, size_bytes, checksum_sha256
		) values ($1, $2, $3, $4, $5, $6, $7)
		returning id, user_id, original_name, storage_key, mime_type, size_bytes, checksum_sha256, created_at
	`
	row := r.pool.QueryRow(ctx, query,
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

func (r *Repository) CreateJob(ctx context.Context, params CreateJobParams) (ConversionJob, error) {
	const query = `
		insert into app.conversion_jobs (
			id, user_id, source_file_id, target_format, status
		) values ($1, $2, $3, $4, 'queued')
		returning id, user_id, source_file_id, target_format, status, output_file_id, error_message, created_at, updated_at, started_at, finished_at
	`
	row := r.pool.QueryRow(ctx, query, uuid.NewString(), params.UserID, params.SourceFileID, params.TargetFormat)
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
		returning j.id, j.user_id, j.source_file_id, j.target_format, j.status, j.output_file_id, j.error_message, j.created_at, j.updated_at, j.started_at, j.finished_at
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
	const query = `
		update app.conversion_jobs
		set status = 'failed',
			error_message = $2,
			finished_at = now(),
			updated_at = now()
		where id = $1
	`
	_, err := r.pool.Exec(ctx, query, jobID, message)
	return err
}

func (r *Repository) RecordJobSuccess(ctx context.Context, jobID string, params CreateStoredFileParams) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	outputID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		insert into app.stored_files (
			id, user_id, original_name, storage_key, mime_type, size_bytes, checksum_sha256
		) values ($1, $2, $3, $4, $5, $6, $7)
	`, outputID, params.UserID, params.OriginalName, params.StorageKey, params.MimeType, params.SizeBytes, nullIfEmpty(params.ChecksumSHA))
	if err != nil {
		return fmt.Errorf("insert output file: %w", err)
	}

	_, err = tx.Exec(ctx, `
		update app.conversion_jobs
		set status = 'done',
			output_file_id = $2,
			error_message = null,
			finished_at = now(),
			updated_at = now()
		where id = $1
	`, jobID, outputID)
	if err != nil {
		return fmt.Errorf("mark job success: %w", err)
	}

	return tx.Commit(ctx)
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
