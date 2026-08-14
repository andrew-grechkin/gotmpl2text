package dict

import (
	"reflect"
	"strings"
	"testing"
)

var nested = map[string]any{
	"server": map[string]any{
		"host": "localhost",
		"port": 8080,
		"tls": map[string]any{
			"enabled": true,
		},
	},
	"empty":     "",
	"nilvalue":  nil,
	"anyKeyMap": map[any]any{"key": "value"},
}

func TestValueAt(t *testing.T) {
	tests := []struct {
		name string
		path string
		args []any
		want any
	}{
		{"single segment", "server", []any{nested}, nested["server"]},
		{"two segments", "server.host", []any{nested}, "localhost"},
		{"three segments", "server.tls.enabled", []any{nested}, true},
		{"empty string value returned as-is", "empty", []any{nested}, ""},
		{"present-but-nil returns nil even with default", "nilvalue", []any{"fallback", nested}, nil},
		{"missing top-level, no default", "missing", []any{nested}, nil},
		{"missing top-level, with default", "missing", []any{"fallback", nested}, "fallback"},
		{"missing mid-level, with default", "server.missing", []any{"fallback", nested}, "fallback"},
		{"missing deep, with default", "server.tls.missing", []any{"fallback", nested}, "fallback"},
		{"intermediate is scalar, with default", "server.host.deeper", []any{"fallback", nested}, "fallback"},
		{"map[any]any support", "anyKeyMap.key", []any{nested}, "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueAt(tt.path, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("valueAt(%q, %v) = %v, want %v", tt.path, tt.args, got, tt.want)
			}
		})
	}
}

func TestValueAtWrongArgCountErrors(t *testing.T) {
	// Both under- and over-count are template author mistakes; must be loud
	cases := [][]any{
		{},                      // no data - only path
		{nil, nil, nil},         // too many
		{"a", "b", "c", nested}, // way too many
	}
	for _, args := range cases {
		if _, err := valueAt("a.b", args...); err == nil {
			t.Errorf("expected error for args=%v, got nil", args)
		} else if !strings.Contains(err.Error(), "valueAt:") {
			t.Errorf("error should be prefixed with 'valueAt:', got: %v", err)
		}
	}
}

func TestExistsAt(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"top-level exists", "server", true},
		{"deep exists", "server.tls.enabled", true},
		{"empty string exists", "empty", true},
		{"nil value exists (present-but-nil)", "nilvalue", true},
		{"map[any]any traversal", "anyKeyMap.key", true},
		{"missing top-level", "missing", false},
		{"missing mid-level", "server.missing", false},
		{"missing deep", "server.tls.missing", false},
		{"intermediate is scalar", "server.host.deeper", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := existsAt(tt.path, nested); got != tt.want {
				t.Errorf("existsAt(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestToEntries(t *testing.T) {
	// Sorted-key invariant matters - Go map iteration is randomized
	got, err := toEntries(map[string]any{"b": 2, "a": 1, "c": 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []map[string]any{
		{"key": "a", "value": 1},
		{"key": "b", "value": 2},
		{"key": "c", "value": 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestToEntriesEmptyMap(t *testing.T) {
	got, err := toEntries(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestToEntriesPreservesNestedValues(t *testing.T) {
	got, err := toEntries(map[string]any{
		"nested": map[string]any{"deep": 42},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []map[string]any{
		{"key": "nested", "value": map[string]any{"deep": 42}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestToEntriesRejectsNonMap(t *testing.T) {
	rejects := []any{
		nil,
		42,
		"string",
		[]int{1, 2},
		true,
	}
	for _, v := range rejects {
		if _, err := toEntries(v); err == nil {
			t.Errorf("expected error for %T, got nil", v)
		} else if !strings.Contains(err.Error(), "toEntries:") {
			t.Errorf("error should be prefixed with 'to_entries:', got: %v", err)
		}
	}
}

func TestFromEntries(t *testing.T) {
	in := []map[string]any{
		{"key": "a", "value": 1},
		{"key": "b", "value": 2},
		{"key": "c", "value": 3},
	}
	got, err := fromEntries(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": 1, "b": 2, "c": 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFromEntriesRoundTripsWithToEntries(t *testing.T) {
	orig := map[string]any{"a": 1, "b": "two", "c": true, "d": nil}
	entries, err := toEntries(orig)
	if err != nil {
		t.Fatalf("toEntries: %v", err)
	}
	back, err := fromEntries(entries)
	if err != nil {
		t.Fatalf("fromEntries: %v", err)
	}
	if !reflect.DeepEqual(back, orig) {
		t.Errorf("round-trip lost data: got %v, want %v", back, orig)
	}
}

func TestFromEntriesMissingValueBecomesNil(t *testing.T) {
	got, err := fromEntries([]map[string]any{
		{"key": "a", "value": 1},
		{"key": "b"}, // no "value"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["b"] != nil {
		t.Errorf("expected b -> nil, got %v", got["b"])
	}
}

func TestFromEntriesRejectsNonSlice(t *testing.T) {
	rejects := []any{
		nil,
		42,
		"string",
		map[string]int{"a": 1},
		true,
	}
	for _, v := range rejects {
		if _, err := fromEntries(v); err == nil {
			t.Errorf("expected error for %T, got nil", v)
		} else if !strings.Contains(err.Error(), "fromEntries:") {
			t.Errorf("error should be prefixed with 'fromEntries:', got: %v", err)
		}
	}
}

func TestFromEntriesRejectsBadEntries(t *testing.T) {
	tests := []struct {
		name    string
		in      any
		wantMsg string
	}{
		{"non-map entry", []any{"not a map"}, "fromEntries[0]: expected map"},
		{"missing key field", []map[string]any{{"value": 1}}, "fromEntries[0]: missing 'key' field"},
		{"non-string key", []map[string]any{{"key": 42, "value": 1}}, "fromEntries[0]: 'key' must be a string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fromEntries(tt.in)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error should contain %q, got: %v", tt.wantMsg, err)
			}
		})
	}
}

func TestDefinedAt(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"deep non-nil value", "server.tls.enabled", true},
		{"empty string is defined (non-nil)", "empty", true},
		{"nil value is NOT defined", "nilvalue", false},
		{"missing path is NOT defined", "missing", false},
		{"missing deep is NOT defined", "server.missing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := definedAt(tt.path, nested); got != tt.want {
				t.Errorf("definedAt(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
