package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/config"
	"github.com/alexanderzull/file-converter/internal/converter"
	"github.com/alexanderzull/file-converter/internal/db"
	"github.com/alexanderzull/file-converter/internal/storage"
)

type repository interface {
	Ping(ctx context.Context) error
	ListJobs(ctx context.Context, userID string, limit int) ([]db.ConversionJob, error)
	DailyUsageSummary(ctx context.Context, userID string, quotaDate time.Time) (db.DailyUsage, error)
	GuestDailyUsageSummary(ctx context.Context, guestToken string, quotaDate time.Time) (db.GuestDailyUsage, error)
	ReserveGuestDailySlot(ctx context.Context, guestToken string, quotaDate time.Time, limit int) error
	CompleteGuestDailySlot(ctx context.Context, guestToken string, quotaDate time.Time) error
	RefundGuestDailySlot(ctx context.Context, guestToken string, quotaDate time.Time) error
	CreateQueuedConversion(ctx context.Context, params db.CreateQueuedConversionParams) (db.ConversionJob, error)
	FetchDownloadableFile(ctx context.Context, userID, jobID string) (db.DownloadableFile, error)
}

type fileStorage interface {
	SaveUpload(ctx context.Context, originalName string, source io.Reader) (storage.SavedFile, error)
	PrepareOutputPath(targetFormat string) (storage.SavedFile, error)
	AbsPath(storageKey string) string
	Open(storageKey string) (*os.File, error)
	Remove(storageKey string) error
}

type Server struct {
	cfg       config.Config
	log       *slog.Logger
	repo      repository
	auth      *auth.Client
	storage   fileStorage
	converter *converter.Service
}

func New(cfg config.Config, log *slog.Logger, repo repository, authClient *auth.Client, storage fileStorage, converter *converter.Service) *Server {
	return &Server{
		cfg:       cfg,
		log:       log,
		repo:      repo,
		auth:      authClient,
		storage:   storage,
		converter: converter,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(s.cfg.AppHTTPWriteTimeout))
	r.Use(s.loggingMiddleware)
	r.Use(s.securityHeaders)

	csrfMiddleware := csrf.Protect(
		[]byte(s.cfg.CSRFAuthKey),
		csrf.Secure(s.cfg.AppSecureCookies),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.CookieName(s.cfg.AppSessionCookieName+"_csrf"),
		csrf.TrustedOrigins(csrfTrustedOrigins(s.cfg)),
	)

	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))

	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Group(func(public chi.Router) {
		public.Use(csrfMiddleware)
		public.Get("/", s.handleLanding)
		public.Post("/guest/conversions", s.handleGuestConversion)
		public.Get("/login", s.handleLoginForm)
		public.Post("/login", s.handleLogin)
		public.Get("/register", s.handleRegisterForm)
		public.Post("/register", s.handleRegister)
		public.Get("/forgot-password", s.handleForgotPasswordForm)
		public.Post("/forgot-password", s.handleForgotPassword)
		public.Get("/auth/callback", s.handleOAuthCallbackPage)
		public.Post("/auth/callback/session", s.handleOAuthCallbackSession)
		public.Get("/auth/reset", s.handleResetPasswordForm)
		public.Post("/auth/reset", s.handleResetPassword)
		public.Get("/auth/google", s.handleGoogleAuth)
		public.Post("/logout", s.handleLogout)
	})

	r.Group(func(protected chi.Router) {
		protected.Use(csrfMiddleware)
		protected.Use(s.requireUser)
		protected.Get("/dashboard", s.handleDashboard)
		protected.Get("/app/jobs", s.handleJobs)
		protected.Post("/app/conversions", s.handleCreateConversion)
		protected.Get("/app/conversions/{jobID}/download", s.handleDownloadConversion)
	})

	return r
}
