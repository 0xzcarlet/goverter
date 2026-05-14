package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

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

	if quotaStatus == QuotaStatusReserved {
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

	if quotaStatus == QuotaStatusReserved {
		if err := adjustUsageCounts(ctx, tx, userID, quotaDate, -1, 1); err != nil {
			return fmt.Errorf("finalize quota slot: %w", err)
		}
	}

	return tx.Commit(ctx)
}
