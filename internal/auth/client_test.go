package auth

import (
	"strings"
	"testing"
)

func TestGoogleAuthorizeURL(t *testing.T) {
	client := New("https://demo.supabase.co", "anon")
	got := client.GoogleAuthorizeURL("https://app.example.com/auth/callback", "challenge")
	if got == "" {
		t.Fatal("GoogleAuthorizeURL returned empty string")
	}
	if want := "provider=google"; !contains(got, want) {
		t.Fatalf("GoogleAuthorizeURL() missing %q: %s", want, got)
	}
	if want := "code_challenge=challenge"; !contains(got, want) {
		t.Fatalf("GoogleAuthorizeURL() missing %q: %s", want, got)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
