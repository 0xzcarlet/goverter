package quota

import (
	"testing"
	"time"
)

func TestDateForTimeUsesJakartaCalendarDay(t *testing.T) {
	input := time.Date(2026, time.May, 14, 23, 30, 0, 0, time.UTC)
	got := DateForTime(input)

	if got.Location().String() != JakartaTimezone {
		t.Fatalf("location = %s, want %s", got.Location(), JakartaTimezone)
	}
	if got.Format("2006-01-02") != "2026-05-15" {
		t.Fatalf("date = %s, want 2026-05-15", got.Format("2006-01-02"))
	}
}

func TestSummaryRemaining(t *testing.T) {
	summary := Summary{
		Limit:          3,
		ReservedCount:  1,
		CompletedCount: 1,
	}

	if got := summary.ActiveCount(); got != 2 {
		t.Fatalf("ActiveCount() = %d, want 2", got)
	}
	if got := summary.Remaining(); got != 1 {
		t.Fatalf("Remaining() = %d, want 1", got)
	}
	if summary.Exhausted() {
		t.Fatal("Exhausted() = true, want false")
	}
}

func TestSummaryExhaustedFloorAtZero(t *testing.T) {
	summary := Summary{
		Limit:          3,
		ReservedCount:  2,
		CompletedCount: 2,
	}

	if got := summary.Remaining(); got != 0 {
		t.Fatalf("Remaining() = %d, want 0", got)
	}
	if !summary.Exhausted() {
		t.Fatal("Exhausted() = false, want true")
	}
}
