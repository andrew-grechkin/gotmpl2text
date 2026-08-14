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
