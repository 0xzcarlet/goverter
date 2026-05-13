package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	anonKey string
	http    *http.Client
}

type User struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	UserMetadata map[string]any         `json:"user_metadata"`
	AppMetadata  map[string]any         `json:"app_metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	ConfirmedAt  *time.Time             `json:"confirmed_at"`
	Identities   []map[string]any       `json:"identities"`
	Raw          map[string]interface{} `json:"-"`
}

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

type apiError struct {
	Code             int    `json:"code"`
	ErrorCode        string `json:"error_code"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"msg"`
}

func (e apiError) Error() string {
	if e.ErrorDescription != "" {
		return e.ErrorDescription
	}
	if e.Message != "" {
		return e.Message
	}
	if e.ErrorCode != "" {
		return e.ErrorCode
	}
	return "supabase auth request failed"
}

func New(baseURL, anonKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		anonKey: anonKey,
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) SignUp(ctx context.Context, email, password, redirectTo string) error {
	payload := map[string]any{
		"email":       email,
		"password":    password,
		"redirect_to": redirectTo,
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/auth/v1/signup", payload, "")
	return err
}

func (c *Client) SignIn(ctx context.Context, email, password string) (Session, error) {
	payload := map[string]any{
		"email":    email,
		"password": password,
	}
	var session Session
	resp, err := c.doJSON(ctx, http.MethodPost, "/auth/v1/token?grant_type=password", payload, "")
	if err != nil {
		return session, err
	}
	err = json.Unmarshal(resp, &session)
	return session, err
}

func (c *Client) Recover(ctx context.Context, email, redirectTo string) error {
	payload := map[string]any{
		"email":       email,
		"redirect_to": redirectTo,
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/auth/v1/recover", payload, "")
	return err
}

func (c *Client) RefreshSession(ctx context.Context, refreshToken string) (Session, error) {
	payload := map[string]any{
		"refresh_token": refreshToken,
	}
	var session Session
	resp, err := c.doJSON(ctx, http.MethodPost, "/auth/v1/token?grant_type=refresh_token", payload, "")
	if err != nil {
		return session, err
	}
	err = json.Unmarshal(resp, &session)
	return session, err
}

func (c *Client) User(ctx context.Context, accessToken string) (User, error) {
	var user User
	resp, err := c.doJSON(ctx, http.MethodGet, "/auth/v1/user", nil, accessToken)
	if err != nil {
		return user, err
	}
	err = json.Unmarshal(resp, &user)
	return user, err
}

func (c *Client) UpdatePassword(ctx context.Context, accessToken, password string) error {
	payload := map[string]any{
		"password": password,
	}
	_, err := c.doJSON(ctx, http.MethodPut, "/auth/v1/user", payload, accessToken)
	return err
}

func (c *Client) Logout(ctx context.Context, accessToken string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/auth/v1/logout", map[string]any{}, accessToken)
	return err
}

func (c *Client) GoogleAuthorizeURL(redirectTo, codeChallenge string) string {
	q := url.Values{}
	q.Set("provider", "google")
	q.Set("redirect_to", redirectTo)
	q.Set("scopes", "openid email profile")
	if codeChallenge != "" {
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}
	return c.baseURL + "/auth/v1/authorize?" + q.Encode()
}

func (c *Client) ExchangeOAuthCode(ctx context.Context, code, redirectTo, codeVerifier string) (Session, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectTo)
	form.Set("code_verifier", codeVerifier)

	var session Session
	resp, err := c.doForm(ctx, http.MethodPost, "/auth/v1/oauth/token", form)
	if err != nil {
		return session, err
	}
	err = json.Unmarshal(resp, &session)
	return session, err
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, bearer string) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, err
		}
		body = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", c.anonKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return data, nil
	}

	var apiErr apiError
	if err := json.Unmarshal(data, &apiErr); err == nil && (apiErr.ErrorDescription != "" || apiErr.Message != "" || apiErr.ErrorCode != "") {
		return nil, apiErr
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized")
	}
	return nil, fmt.Errorf("supabase auth: %s", strings.TrimSpace(string(data)))
}

func (c *Client) doForm(ctx context.Context, method, path string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("apikey", c.anonKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return data, nil
	}
	var apiErr apiError
	if err := json.Unmarshal(data, &apiErr); err == nil && (apiErr.ErrorDescription != "" || apiErr.Message != "" || apiErr.ErrorCode != "") {
		return nil, apiErr
	}
	return nil, fmt.Errorf("supabase auth: %s", strings.TrimSpace(string(data)))
}
