package config

import "testing"

func TestMaxUploadBytes(t *testing.T) {
	cfg := Config{AppMaxUploadMB: 25}
	if got, want := cfg.MaxUploadBytes(), int64(25*1024*1024); got != want {
		t.Fatalf("MaxUploadBytes() = %d, want %d", got, want)
	}
}

func TestValidateRequiresDailyConversionLimit(t *testing.T) {
	cfg := Config{
		AppHost:                 "127.0.0.1",
		AppPort:                 8081,
		AppMaxUploadMB:          25,
		AppMaxConcurrentJobs:    1,
		AppDailyConversionLimit: 0,
		CSRFAuthKey:             "12345678901234567890123456789012",
		SupabaseURL:             "https://example.supabase.co",
		SupabaseAnonKey:         "anon",
		DatabaseURL:             "postgres://example",
	}
	if err := cfg.Validate(); err == nil || err.Error() != "APP_DAILY_CONVERSION_LIMIT must be at least 1" {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestGetRequiredEnvInt(t *testing.T) {
	t.Setenv("APP_DAILY_CONVERSION_LIMIT", "5")
	if got := getRequiredEnvInt("APP_DAILY_CONVERSION_LIMIT"); got != 5 {
		t.Fatalf("getRequiredEnvInt() = %d, want 5", got)
	}
}

func TestGetRequiredEnvIntMissingOrInvalid(t *testing.T) {
	t.Setenv("APP_DAILY_CONVERSION_LIMIT", "")
	if got := getRequiredEnvInt("APP_DAILY_CONVERSION_LIMIT"); got != 0 {
		t.Fatalf("missing env = %d, want 0", got)
	}

	t.Setenv("APP_DAILY_CONVERSION_LIMIT", "abc")
	if got := getRequiredEnvInt("APP_DAILY_CONVERSION_LIMIT"); got != 0 {
		t.Fatalf("invalid env = %d, want 0", got)
	}
}
