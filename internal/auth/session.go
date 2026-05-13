package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alexanderzull/file-converter/internal/config"
)

type CookieSession struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func ReadCookieSession(r *http.Request, cfg config.Config) CookieSession {
	return CookieSession{
		AccessToken:  readCookie(r, cfg.AppSessionCookieName+"_access"),
		RefreshToken: readCookie(r, cfg.AppSessionCookieName+"_refresh"),
		ExpiresAt:    readExpiry(r, cfg.AppSessionCookieName+"_expires"),
	}
}

func WriteCookieSession(w http.ResponseWriter, cfg config.Config, session Session) {
	expiresAt := time.Now().Add(time.Duration(session.ExpiresIn) * time.Second)
	writeCookie(w, cfg, cfg.AppSessionCookieName+"_access", session.AccessToken, expiresAt, true)
	writeCookie(w, cfg, cfg.AppSessionCookieName+"_refresh", session.RefreshToken, expiresAt.Add(30*24*time.Hour), true)
	writeCookie(w, cfg, cfg.AppSessionCookieName+"_expires", strconv.FormatInt(expiresAt.Unix(), 10), expiresAt, true)
}

func ClearCookieSession(w http.ResponseWriter, cfg config.Config) {
	expired := time.Unix(0, 0)
	writeCookie(w, cfg, cfg.AppSessionCookieName+"_access", "", expired, true)
	writeCookie(w, cfg, cfg.AppSessionCookieName+"_refresh", "", expired, true)
	writeCookie(w, cfg, cfg.AppSessionCookieName+"_expires", "", expired, true)
}

func writeCookie(w http.ResponseWriter, cfg config.Config, name, value string, expires time.Time, httpOnly bool) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: httpOnly,
		Secure:   cfg.AppSecureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   -1,
	}
	if value != "" {
		c.MaxAge = int(time.Until(expires).Seconds())
		if c.MaxAge < 0 {
			c.MaxAge = 0
		}
	}
	if cfg.AppCookieDomain != "" {
		c.Domain = cfg.AppCookieDomain
	}
	http.SetCookie(w, c)
}

func readCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func readExpiry(r *http.Request, name string) time.Time {
	raw := readCookie(r, name)
	if raw == "" {
		return time.Time{}
	}
	unix, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}
