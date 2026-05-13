package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/alexanderzull/file-converter/internal/converter"
	"github.com/alexanderzull/file-converter/internal/db"
	"github.com/alexanderzull/file-converter/internal/storage"
)

type Worker struct {
	log       *slog.Logger
	repo      *db.Repository
	storage   *storage.Local
	converter *converter.Service
	pollEvery time.Duration
}

func NewWorker(log *slog.Logger, repo *db.Repository, storage *storage.Local, converter *converter.Service, pollEvery time.Duration) *Worker {
	return &Worker{
		log:       log,
		repo:      repo,
		storage:   storage,
		converter: converter,
		pollEvery: pollEvery,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollEvery)
	defer ticker.Stop()

	w.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	job, err := w.repo.ClaimNextJob(ctx)
	if err != nil {
		w.log.Error("claim job failed", "err", err)
		return
	}
	if job == nil {
		return
	}

	output, err := w.storage.PrepareOutputPath(job.TargetFormat)
	if err != nil {
		w.failJob(ctx, job.ID, err)
		return
	}

	inputPath := w.storage.AbsPath(job.SourceStorageKey)
	if err := w.converter.Convert(ctx, inputPath, output.AbsPath); err != nil {
		w.failJob(ctx, job.ID, err)
		return
	}

	sizeBytes, checksum, err := fileInfo(output.AbsPath)
	if err != nil {
		w.failJob(ctx, job.ID, err)
		return
	}

	err = w.repo.RecordJobSuccess(ctx, job.ID, db.CreateStoredFileParams{
		UserID:       job.UserID,
		OriginalName: output.OriginalName,
		StorageKey:   output.StorageKey,
		MimeType:     converter.MIMEByFormat(filepath.Ext(output.AbsPath)),
		SizeBytes:    sizeBytes,
		ChecksumSHA:  checksum,
	})
	if err != nil {
		w.failJob(ctx, job.ID, err)
		return
	}

	w.log.Info("conversion completed", "job_id", job.ID, "target_format", job.TargetFormat)
}

func (w *Worker) failJob(ctx context.Context, jobID string, err error) {
	w.log.Error("conversion failed", "job_id", jobID, "err", err)
	_ = w.repo.MarkJobFailed(ctx, jobID, err.Error())
}

func fileInfo(path string) (int64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	hash := sha256.New()
	written, err := io.Copy(hash, f)
	if err != nil {
		return 0, "", err
	}
	return written, hex.EncodeToString(hash.Sum(nil)), nil
}
