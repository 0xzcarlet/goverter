package web

import (
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/converter"
	"github.com/alexanderzull/file-converter/internal/db"
	"github.com/alexanderzull/file-converter/internal/quota"
	"github.com/alexanderzull/file-converter/internal/ui"
)

const guestDailyConversionLimit = 1

func (s *Server) handleGuestConversion(w http.ResponseWriter, r *http.Request) {
	if currentUser := s.landingCurrentUser(w, r); currentUser != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	guestToken := s.ensureGuestToken(w, r)
	quotaDate := quota.DateForTime(time.Now())

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes())
	if err := r.ParseMultipartForm(s.cfg.MaxUploadBytes()); err != nil {
		s.renderLandingWithGuestToken(w, r, http.StatusBadRequest, guestToken, &ui.Flash{Kind: "error", Message: "Upload gagal dibaca. Pastikan ukuran file tidak melewati batas yang diizinkan."})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.renderLandingWithGuestToken(w, r, http.StatusBadRequest, guestToken, &ui.Flash{Kind: "error", Message: "Pilih file PDF atau EPUB terlebih dulu."})
		return
	}
	defer file.Close()

	targetFormat := normalizeFormat(r.FormValue("target_format"))
	if err := converter.ValidatePair(header.Filename, targetFormat); err != nil {
		s.renderLandingWithGuestToken(w, r, http.StatusBadRequest, guestToken, &ui.Flash{Kind: "error", Message: err.Error()})
		return
	}

	if err := s.repo.ReserveGuestDailySlot(r.Context(), guestToken, quotaDate, guestDailyConversionLimit); err != nil {
		if err == db.ErrDailyLimitReached {
			s.renderLandingWithGuestToken(w, r, http.StatusTooManyRequests, guestToken, &ui.Flash{Kind: "error", Message: "Jatah guest untuk hari ini sudah habis. Login untuk lanjut dengan dashboard dan quota penuh."})
			return
		}
		s.internalError(w, r, err)
		return
	}

	reserved := true
	refund := func() {
		if reserved {
			_ = s.repo.RefundGuestDailySlot(r.Context(), guestToken, quotaDate)
		}
	}

	saved, err := s.storage.SaveUpload(r.Context(), header.Filename, file)
	if err != nil {
		refund()
		s.renderLandingWithGuestToken(w, r, http.StatusInternalServerError, guestToken, &ui.Flash{Kind: "error", Message: "File tidak bisa disimpan untuk diproses. Coba lagi sebentar lagi."})
		return
	}
	defer func() { _ = s.storage.Remove(saved.StorageKey) }()

	output, err := s.storage.PrepareOutputPath(targetFormat)
	if err != nil {
		refund()
		s.renderLandingWithGuestToken(w, r, http.StatusInternalServerError, guestToken, &ui.Flash{Kind: "error", Message: "Output convert belum bisa disiapkan. Coba ulangi beberapa saat lagi."})
		return
	}
	defer func() { _ = s.storage.Remove(output.StorageKey) }()

	if err := s.converter.Convert(r.Context(), s.storage.AbsPath(saved.StorageKey), output.AbsPath); err != nil {
		refund()
		s.renderLandingWithGuestToken(w, r, http.StatusInternalServerError, guestToken, &ui.Flash{Kind: "error", Message: "Konversi gagal diproses. Silakan coba lagi atau login untuk memakai dashboard."})
		return
	}

	if err := s.repo.CompleteGuestDailySlot(r.Context(), guestToken, quotaDate); err != nil {
		refund()
		s.internalError(w, r, err)
		return
	}
	reserved = false

	handle, err := os.Open(output.AbsPath)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer handle.Close()

	info, err := handle.Stat()
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	filename := buildDownloadFilename(header.Filename, targetFormat)
	w.Header().Set("Content-Type", converter.MIMEByFormat(targetFormat))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	http.ServeContent(w, r, filename, info.ModTime(), handle)
}

func (s *Server) landingCurrentUser(w http.ResponseWriter, r *http.Request) *auth.User {
	if user, ok := CurrentUser(r.Context()); ok {
		return &user
	}
	if s.auth == nil {
		return nil
	}
	session := auth.ReadCookieSession(r, s.cfg)
	if session.AccessToken == "" {
		return nil
	}

	user, refreshedSession, refreshed, err := s.resolveUserSession(r)
	if err != nil {
		auth.ClearCookieSession(w, s.cfg)
		return nil
	}
	if refreshed {
		auth.WriteCookieSession(w, s.cfg, auth.Session{
			AccessToken:  refreshedSession.AccessToken,
			RefreshToken: refreshedSession.RefreshToken,
			ExpiresIn:    int(time.Until(refreshedSession.ExpiresAt).Seconds()),
			User:         user,
		})
	}
	return &user
}

func (s *Server) ensureGuestToken(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(s.cfg.AppSessionCookieName + "_guest"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	guestToken := uuid.NewString()
	cookie := &http.Cookie{
		Name:     s.cfg.AppSessionCookieName + "_guest",
		Value:    guestToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.AppSecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   365 * 24 * 60 * 60,
	}
	if s.cfg.AppCookieDomain != "" {
		cookie.Domain = s.cfg.AppCookieDomain
	}
	http.SetCookie(w, cookie)
	return guestToken
}

func (s *Server) landingComponent(w http.ResponseWriter, r *http.Request, currentUser *auth.User, flash *ui.Flash) (templ.Component, error) {
	guestToken := ""
	if currentUser == nil {
		guestToken = s.ensureGuestToken(w, r)
	}
	return s.buildLandingComponent(r, currentUser, flash, guestToken)
}

func (s *Server) renderLandingWithGuestToken(w http.ResponseWriter, r *http.Request, status int, guestToken string, flash *ui.Flash) {
	component, err := s.buildLandingComponent(r, nil, flash, guestToken)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.render(w, r, status, component)
}

func (s *Server) buildLandingComponent(r *http.Request, currentUser *auth.User, flash *ui.Flash, guestToken string) (templ.Component, error) {
	heroMode := ui.LandingHeroModeMember
	view := ui.LandingView{
		Flash:       flash,
		CSRFField:   csrf.TemplateField(r),
		CurrentUser: currentUser,
		HeroMode:    heroMode,
		MaxUploadMB: s.cfg.AppMaxUploadMB,
	}

	if currentUser == nil {
		heroMode = ui.LandingHeroModeGuest
		usage, err := s.repo.GuestDailyUsageSummary(r.Context(), guestToken, quota.DateForTime(time.Now()))
		if err != nil {
			return nil, err
		}
		view.GuestQuota = quota.Summary{
			Limit:          guestDailyConversionLimit,
			ReservedCount:  usage.ReservedCount,
			CompletedCount: usage.CompletedCount,
		}
		view.HeroMode = heroMode
	}

	return ui.Landing(s.cfg.AppName, csrf.Token(r), view), nil
}
