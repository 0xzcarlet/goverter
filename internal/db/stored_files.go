package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
