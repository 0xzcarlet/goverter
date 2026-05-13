package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/config"
	"github.com/alexanderzull/file-converter/internal/db"
	"github.com/alexanderzull/file-converter/internal/storage"
)

func TestHandleDownloadConversionReturnsFileForOwner(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "result.epub")
	if err := os.WriteFile(filePath, []byte("converted"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	srv := &Server{
		cfg: config.Config{AppDailyConversionLimit: 3},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo: fakeRepository{
			downloadFile: db.DownloadableFile{
				StorageKey:   "outputs/result.epub",
				MimeType:     "application/epub+zip",
				TargetFormat: "epub",
				SourceName:   "book.pdf",
			},
		},
		storage: fakeFileStorage{openPaths: map[string]string{
			"outputs/result.epub": filePath,
		}},
	}

	req := httptest.NewRequest(http.MethodGet, "/app/conversions/job-1/download", nil)
	req = req.WithContext(withUser(req.Context(), auth.User{ID: "user-1"}))
	req = withRouteParam(req, "jobID", "job-1")
	rr := httptest.NewRecorder()

	srv.handleDownloadConversion(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="book.epub"`) {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/epub+zip" {
		t.Fatalf("Content-Type = %q, want application/epub+zip", got)
	}
	if body := rr.Body.String(); body != "converted" {
		t.Fatalf("body = %q, want converted", body)
	}
}

func TestHandleDownloadConversionRejectsNonReadyJob(t *testing.T) {
	srv := &Server{
		cfg:  config.Config{AppDailyConversionLimit: 3},
		log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo: fakeRepository{downloadErr: db.ErrJobNotReady},
	}

	req := httptest.NewRequest(http.MethodGet, "/app/conversions/job-1/download", nil)
	req = req.WithContext(withUser(req.Context(), auth.User{ID: "user-1"}))
	req = withRouteParam(req, "jobID", "job-1")
	rr := httptest.NewRecorder()

	srv.handleDownloadConversion(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
}

func TestHandleDownloadConversionReturnsNotFoundForMissingFile(t *testing.T) {
	srv := &Server{
		cfg: config.Config{AppDailyConversionLimit: 3},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo: fakeRepository{
			downloadFile: db.DownloadableFile{
				StorageKey:   "outputs/missing.epub",
				MimeType:     "application/epub+zip",
				TargetFormat: "epub",
				SourceName:   "book.pdf",
			},
		},
		storage: fakeFileStorage{openErr: os.ErrNotExist},
	}

	req := httptest.NewRequest(http.MethodGet, "/app/conversions/job-1/download", nil)
	req = req.WithContext(withUser(req.Context(), auth.User{ID: "user-1"}))
	req = withRouteParam(req, "jobID", "job-1")
	rr := httptest.NewRecorder()

	srv.handleDownloadConversion(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestBuildDownloadFilenameSanitizesSourceName(t *testing.T) {
	got := buildDownloadFilename("../weird\nname.pdf", "epub")
	if got != "weird name.epub" {
		t.Fatalf("buildDownloadFilename() = %q, want %q", got, "weird name.epub")
	}
}

type fakeRepository struct {
	downloadFile db.DownloadableFile
	downloadErr  error
}

func (f fakeRepository) Ping(context.Context) error { return nil }

func (f fakeRepository) ListJobs(context.Context, string, int) ([]db.ConversionJob, error) {
	return nil, nil
}

func (f fakeRepository) DailyUsageSummary(context.Context, string, time.Time) (db.DailyUsage, error) {
	return db.DailyUsage{}, nil
}

func (f fakeRepository) CreateQueuedConversion(context.Context, db.CreateQueuedConversionParams) (db.ConversionJob, error) {
	return db.ConversionJob{}, nil
}

func (f fakeRepository) FetchDownloadableFile(context.Context, string, string) (db.DownloadableFile, error) {
	if f.downloadErr != nil {
		return db.DownloadableFile{}, f.downloadErr
	}
	return f.downloadFile, nil
}

type fakeFileStorage struct {
	openPaths map[string]string
	openErr   error
}

func (f fakeFileStorage) SaveUpload(context.Context, string, io.Reader) (storage.SavedFile, error) {
	return storage.SavedFile{}, nil
}

func (f fakeFileStorage) Open(storageKey string) (*os.File, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	path, ok := f.openPaths[storageKey]
	if !ok {
		return nil, errors.New("unexpected storage key")
	}
	return os.Open(path)
}

func (f fakeFileStorage) Remove(string) error { return nil }

func withRouteParam(r *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}
