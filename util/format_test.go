package util_test

import (
	"strings"
	"testing"
	"time"

	"timetracker/util"
)

// ─── FormatDuration ───────────────────────────────────────────────────────────

func TestFormatDuration_Seconds(t *testing.T) {
	got := util.FormatDuration(45 * time.Second)
	if got != "45s" {
		t.Errorf("FormatDuration(45s) = %q, want %q", got, "45s")
	}
}

func TestFormatDuration_MinutesAndSeconds(t *testing.T) {
	got := util.FormatDuration(2*time.Minute + 5*time.Second)
	if got != "2m 05s" {
		t.Errorf("FormatDuration(2m5s) = %q, want %q", got, "2m 05s")
	}
}

func TestFormatDuration_HoursMinutesSeconds(t *testing.T) {
	got := util.FormatDuration(1*time.Hour + 30*time.Minute + 7*time.Second)
	if got != "1h 30m 07s" {
		t.Errorf("FormatDuration(1h30m7s) = %q, want %q", got, "1h 30m 07s")
	}
}

func TestFormatDuration_Zero(t *testing.T) {
	got := util.FormatDuration(0)
	if got != "0s" {
		t.Errorf("FormatDuration(0) = %q, want %q", got, "0s")
	}
}

func TestFormatDuration_RoundsToSecond(t *testing.T) {
	// 500ms rounds to 1s
	got := util.FormatDuration(500 * time.Millisecond)
	if got != "1s" {
		t.Errorf("FormatDuration(500ms) = %q, want %q", got, "1s")
	}
}

// ─── FormatDurationShort ──────────────────────────────────────────────────────

func TestFormatDurationShort_Minutes(t *testing.T) {
	got := util.FormatDurationShort(45 * time.Minute)
	if got != "45m" {
		t.Errorf("FormatDurationShort(45m) = %q, want %q", got, "45m")
	}
}

func TestFormatDurationShort_HoursMinutes(t *testing.T) {
	got := util.FormatDurationShort(2*time.Hour + 5*time.Minute)
	if got != "2h 05m" {
		t.Errorf("FormatDurationShort(2h5m) = %q, want %q", got, "2h 05m")
	}
}

func TestFormatDurationShort_SubMinuteRoundsUp(t *testing.T) {
	// 30s is exactly halfway → rounds up to 1m (Go's time.Round ties round to even/up)
	got := util.FormatDurationShort(30 * time.Second)
	if got != "1m" {
		t.Errorf("FormatDurationShort(30s) = %q, want %q", got, "1m")
	}
}

func TestFormatDurationShort_BelowHalfMinuteRoundsToZero(t *testing.T) {
	got := util.FormatDurationShort(29 * time.Second)
	if got != "0m" {
		t.Errorf("FormatDurationShort(29s) = %q, want %q", got, "0m")
	}
}

// ─── FormatDurationHours ──────────────────────────────────────────────────────

func TestFormatDurationHours_OneDecimal(t *testing.T) {
	got := util.FormatDurationHours(90 * time.Minute)
	if got != "1.5h" {
		t.Errorf("FormatDurationHours(90m) = %q, want %q", got, "1.5h")
	}
}

func TestFormatDurationHours_Zero(t *testing.T) {
	got := util.FormatDurationHours(0)
	if got != "0.0h" {
		t.Errorf("FormatDurationHours(0) = %q, want %q", got, "0.0h")
	}
}

// ─── Bar ──────────────────────────────────────────────────────────────────────

func TestBar_Full(t *testing.T) {
	got := util.Bar(1.0, 10)
	if got != "██████████" {
		t.Errorf("Bar(1.0, 10) = %q, want all filled", got)
	}
}

func TestBar_Empty(t *testing.T) {
	got := util.Bar(0.0, 10)
	if got != "░░░░░░░░░░" {
		t.Errorf("Bar(0.0, 10) = %q, want all empty", got)
	}
}

func TestBar_Half(t *testing.T) {
	got := util.Bar(0.5, 10)
	if got != "█████░░░░░" {
		t.Errorf("Bar(0.5, 10) = %q, want half filled", got)
	}
}

func TestBar_TotalLengthAlwaysWidth(t *testing.T) {
	for _, frac := range []float64{0, 0.1, 0.33, 0.5, 0.75, 1.0} {
		got := util.Bar(frac, 20)
		// Count runes (characters), not bytes
		n := len([]rune(got))
		if n != 20 {
			t.Errorf("Bar(%.2f, 20) has length %d, want 20", frac, n)
		}
	}
}

func TestBar_ClampsAboveOne(t *testing.T) {
	got := util.Bar(2.0, 10)
	if strings.Contains(got, "░") {
		t.Errorf("Bar(2.0, 10) should be fully filled, got %q", got)
	}
}

func TestBar_ClampsBelowZero(t *testing.T) {
	got := util.Bar(-1.0, 10)
	if strings.Contains(got, "█") {
		t.Errorf("Bar(-1.0, 10) should be fully empty, got %q", got)
	}
}

// ─── Truncate ─────────────────────────────────────────────────────────────────

func TestTruncate_ShortStringUnchanged(t *testing.T) {
	got := util.Truncate("hello", 10)
	if got != "hello" {
		t.Errorf("Truncate short = %q, want %q", got, "hello")
	}
}

func TestTruncate_ExactLengthUnchanged(t *testing.T) {
	got := util.Truncate("hello", 5)
	if got != "hello" {
		t.Errorf("Truncate exact = %q, want %q", got, "hello")
	}
}

func TestTruncate_LongStringTruncated(t *testing.T) {
	got := util.Truncate("hello world", 8)
	if got != "hello w…" {
		t.Errorf("Truncate long = %q, want %q", got, "hello w…")
	}
}

func TestTruncate_ResultFitsMax(t *testing.T) {
	input := "this is a very long task description that exceeds max"
	max := 20
	got := util.Truncate(input, max)
	if len(got) > max+2 { // +2 for multi-byte ellipsis
		t.Errorf("Truncate result length %d > max %d: %q", len(got), max, got)
	}
}
