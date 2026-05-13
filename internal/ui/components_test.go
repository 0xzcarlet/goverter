package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/db"
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

func TestDashboardShowsDownloadActionForCompletedJobs(t *testing.T) {
	component := Dashboard(
		"File Converter",
		&auth.User{ID: "user-1"},
		"csrf-token",
		"",
		nil,
		[]db.ConversionJob{{
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
