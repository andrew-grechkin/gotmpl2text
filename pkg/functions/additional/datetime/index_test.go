package datetime

import (
	"strings"
	"testing"
	"time"
)

func TestStrftimeTokens(t *testing.T) {
	// Fixed reference time: 2024-03-15 14:07:09 UTC (Friday, day 75 of the year)
	ref := time.Date(2024, 3, 15, 14, 7, 9, 0, time.UTC)

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"year 4-digit", "%Y", "2024"},
		{"year 2-digit", "%y", "24"},
		{"month numeric", "%m", "03"},
		{"month full name", "%B", "March"},
		{"month abbrev", "%b", "Mar"},
		{"day of month", "%d", "15"},
		{"day of year", "%j", "075"},
		{"hour 24", "%H", "14"},
		{"hour 12", "%I", "02"},
		{"minute", "%M", "07"},
		{"second", "%S", "09"},
		{"AM/PM", "%p", "PM"},
		{"weekday full", "%A", "Friday"},
		{"weekday abbrev", "%a", "Fri"},
		{"timezone name", "%Z", "UTC"},
		{"timezone offset", "%z", "+0000"},
		{"epoch seconds", "%s", "1710511629"},
		{"literal percent", "100%%", "100%"},
		{"combined ISO date", "%Y-%m-%d", "2024-03-15"},
		{"combined datetime", "%Y-%m-%d %H:%M:%S", "2024-03-15 14:07:09"},
		{"unknown token passed through", "%Q", "%Q"},
		{"trailing single percent kept", "abc%", "abc%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strftime(tt.format, ref)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStrftimeInputTypes(t *testing.T) {
	ref := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	epoch := ref.Unix()

	// time.Time
	if got, err := strftime("%Y-%m-%d", ref); err != nil || got != "2024-03-15" {
		t.Errorf("time.Time: got %q err %v", got, err)
	}
	// pointer to time.Time
	if got, err := strftime("%Y-%m-%d", &ref); err != nil || got != "2024-03-15" {
		t.Errorf("*time.Time: got %q err %v", got, err)
	}
	// int64 epoch
	if got, err := strftime("%Y-%m-%d", epoch); err != nil {
		t.Errorf("int64: unexpected error: %v", err)
	} else if !strings.HasPrefix(got, "2024-03-") {
		t.Errorf("int64: got %q, expected 2024-03-*", got)
	}
	// int epoch
	if _, err := strftime("%Y", int(epoch)); err != nil {
		t.Errorf("int: unexpected error: %v", err)
	}
	// RFC3339 string
	if got, err := strftime("%Y", "2024-03-15T14:07:09Z"); err != nil || got != "2024-03-15"[:4] {
		t.Errorf("RFC3339: got %q err %v", got, err)
	}
	// date-only string
	if got, err := strftime("%Y-%m-%d", "2024-03-15"); err != nil || got != "2024-03-15" {
		t.Errorf("date-only: got %q err %v", got, err)
	}
}

func TestStrftimeInvalidInputErrors(t *testing.T) {
	if _, err := strftime("%Y", "not-a-date"); err == nil {
		t.Error("expected error for unparseable string")
	}
	if _, err := strftime("%Y", []int{1, 2, 3}); err == nil {
		t.Error("expected error for unsupported type")
	}
}

// withFrozenNow pins nowFn to a fixed instant for the duration of the test, restoring the previous
// source on cleanup. Every earlier/later test uses it so results are exact and deterministic
func withFrozenNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := nowFn
	nowFn = func() time.Time { return at }
	t.Cleanup(func() { nowFn = prev })
}

