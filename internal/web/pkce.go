package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/alexanderzull/file-converter/internal/config"
)

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
