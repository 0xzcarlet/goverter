package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
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
	"github.com/alexanderzull/file-converter/internal/converter"
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
		repo: &fakeRepository{
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
		repo: &fakeRepository{downloadErr: db.ErrJobNotReady},
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
		repo: &fakeRepository{
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

func TestHandleGuestConversionReturnsConvertedFile(t *testing.T) {
	rootDir := t.TempDir()
	fileStorage := storage.New(rootDir, filepath.Join(rootDir, "uploads"), filepath.Join(rootDir, "outputs"), filepath.Join(rootDir, "tmp"))
	if err := fileStorage.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	repo := &fakeRepository{}
	srv := &Server{
		cfg: config.Config{
			AppSessionCookieName:    "fc_session",
			AppDailyConversionLimit: 3,
			AppMaxUploadMB:          25,
		},
		log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo:      repo,
		storage:   fileStorage,
		converter: newTestConverter(t, "#!/bin/sh\ncp \"$1\" \"$2\"\n"),
	}

	req := newGuestConversionRequest(t, "book.pdf", "epub", "fake pdf payload")
	rr := httptest.NewRecorder()

	srv.handleGuestConversion(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != converter.MIMEEPUB {
		t.Fatalf("Content-Type = %q, want %q", got, converter.MIMEEPUB)
	}
	if got := rr.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="book.epub"`) {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if body := rr.Body.String(); body != "fake pdf payload" {
		t.Fatalf("body = %q, want %q", body, "fake pdf payload")
	}
	if repo.reserveGuestCalls != 1 {
		t.Fatalf("reserveGuestCalls = %d, want 1", repo.reserveGuestCalls)
	}
	if repo.completeGuestCalls != 1 {
		t.Fatalf("completeGuestCalls = %d, want 1", repo.completeGuestCalls)
	}
	if repo.refundGuestCalls != 0 {
		t.Fatalf("refundGuestCalls = %d, want 0", repo.refundGuestCalls)
	}
	if len(rr.Result().Cookies()) == 0 {
		t.Fatal("expected guest cookie to be set")
	}
	assertNoFilesInDir(t, filepath.Join(rootDir, "uploads"))
	assertNoFilesInDir(t, filepath.Join(rootDir, "outputs"))
}

func TestHandleGuestConversionReturnsQuotaErrorWhenDailyLimitReached(t *testing.T) {
	srv := &Server{
		cfg: config.Config{
			AppSessionCookieName:    "fc_session",
			AppDailyConversionLimit: 3,
			AppMaxUploadMB:          25,
		},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo: &fakeRepository{
			guestSummary:    db.GuestDailyUsage{CompletedCount: 1},
			reserveGuestErr: db.ErrDailyLimitReached,
		},
		storage:   fakeFileStorage{},
		converter: newTestConverter(t, "#!/bin/sh\ncp \"$1\" \"$2\"\n"),
	}

	req := newGuestConversionRequest(t, "book.pdf", "epub", "fake pdf payload")
	rr := httptest.NewRecorder()

	srv.handleGuestConversion(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
	if !strings.Contains(rr.Body.String(), "Jatah guest untuk hari ini sudah habis") {
		t.Fatalf("body = %q, want daily limit message", rr.Body.String())
	}
}

func TestHandleGuestConversionRejectsInvalidPairWithoutReservingQuota(t *testing.T) {
	repo := &fakeRepository{}
	srv := &Server{
		cfg: config.Config{
			AppSessionCookieName:    "fc_session",
			AppDailyConversionLimit: 3,
			AppMaxUploadMB:          25,
		},
		log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo:      repo,
		storage:   fakeFileStorage{},
		converter: newTestConverter(t, "#!/bin/sh\ncp \"$1\" \"$2\"\n"),
	}

	req := newGuestConversionRequest(t, "book.pdf", "pdf", "fake pdf payload")
	rr := httptest.NewRecorder()

	srv.handleGuestConversion(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if repo.reserveGuestCalls != 0 {
		t.Fatalf("reserveGuestCalls = %d, want 0", repo.reserveGuestCalls)
	}
}

func TestHandleGuestConversionRefundsQuotaOnConverterFailure(t *testing.T) {
	rootDir := t.TempDir()
	fileStorage := storage.New(rootDir, filepath.Join(rootDir, "uploads"), filepath.Join(rootDir, "outputs"), filepath.Join(rootDir, "tmp"))
	if err := fileStorage.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	repo := &fakeRepository{}
	srv := &Server{
		cfg: config.Config{
			AppSessionCookieName:    "fc_session",
			AppDailyConversionLimit: 3,
			AppMaxUploadMB:          25,
		},
		log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo:      repo,
		storage:   fileStorage,
		converter: newTestConverter(t, "#!/bin/sh\necho boom >&2\nexit 1\n"),
	}

	req := newGuestConversionRequest(t, "book.pdf", "epub", "fake pdf payload")
	rr := httptest.NewRecorder()

	srv.handleGuestConversion(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if repo.refundGuestCalls != 1 {
		t.Fatalf("refundGuestCalls = %d, want 1", repo.refundGuestCalls)
	}
	if !strings.Contains(rr.Body.String(), "Konversi gagal diproses") {
		t.Fatalf("body = %q, want conversion failure message", rr.Body.String())
	}
}

func TestHandleLandingShowsDashboardCTAForLoggedInUser(t *testing.T) {
	srv := &Server{
		cfg: config.Config{
			AppName:              "File Converter",
			AppSessionCookieName: "fc_session",
			AppMaxUploadMB:       25,
		},
		log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo: &fakeRepository{},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withUser(req.Context(), auth.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	srv.handleLanding(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if strings.Contains(rr.Body.String(), `action="/guest/conversions"`) {
		t.Fatalf("landing should not show guest form for logged-in user: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `href="/dashboard"`) {
		t.Fatalf("landing missing dashboard link: %s", rr.Body.String())
	}
}

type fakeRepository struct {
	downloadFile db.DownloadableFile
	downloadErr  error
	guestSummary db.GuestDailyUsage

	reserveGuestErr error
	guestSummaryErr error

	reserveGuestCalls  int
	completeGuestCalls int
	refundGuestCalls   int
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

func (f fakeFileStorage) PrepareOutputPath(string) (storage.SavedFile, error) {
	return storage.SavedFile{}, nil
}

func (f fakeFileStorage) AbsPath(string) string {
	return ""
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

func (f *fakeRepository) GuestDailyUsageSummary(context.Context, string, time.Time) (db.GuestDailyUsage, error) {
	if f.guestSummaryErr != nil {
		return db.GuestDailyUsage{}, f.guestSummaryErr
	}
	return f.guestSummary, nil
}

func (f *fakeRepository) ReserveGuestDailySlot(context.Context, string, time.Time, int) error {
	f.reserveGuestCalls++
	return f.reserveGuestErr
}

func (f *fakeRepository) CompleteGuestDailySlot(context.Context, string, time.Time) error {
	f.completeGuestCalls++
	return nil
}

func (f *fakeRepository) RefundGuestDailySlot(context.Context, string, time.Time) error {
	f.refundGuestCalls++
	return nil
}

func withRouteParam(r *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}

func newGuestConversionRequest(t *testing.T, filename, targetFormat, contents string) *http.Request {
	t.Helper()

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := io.WriteString(part, contents); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := writer.WriteField("target_format", targetFormat); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/guest/conversions", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func newTestConverter(t *testing.T, script string) *converter.Service {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "ebook-convert")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return converter.New(path)
}

func assertNoFilesInDir(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	if len(entries) != 0 {
		t.Fatalf("dir %q still has %d entries", dir, len(entries))
	}
}