func TestEarlierAndLater(t *testing.T) {
	base := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	withFrozenNow(t, base)

	tests := []struct {
		name string
		fn   func(...any) (time.Time, error)
		in   string
		want time.Time
	}{
		{"earlier seconds", earlier, "5s", base.Add(-5 * time.Second)},
		{"earlier minutes", earlier, "10m", base.Add(-10 * time.Minute)},
		{"earlier hours", earlier, "2h", base.Add(-2 * time.Hour)},
		{"earlier combined", earlier, "1h30m", base.Add(-90 * time.Minute)},
		{"earlier fractional", earlier, "1.5h", base.Add(-90 * time.Minute)},
		{"earlier sub-second", earlier, "500ms", base.Add(-500 * time.Millisecond)},
		{"earlier zero is now", earlier, "0s", base},
		{"later seconds", later, "5s", base.Add(5 * time.Second)},
		{"later hours", later, "2h", base.Add(2 * time.Hour)},
		{"later combined", later, "1h30m", base.Add(90 * time.Minute)},
		{"later zero is now", later, "0s", base},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// verifies the explicit-base overloads. Base can be supplied in any order relative to the duration,
// and can be any type toTime() accepts. nowFn is intentionally not frozen: these tests must not
// touch it - if any path falls back to now, the assertion fails
func TestEarlierAndLaterWithExplicitBase(t *testing.T) {
	// nowFn set to a poison value: if any code path falls back to it, tests explode loudly
	poison := time.Unix(1, 0)
	withFrozenNow(t, poison)

	baseTime := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	baseEpoch := baseTime.Unix()
	baseStr := "2024-03-15T14:00:00Z"
	baseDateOnly := "2024-03-15" // parses as 00:00:00 UTC

	tests := []struct {
		name string
		fn   func(...any) (time.Time, error)
		args []any
		want time.Time
	}{
		{"earlier: dur then base", earlier, []any{"1h", baseTime}, baseTime.Add(-time.Hour)},
		{"earlier: base then dur (order-independent)", earlier, []any{baseTime, "1h"}, baseTime.Add(-time.Hour)},
		{"earlier: base as int64 epoch", earlier, []any{"30m", baseEpoch}, baseTime.Add(-30 * time.Minute)},
		{"earlier: base as int epoch", earlier, []any{"30m", int(baseEpoch)}, baseTime.Add(-30 * time.Minute)},
		{"earlier: base as RFC3339 string", earlier, []any{"30m", baseStr}, baseTime.Add(-30 * time.Minute)},
		{"earlier: base as date-only string", earlier, []any{"1h", baseDateOnly}, time.Date(2024, 3, 14, 23, 0, 0, 0, time.UTC)},
		{"later: dur then base", later, []any{"1h", baseTime}, baseTime.Add(time.Hour)},
		{"later: base then dur", later, []any{baseTime, "1h"}, baseTime.Add(time.Hour)},
		{"later: base as RFC3339 string", later, []any{"30m", baseStr}, baseTime.Add(30 * time.Minute)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn(tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEarlierAndLaterArgErrors(t *testing.T) {
	base := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	withFrozenNow(t, base)

	tests := []struct {
		name    string
		fn      func(...any) (time.Time, error)
		args    []any
		errSubs string // substring the error must contain
	}{
		{"no args", earlier, []any{}, "expected 1 or 2 args"},
		{"three args", earlier, []any{"1h", base, base}, "expected 1 or 2 args"},
		{"only a time, no duration", earlier, []any{base}, "missing duration"},
		{"two durations, no base", earlier, []any{"1h", "30m"}, "duplicate duration"},
		{"two bases, no duration", earlier, []any{base, base}, "duplicate base-time"},
		{"unrecognized base type", earlier, []any{"1h", []int{1, 2}}, "unsupported time type"},
		{"unparseable base string", earlier, []any{"1h", "not a date and not a duration"}, "cannot parse time"},
		{"no args (later)", later, []any{}, "expected 1 or 2 args"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn(tt.args...)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errSubs) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errSubs)
			}
		})
	}
}

func TestEarlierAndLaterRejectNegative(t *testing.T) {
	base := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	withFrozenNow(t, base)

	if _, err := earlier("-5s"); err == nil {
		t.Error("earlier(-5s): expected error, got nil")
	} else if !strings.Contains(err.Error(), "non-negative") {
		t.Errorf("earlier(-5s): unexpected error message: %v", err)
	}
	if _, err := later("-1h"); err == nil {
		t.Error("later(-1h): expected error, got nil")
	} else if !strings.Contains(err.Error(), "non-negative") {
		t.Errorf("later(-1h): unexpected error message: %v", err)
	}
}

func TestEarlierAndLaterRejectUnparseable(t *testing.T) {
	// stdlib time.ParseDuration accepts only s/m/h/ms/us/µs/ns; anything else must be rejected loudly
	unparseable := []string{
		"1d",         // no days in stdlib
		"1w",         // no weeks
		"1hour",      // no verbose forms
		"5 sec",      // no spaces
		"yesterday",  // no natural language
		"5 hours",    // no verbose plural
		"",           // empty string
		"garbage",    // random text
	}
	for _, in := range unparseable {
		if _, err := earlier(in); err == nil {
			t.Errorf("earlier(%q): expected error, got nil", in)
		}
		if _, err := later(in); err == nil {
			t.Errorf("later(%q): expected error, got nil", in)
		}
	}
}
