package json

import (
	"reflect"
	"strings"
	"testing"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"scalar", 42, "42"},
		{"string", "hi", `"hi"`},
		{"map", map[string]any{"a": 1}, `{"a":1}`},
		{"slice", []int{1, 2, 3}, "[1,2,3]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toJSON(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("toJSON(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToJSONErrorsOnUnmarshallable(t *testing.T) {
	// Channels can't be marshaled - previously sprig's toJson would return "", silently corrupting output. Now it halts
	if _, err := toJSON(make(chan int)); err == nil {
		t.Error("expected error for channel, got nil")
	} else if !strings.Contains(err.Error(), "toJson:") {
		t.Errorf("error should be prefixed with 'toJson:', got: %v", err)
	}
}

func TestToPrettyJSON(t *testing.T) {
	got, err := toPrettyJSON(map[string]any{"a": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "{\n  \"a\": 1\n}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToRawJSONDoesNotEscapeHTML(t *testing.T) {
	// Standard toJson escapes < > & for HTML safety. toRawJson must NOT
	got, err := toRawJSON("<b>&</b>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `"<b>&</b>"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToRawJSONStripsTrailingNewline(t *testing.T) {
	// json.Encoder appends "\n"; we strip it
	got, err := toRawJSON(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.HasSuffix(got, "\n") {
		t.Errorf("expected no trailing newline, got %q", got)
	}
}

func TestToJSONL(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"array of ints", []int{1, 2, 3}, "1\n2\n3"},
		{"array of strings", []string{"a", "b"}, "\"a\"\n\"b\""},
		{"array of maps", []map[string]any{{"a": 1}, {"b": 2}}, "{\"a\":1}\n{\"b\":2}"},
		{"empty slice", []int{}, ""},
		{"nil typed slice", []int(nil), ""},
		{"fixed array", [2]string{"x", "y"}, "\"x\"\n\"y\""},
		{"mixed types via any", []any{1, "two", true, nil}, "1\n\"two\"\ntrue\nnull"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toJSONL(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("toJSONL(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToJSONLNoTrailingNewline(t *testing.T) {
	got, err := toJSONL([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.HasSuffix(got, "\n") {
		t.Errorf("expected no trailing newline, got %q", got)
	}
}

func TestToJSONLRejectsNonArrays(t *testing.T) {
	rejects := []any{
		42,
		"a string",
		map[string]int{"a": 1},
		nil,
		true,
	}
	for _, v := range rejects {
		if _, err := toJSONL(v); err == nil {
			t.Errorf("expected error for %T (%v), got nil", v, v)
		} else if !strings.Contains(err.Error(), "toJsonL:") {
			t.Errorf("error should be prefixed with 'toJsonL:', got: %v", err)
		}
	}
}

func TestToJSONLPropagatesElementError(t *testing.T) {
	// A channel element can't be marshaled; the error should mention the index
	if _, err := toJSONL([]any{1, make(chan int), 3}); err == nil {
		t.Error("expected error for unmarshallable element, got nil")
	} else if !strings.Contains(err.Error(), "toJsonL[1]") {
		t.Errorf("error should include element index [1], got: %v", err)
	}
}

func TestFromJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want any
	}{
		{"object", `{"a":1}`, map[string]any{"a": float64(1)}}, // JSON numbers unmarshal as float64
		{"array", `[1,2,3]`, []any{float64(1), float64(2), float64(3)}},
		{"scalar number", `42`, float64(42)},
		{"scalar string", `"hi"`, "hi"},
		{"null", `null`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fromJSON(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromJSON(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestFromJSONErrorsOnMalformed(t *testing.T) {
	// Previously sprig's fromJson returned map{"Error": "..."} - a value downstream code couldn't distinguish from
	// legitimate data. Now halts
	if _, err := fromJSON("not valid"); err == nil {
		t.Error("expected error for malformed JSON, got nil")
	} else if !strings.Contains(err.Error(), "fromJson:") {
		t.Errorf("error should be prefixed with 'fromJson:', got: %v", err)
	}
}

func TestFromJSONAcceptsArraysAndScalars(t *testing.T) {
	// Regression: sprig's fromJson only accepted objects. Ours accepts any valid JSON value (matches sprig's
	// mustFromJson shape)
	if _, err := fromJSON(`[1,2,3]`); err != nil {
		t.Errorf("expected array to parse, got: %v", err)
	}
	if _, err := fromJSON(`42`); err != nil {
		t.Errorf("expected scalar to parse, got: %v", err)
	}
}
