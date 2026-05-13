package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/config"
	"github.com/alexanderzull/file-converter/internal/converter"
	"github.com/alexanderzull/file-converter/internal/db"
	"github.com/alexanderzull/file-converter/internal/quota"
	"github.com/alexanderzull/file-converter/internal/storage"
	"github.com/alexanderzull/file-converter/internal/ui"
)

type repository interface {
	Ping(ctx context.Context) error
	ListJobs(ctx context.Context, userID string, limit int) ([]db.ConversionJob, error)
	DailyUsageSummary(ctx context.Context, userID string, quotaDate time.Time) (db.DailyUsage, error)
	CreateQueuedConversion(ctx context.Context, params db.CreateQueuedConversionParams) (db.ConversionJob, error)
	FetchDownloadableFile(ctx context.Context, userID, jobID string) (db.DownloadableFile, error)
}

type fileStorage interface {
	SaveUpload(ctx context.Context, originalName string, source io.Reader) (storage.SavedFile, error)
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

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	component := ui.Landing(s.cfg.AppName, s.optionalCurrentUser(r.Context()))
	s.render(w, r, http.StatusOK, component)
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, ui.Login(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), nil, ""))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	session, err := s.auth.SignIn(r.Context(), email, password)
	if err != nil {
		s.render(w, r, http.StatusUnauthorized, ui.Login(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: err.Error()}, email))
		return
	}
	auth.WriteCookieSession(w, s.cfg, session)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleRegisterForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, ui.Register(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), nil, ""))
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	err := s.auth.SignUp(r.Context(), email, password, s.cfg.SignupRedirectURL())
	if err != nil {
		s.render(w, r, http.StatusBadRequest, ui.Register(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: err.Error()}, email))
		return
	}
	s.render(w, r, http.StatusAccepted, ui.Register(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "success", Message: "Registration received. Check your email if confirmation is enabled."}, email))
}

func (s *Server) handleForgotPasswordForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, ui.ForgotPassword(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), nil, ""))
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	err := s.auth.Recover(r.Context(), email, s.cfg.PasswordResetURL())
	if err != nil {
		s.render(w, r, http.StatusBadRequest, ui.ForgotPassword(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: err.Error()}, email))
		return
	}
	s.render(w, r, http.StatusAccepted, ui.ForgotPassword(s.cfg.AppName, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "success", Message: "Recovery email sent if the account exists."}, email))
}

func (s *Server) handleResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	user, session, refreshed, err := s.resolveUserSession(r)
	ok := err == nil
	if refreshed {
		auth.WriteCookieSession(w, s.cfg, auth.Session{
			AccessToken:  session.AccessToken,
			RefreshToken: session.RefreshToken,
			ExpiresIn:    int(time.Until(session.ExpiresAt).Seconds()),
			User:         user,
		})
	}
	component := ui.ResetPassword(s.cfg.AppName, optionalUser(user, ok), csrf.Token(r), csrf.TemplateField(r), nil, ok)
	s.render(w, r, http.StatusOK, component)
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	user, session, refreshed, err := s.resolveUserSession(r)
	if err != nil {
		s.render(w, r, http.StatusUnauthorized, ui.ResetPassword(s.cfg.AppName, nil, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: "Recovery session is missing. Open the reset link from your email again."}, false))
		return
	}
	if refreshed {
		auth.WriteCookieSession(w, s.cfg, auth.Session{
			AccessToken:  session.AccessToken,
			RefreshToken: session.RefreshToken,
			ExpiresIn:    int(time.Until(session.ExpiresAt).Seconds()),
			User:         user,
		})
	}
	if session.AccessToken == "" {
		s.render(w, r, http.StatusUnauthorized, ui.ResetPassword(s.cfg.AppName, &user, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: "Access token is missing."}, true))
		return
	}

	_ = r.ParseForm()
	password := r.FormValue("password")
	if len(password) < 8 {
		s.render(w, r, http.StatusBadRequest, ui.ResetPassword(s.cfg.AppName, &user, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: "Password must be at least 8 characters."}, true))
		return
	}
	if err := s.auth.UpdatePassword(r.Context(), session.AccessToken, password); err != nil {
		s.render(w, r, http.StatusBadRequest, ui.ResetPassword(s.cfg.AppName, &user, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: err.Error()}, true))
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	verifier, challenge, err := newPKCEPair()
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	c := &http.Cookie{
		Name:     s.cfg.AppSessionCookieName + "_pkce_verifier",
		Value:    verifier,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   s.cfg.AppSecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	}
	if s.cfg.AppCookieDomain != "" {
		c.Domain = s.cfg.AppCookieDomain
	}
	http.SetCookie(w, c)
	http.Redirect(w, r, s.auth.GoogleAuthorizeURL(s.cfg.OAuthCallbackURL(), challenge), http.StatusTemporaryRedirect)
}

