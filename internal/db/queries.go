package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

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
	if status != JobStatusDone || outputID == nil || storageKey == nil || mimeType == nil || outputName == nil {
		return DownloadableFile{}, ErrJobNotReady
	}
	file.OutputFileID = *outputID
	file.StorageKey = *storageKey
	file.MimeType = *mimeType
	file.OutputName = *outputName
	return file, nil
}
