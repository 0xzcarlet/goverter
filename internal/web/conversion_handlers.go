package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/converter"
	"github.com/alexanderzull/file-converter/internal/db"
	"github.com/alexanderzull/file-converter/internal/quota"
	"github.com/alexanderzull/file-converter/internal/ui"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	jobs, usage, err := s.dashboardData(r.Context(), user.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, ui.Dashboard(s.cfg.AppName, &user, csrf.Token(r), csrf.TemplateField(r), nil, jobViews(jobs), s.cfg.AppMaxUploadMB, usage))
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	jobs, usage, err := s.dashboardData(r.Context(), user.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, ui.JobsPanel(jobViews(jobs), usage))
}

func (s *Server) handleCreateConversion(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes())
	if err := r.ParseMultipartForm(s.cfg.MaxUploadBytes()); err != nil {
		s.renderDashboardError(w, r, user, "upload exceeds configured size or multipart parsing failed")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.renderDashboardError(w, r, user, "select a source file first")
		return
	}
	defer file.Close()

	targetFormat := normalizeFormat(r.FormValue("target_format"))
	if err := converter.ValidatePair(header.Filename, targetFormat); err != nil {
		s.renderDashboardError(w, r, user, err.Error())
		return
	}

	saved, err := s.storage.SaveUpload(r.Context(), header.Filename, file)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	quotaDate := quota.DateForTime(time.Now())
	_, err = s.repo.CreateQueuedConversion(r.Context(), db.CreateQueuedConversionParams{
		UserID:       user.ID,
		TargetFormat: targetFormat,
		QuotaDate:    quotaDate,
		Limit:        s.cfg.AppDailyConversionLimit,
		SourceFile: db.CreateStoredFileParams{
			UserID:       user.ID,
			OriginalName: header.Filename,
			StorageKey:   saved.StorageKey,
			MimeType:     detectMimeType(header.Filename),
			SizeBytes:    saved.SizeBytes,
			ChecksumSHA:  saved.ChecksumSHA,
		},
	})
	if err != nil {
		_ = s.storage.Remove(saved.StorageKey)
		if errors.Is(err, db.ErrDailyLimitReached) {
			s.renderDashboardError(w, r, user, "daily conversion quota reached for today")
			return
		}
		s.internalError(w, r, err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleDownloadConversion(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	jobID := chi.URLParam(r, "jobID")
	file, err := s.repo.FetchDownloadableFile(r.Context(), user.ID, jobID)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrDownloadNotFound):
			http.NotFound(w, r)
		case errors.Is(err, db.ErrJobNotReady):
			http.Error(w, "conversion output is not ready", http.StatusConflict)
		default:
			s.internalError(w, r, err)
		}
		return
	}

	handle, err := s.storage.Open(file.StorageKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}
	defer handle.Close()

	info, err := handle.Stat()
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	filename := buildDownloadFilename(file.SourceName, file.TargetFormat)
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeContent(w, r, filename, info.ModTime(), handle)
}

func (s *Server) renderDashboardError(w http.ResponseWriter, r *http.Request, user auth.User, message string) {
	jobs, usage, err := s.dashboardData(r.Context(), user.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.render(w, r, http.StatusBadRequest, ui.Dashboard(s.cfg.AppName, &user, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: message}, jobViews(jobs), s.cfg.AppMaxUploadMB, usage))
}

func (s *Server) dashboardData(ctx context.Context, userID string) ([]db.ConversionJob, quota.Summary, error) {
	jobs, err := s.repo.ListJobs(ctx, userID, 20)
	if err != nil {
		return nil, quota.Summary{}, err
	}

	usage, err := s.repo.DailyUsageSummary(ctx, userID, quota.DateForTime(time.Now()))
	if err != nil {
		return nil, quota.Summary{}, err
	}

	return jobs, quota.Summary{
		Limit:          s.cfg.AppDailyConversionLimit,
		ReservedCount:  usage.ReservedCount,
		CompletedCount: usage.CompletedCount,
	}, nil
}

func jobViews(jobs []db.ConversionJob) []ui.Job {
	out := make([]ui.Job, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, ui.Job{
			ID:               job.ID,
			Status:           job.Status,
			TargetFormat:     job.TargetFormat,
			SourceFileName:   job.SourceFileName,
			OutputStorageKey: job.OutputStorageKey,
			ErrorMessage:     job.ErrorMessage,
			CreatedAt:        job.CreatedAt,
			FinishedAt:       job.FinishedAt,
		})
	}
	return out
}
