package config

import "testing"

func TestMaxUploadBytes(t *testing.T) {
	cfg := Config{AppMaxUploadMB: 25}
	if got, want := cfg.MaxUploadBytes(), int64(25*1024*1024); got != want {
		t.Fatalf("MaxUploadBytes() = %d, want %d", got, want)
	}
}
