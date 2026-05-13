package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv               string
	AppName              string
	AppBaseURL           string
	AppHost              string
	AppPort              int
	AppCookieDomain      string
	AppSecureCookies     bool
	AppSessionCookieName string
	AppDataDir           string
	AppMaxUploadMB       int64
	AppMaxConcurrentJobs int
	AppWorkerPoll        time.Duration
	AppHTTPReadTimeout   time.Duration
	AppHTTPWriteTimeout  time.Duration
	AppHTTPIdleTimeout   time.Duration
	CSRFAuthKey          string

	SupabaseURL          string
	SupabaseAnonKey      string
	DatabaseURL          string
	MigrationDatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		AppName:              getEnv("APP_NAME", "File Converter"),
		AppBaseURL:           strings.TrimRight(getEnv("APP_BASE_URL", "http://localhost:8081"), "/"),
		AppHost:              getEnv("APP_HOST", "127.0.0.1"),
		AppPort:              getEnvInt("APP_PORT", 8081),
		AppCookieDomain:      os.Getenv("APP_COOKIE_DOMAIN"),
		AppSecureCookies:     getEnvBool("APP_SECURE_COOKIES", false),
		AppSessionCookieName: getEnv("APP_SESSION_COOKIE_NAME", "fc_session"),
		AppDataDir:           getEnv("APP_DATA_DIR", "./var/data"),
		AppMaxUploadMB:       getEnvInt64("APP_MAX_UPLOAD_MB", 25),
		AppMaxConcurrentJobs: getEnvInt("APP_MAX_CONCURRENT_JOBS", 1),
		AppWorkerPoll:        getEnvDuration("APP_WORKER_POLL_INTERVAL", 3*time.Second),
		AppHTTPReadTimeout:   getEnvDuration("APP_HTTP_READ_TIMEOUT", 30*time.Second),
		AppHTTPWriteTimeout:  getEnvDuration("APP_HTTP_WRITE_TIMEOUT", 300*time.Second),
		AppHTTPIdleTimeout:   getEnvDuration("APP_HTTP_IDLE_TIMEOUT", 60*time.Second),
		CSRFAuthKey:          os.Getenv("CSRF_AUTH_KEY"),
		SupabaseURL:          strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		SupabaseAnonKey:      os.Getenv("SUPABASE_ANON_KEY"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		MigrationDatabaseURL: os.Getenv("MIGRATION_DATABASE_URL"),
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	var missing []string
	for key, value := range map[string]string{
		"CSRF_AUTH_KEY":     c.CSRFAuthKey,
		"SUPABASE_URL":      c.SupabaseURL,
		"SUPABASE_ANON_KEY": c.SupabaseAnonKey,
		"DATABASE_URL":      c.DatabaseURL,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	if len(c.CSRFAuthKey) < 32 {
		return errors.New("CSRF_AUTH_KEY must be at least 32 bytes")
	}
	if c.AppMaxConcurrentJobs < 1 {
		return errors.New("APP_MAX_CONCURRENT_JOBS must be at least 1")
	}
	if c.AppMaxUploadMB < 1 {
		return errors.New("APP_MAX_UPLOAD_MB must be at least 1")
	}
	if _, err := net.ResolveTCPAddr("tcp", c.ListenAddr()); err != nil {
		return fmt.Errorf("invalid listen addr: %w", err)
	}
	return nil
}

func (c Config) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.AppHost, c.AppPort)
}

func (c Config) UploadDir() string {
	return filepath.Join(c.AppDataDir, "uploads")
}

func (c Config) OutputDir() string {
	return filepath.Join(c.AppDataDir, "outputs")
}

func (c Config) TempDir() string {
	return filepath.Join(c.AppDataDir, "tmp")
}

func (c Config) MaxUploadBytes() int64 {
	return c.AppMaxUploadMB * 1024 * 1024
}

func (c Config) SignupRedirectURL() string {
	return c.AppBaseURL + "/auth/callback"
}

func (c Config) OAuthCallbackURL() string {
	return c.AppBaseURL + "/auth/callback"
}

func (c Config) PasswordResetURL() string {
	return c.AppBaseURL + "/auth/reset"
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
