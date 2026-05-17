package ui

import (
	"bytes"
	"context"
	"html/template"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/quota"
)

func TestDashboardShowsDisabledSubmitWhenQuotaExhausted(t *testing.T) {
	component := Dashboard(
		"File Converter",
		&auth.User{ID: "user-1"},
		"csrf-token",
		"",
		nil,
		nil,
		25,
		quota.Summary{Limit: 3, ReservedCount: 1, CompletedCount: 2},
	)

	html := renderComponentHTML(t, component)
	if !strings.Contains(html, "Queue conversion</button>") {
		t.Fatalf("dashboard html missing submit button: %s", html)
	}
	if !strings.Contains(html, "button type=\"submit\" disabled") {
		t.Fatalf("dashboard html missing disabled submit state: %s", html)
	}
}

func TestLandingShowsGuestHeroFormAndQuotaCopy(t *testing.T) {
	component := Landing("File Converter", "csrf-token", LandingView{
		CSRFField:   template.HTML(`<input type="hidden" name="csrf_token" value="csrf-token">`),
		GuestQuota:  quota.Summary{Limit: 1},
		HeroMode:    LandingHeroModeGuest,
		MaxUploadMB: 25,
	})

	html := renderComponentHTML(t, component)
	if !strings.Contains(html, `action="/guest/conversions"`) {
		t.Fatalf("landing html missing guest form: %s", html)
	}
	if !strings.Contains(html, "1 file per hari") {
		t.Fatalf("landing html missing guest quota copy: %s", html)
	}
	if !strings.Contains(html, "Convert Sekarang") {
		t.Fatalf("landing html missing convert button: %s", html)
	}
}

func TestLandingShowsLoggedInHeroWithoutGuestForm(t *testing.T) {
	component := Landing("File Converter", "csrf-token", LandingView{
		CurrentUser: &auth.User{ID: "user-1"},
		HeroMode:    LandingHeroModeMember,
	})

	html := renderComponentHTML(t, component)
	if strings.Contains(html, `action="/guest/conversions"`) {
		t.Fatalf("landing html should not show guest form for member: %s", html)
	}
	if !strings.Contains(html, `href="/dashboard"`) {
		t.Fatalf("landing html missing dashboard CTA: %s", html)
	}
}

func TestLandingDisablesGuestSubmitWhenQuotaExhausted(t *testing.T) {
	component := Landing("File Converter", "csrf-token", LandingView{
		CSRFField:   template.HTML(`<input type="hidden" name="csrf_token" value="csrf-token">`),
		GuestQuota:  quota.Summary{Limit: 1, CompletedCount: 1},
		HeroMode:    LandingHeroModeGuest,
		MaxUploadMB: 25,
	})

	html := renderComponentHTML(t, component)
	if !strings.Contains(html, `button type="submit" class="hero-submit" disabled`) {
		t.Fatalf("landing html missing disabled guest submit: %s", html)
	}
}

func TestDashboardShowsDownloadActionForCompletedJobs(t *testing.T) {
	component := Dashboard(
		"File Converter",
		&auth.User{ID: "user-1"},
		"csrf-token",
		"",
		nil,
		[]Job{{
			ID:               "job-1",
			Status:           "done",
			TargetFormat:     "epub",
			SourceFileName:   "book.pdf",
			OutputStorageKey: stringPtr("outputs/123.epub"),
		}},
		25,
		quota.Summary{Limit: 3},
	)

	html := renderComponentHTML(t, component)
	if !strings.Contains(html, "/app/conversions/job-1/download") {
		t.Fatalf("dashboard html missing download link: %s", html)
	}
	if !strings.Contains(html, ">Download<") {
		t.Fatalf("dashboard html missing download label: %s", html)
	}
}

func renderComponentHTML(t *testing.T, component templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buf.String()
}

func stringPtr(value string) *string {
	return &value
}
