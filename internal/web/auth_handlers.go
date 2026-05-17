package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/csrf"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/ui"
)

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	user := s.landingCurrentUser(w, r)
	component, err := s.landingComponent(w, r, user, nil)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
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

func optionalUser(user auth.User, ok bool) *auth.User {
	if !ok {
		return nil
	}
	return &user
}