func (s *Server) handleOAuthCallbackPage(w http.ResponseWriter, r *http.Request) {
	if code := strings.TrimSpace(r.URL.Query().Get("code")); code != "" {
		verifierCookie, err := r.Cookie(s.cfg.AppSessionCookieName + "_pkce_verifier")
		if err != nil || verifierCookie.Value == "" {
			s.render(w, r, http.StatusBadRequest, ui.Callback(s.cfg.AppName, csrf.Token(r), "Google sign-in callback", "Missing PKCE verifier cookie. Start the sign-in flow again from the app.", "/login"))
			return
		}
		session, err := s.auth.ExchangeOAuthCode(r.Context(), code, s.cfg.OAuthCallbackURL(), verifierCookie.Value)
		clearPKCECookie(w, s.cfg)
		if err != nil {
			s.render(w, r, http.StatusUnauthorized, ui.Callback(s.cfg.AppName, csrf.Token(r), "Google sign-in callback", err.Error(), "/login"))
			return
		}
		auth.WriteCookieSession(w, s.cfg, session)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	s.render(w, r, http.StatusOK, ui.Callback(s.cfg.AppName, csrf.Token(r), "Google sign-in callback", "The browser should receive the session fragment and hand it to the Go backend.", "/dashboard"))
}

func (s *Server) handleOAuthCallbackSession(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeJSONError(w, http.StatusBadRequest, "invalid callback payload")
		return
	}
	user, err := s.auth.User(r.Context(), payload.AccessToken)
	if err != nil {
		s.writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}
	auth.WriteCookieSession(w, s.cfg, auth.Session{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		ExpiresIn:    payload.ExpiresIn,
		User:         user,
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	session := auth.ReadCookieSession(r, s.cfg)
	if session.AccessToken != "" {
		_ = s.auth.Logout(r.Context(), session.AccessToken)
	}
	auth.ClearCookieSession(w, s.cfg)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	jobs, usage, err := s.dashboardData(r.Context(), user.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, ui.Dashboard(s.cfg.AppName, &user, csrf.Token(r), csrf.TemplateField(r), nil, jobs, s.cfg.AppMaxUploadMB, usage))
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	jobs, usage, err := s.dashboardData(r.Context(), user.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, ui.JobsPanel(jobs, usage))
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

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.repo.Ping(ctx); err != nil {
		s.writeJSONError(w, http.StatusServiceUnavailable, fmt.Sprintf("database not ready: %v", err))
		return
	}
	if err := s.converter.Check(); err != nil {
		s.writeJSONError(w, http.StatusServiceUnavailable, fmt.Sprintf("ebook-convert not ready: %v", err))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, session, refreshed, err := s.resolveUserSession(r)
		if err != nil {
			auth.ClearCookieSession(w, s.cfg)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if refreshed {
			auth.WriteCookieSession(w, s.cfg, auth.Session{
				AccessToken:  session.AccessToken,
				RefreshToken: session.RefreshToken,
				ExpiresIn:    int(time.Until(session.ExpiresAt).Seconds()),
				User:         user,
			})
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

func (s *Server) optionalCurrentUser(ctx context.Context) *auth.User {
	user, ok := CurrentUser(ctx)
	if !ok {
		return nil
	}
	return &user
}

func (s *Server) resolveUserSession(r *http.Request) (auth.User, auth.CookieSession, bool, error) {
	session := auth.ReadCookieSession(r, s.cfg)
	if session.AccessToken == "" {
		return auth.User{}, session, false, errors.New("missing access token")
	}
	user, err := s.auth.User(r.Context(), session.AccessToken)
	if err == nil {
		return user, session, false, nil
	}
	if session.RefreshToken == "" {
		return auth.User{}, session, false, err
	}
	refreshed, refreshErr := s.auth.RefreshSession(r.Context(), session.RefreshToken)
	if refreshErr != nil {
		return auth.User{}, session, false, refreshErr
	}
	return refreshed.User, auth.CookieSession{
		AccessToken:  refreshed.AccessToken,
		RefreshToken: refreshed.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(refreshed.ExpiresIn) * time.Second),
	}, true, nil
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	templ.Handler(
		component,
		templ.WithStatus(status),
		templ.WithContentType("text/html; charset=utf-8"),
	).ServeHTTP(w, r)
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("internal error", "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func (s *Server) renderDashboardError(w http.ResponseWriter, r *http.Request, user auth.User, message string) {
	jobs, usage, err := s.dashboardData(r.Context(), user.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.render(w, r, http.StatusBadRequest, ui.Dashboard(s.cfg.AppName, &user, csrf.Token(r), csrf.TemplateField(r), &ui.Flash{Kind: "error", Message: message}, jobs, s.cfg.AppMaxUploadMB, usage))
}

func (s *Server) writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.log.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start).String(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self' "+s.cfg.SupabaseURL+"; img-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func normalizeFormat(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), ".")
}

func detectMimeType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		return "application/pdf"
	case ".epub":
		return "application/epub+zip"
	default:
		return "application/octet-stream"
	}
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

func buildDownloadFilename(sourceName, targetFormat string) string {
	base := strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
	base = sanitizeFilename(base)
	if base == "" {
		base = "converted-file"
	}
	target := strings.TrimPrefix(strings.ToLower(targetFormat), ".")
	if target == "" {
		target = "bin"
	}
	return base + "." + target
}

func sanitizeFilename(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", "\x00", "", "\n", " ", "\r", " ", "\"", "", "'", "")
	safe := strings.TrimSpace(replacer.Replace(value))
	safe = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case strings.ContainsRune(" ._-()", r):
			return r
		default:
			return '-'
		}
	}, safe)
	safe = strings.Trim(safe, ". -_")
	return safe
}

func optionalUser(user auth.User, ok bool) *auth.User {
	if !ok {
		return nil
	}
	return &user
}

func newPKCEPair() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func clearPKCECookie(w http.ResponseWriter, cfg config.Config) {
	c := &http.Cookie{
		Name:     cfg.AppSessionCookieName + "_pkce_verifier",
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   cfg.AppSecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	if cfg.AppCookieDomain != "" {
		c.Domain = cfg.AppCookieDomain
	}
	http.SetCookie(w, c)
}

func csrfTrustedOrigins(cfg config.Config) []string {
	seen := map[string]struct{}{}
	add := func(origin string) {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			return
		}
		seen[origin] = struct{}{}
	}

	add("localhost:8081")
	add("127.0.0.1:8081")

	if parsed, err := url.Parse(cfg.AppBaseURL); err == nil && parsed.Host != "" {
		add(parsed.Host)
	}

	out := make([]string, 0, len(seen))
	for origin := range seen {
		out = append(out, origin)
	}
	return out
}
