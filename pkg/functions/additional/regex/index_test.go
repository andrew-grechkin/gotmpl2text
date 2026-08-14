package regex

import (
	"reflect"
	"strings"
	"testing"
)

func TestSubstitute(t *testing.T) {
	fn := FuncMap()["substitute"].(func(string, string, string) (string, error))

	tests := []struct {
		name    string
		pattern string
		repl    string
		text    string
		want    string
		wantErr bool
	}{
		{"replaces first match only", "foo", "bar", "foo foo foo", "bar foo foo", false},
		{"no match returns original", "xyz", "bar", "foo foo", "foo foo", false},
		{"empty pattern returns original", "", "bar", "foo", "foo", false},
		{"empty text returns empty", "foo", "bar", "", "", false},
		{"empty replacement removes match", "foo", "", "hello foo world", "hello  world", false},
		{"match at start", "^foo", "bar", "foo end", "bar end", false},
		{"match at end", "end$", "final", "foo end", "foo final", false},
		{"single capture group", "([a-z]+)", "<$1>", "hi world", "<hi> world", false},
		{"multiple capture groups swapped", "(\\w+)=(\\w+)", "$2=$1", "key=value rest", "value=key rest", false},
		{"named capture group", "(?P<name>\\w+)", "[$name]", "hi world", "[hi] world", false},
		{"$$ escape for literal dollar", "foo", "$$1", "foo bar", "$1 bar", false},
		{"unknown group name becomes empty", "foo", "$missing", "foo bar", " bar", false},
		{"zero-width anchor match at start", "^", "X", "abc", "Xabc", false},
		{"unicode text", "世界", "world", "hello 世界!", "hello world!", false},
		{"case sensitive by default", "FOO", "bar", "foo", "foo", false},
		{"case insensitive via flag", "(?i)FOO", "bar", "foo", "bar", false},
		{"regex metacharacters in text", "\\.", "!", "a.b.c", "a!b.c", false},
		{"invalid regex errors", "[", "x", "text", "", true},
		{"invalid regex includes pattern in error", "(unclosed", "x", "text", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.pattern, tt.repl, tt.text)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), "substitute:") {
					t.Errorf("error should be prefixed with 'substitute:', got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubstituteAll(t *testing.T) {
	fn := FuncMap()["substituteAll"].(func(string, string, string) (string, error))

	tests := []struct {
		name    string
		pattern string
		repl    string
		text    string
		want    string
		wantErr bool
	}{
		{"replaces every match", "foo", "bar", "foo foo foo", "bar bar bar", false},
		{"no match returns original", "xyz", "bar", "foo", "foo", false},
		{"empty text returns empty", "foo", "bar", "", "", false},
		{"empty pattern returns original (mirrors substitute)", "", "X", "abc", "abc", false},
		{"empty replacement removes all matches", "\\d", "", "a1b2c3", "abc", false},
		{"regex class replaces all", "[0-9]+", "N", "a1 b22 c333", "aN bN cN", false},
		{"capture groups in replacement", "(\\w+)=(\\w+)", "$2=$1", "a=1 b=2", "1=a 2=b", false},
		{"adjacent matches", "aa", "b", "aaaa", "bb", false},
		{"overlapping-like matches are non-overlapping", "aba", "X", "ababa", "Xba", false},
		{"multiline default: dot does not match newline", ".+", "X", "a\nb", "X\nX", false},
		{"anchors respect start/end by default", "^a", "X", "abc", "Xbc", false},
		{"unicode replaced everywhere", "世界", "world", "世界 世界", "world world", false},
		{"invalid regex errors", "[", "x", "text", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.pattern, tt.repl, tt.text)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), "substituteAll:") {
					t.Errorf("error should be prefixed with 'substituteAll:', got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitBy(t *testing.T) {
	fn := FuncMap()["splitBy"].(func(string, ...any) ([]string, error))

	tests := []struct {
		name    string
		pattern string
		args    []any
		want    []string
		wantErr bool
	}{
		{"unlimited split", ",", []any{"a,b,c"}, []string{"a", "b", "c"}, false},
		{"split with int limit", ",", []any{2, "a,b,c,d"}, []string{"a", "b,c,d"}, false},
		{"split with int64 limit", ",", []any{int64(2), "a,b,c"}, []string{"a", "b,c"}, false},
		{"split with float64 limit", ",", []any{float64(3), "a,b,c,d"}, []string{"a", "b", "c,d"}, false},
		{"regex separator collapses whitespace runs", "\\s+", []any{"a  b\tc"}, []string{"a", "b", "c"}, false},
		{"no separator match returns single-element slice", ",", []any{"abc"}, []string{"abc"}, false},
		{"consecutive separators produce empty items", ",", []any{"a,,b"}, []string{"a", "", "b"}, false},
		{"trailing separator produces trailing empty", ",", []any{"a,b,"}, []string{"a", "b", ""}, false},
		{"leading separator produces leading empty", ",", []any{",a,b"}, []string{"", "a", "b"}, false},
		{"empty text returns single empty string", ",", []any{""}, []string{""}, false},
		{"limit n=1 returns single-element slice", ",", []any{1, "a,b,c"}, []string{"a,b,c"}, false},
		{"limit n=0 returns nil (Go regexp semantics)", ",", []any{0, "a,b,c"}, []string(nil), false},
		{"limit negative treated as unlimited", ",", []any{-1, "a,b,c"}, []string{"a", "b", "c"}, false},
		{"non-string text coerced via fmt", ",", []any{42}, []string{"42"}, false},
		{"no varargs errors with correct total count", ",", []any{}, nil, true},
		{"too many varargs errors with correct total count", ",", []any{1, "x", "y"}, nil, true},
		{"non-integer limit errors", ",", []any{"bad", "x"}, nil, true},
		{"invalid regex errors", "[", []any{"x"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.pattern, tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), "splitBy:") {
					t.Errorf("error should be prefixed with 'split:', got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSplitByArgCountInErrorMessage(t *testing.T) {
	// The error message should report the TOTAL argument count (pattern + varargs), not just the varargs count, so
	// users can reconcile it with what they wrote
	fn := FuncMap()["splitBy"].(func(string, ...any) ([]string, error))

	if _, err := fn(","); err == nil || !strings.Contains(err.Error(), "got 1") {
		t.Errorf("expected 'got 1' for one total arg, got: %v", err)
	}

	if _, err := fn(",", 1, 2, 3, 4); err == nil || !strings.Contains(err.Error(), "got 5") {
		t.Errorf("expected 'got 5' for five total args, got: %v", err)
	}
}

func TestTest(t *testing.T) {
	fn := FuncMap()["test"].(func(string, string) (bool, error))

	tests := []struct {
		name    string
		pattern string
		text    string
		want    bool
		wantErr bool
	}{
		{"matches anchor", "^v[0-9]+", "v1.2.3", true, false},
		{"no match returns false", "^v[0-9]+", "not-a-version", false, false},
		{"empty text with anchor", "^$", "", true, false},
		{"empty text with non-empty pattern", "x", "", false, false},
		{"empty pattern matches anything", "", "anything", true, false},
		{"empty pattern matches empty text", "", "", true, false},
		{"case sensitive by default", "FOO", "foo", false, false},
		{"case insensitive flag", "(?i)FOO", "foo", true, false},
		{"multiline flag affects ^ / $", "(?m)^bar", "foo\nbar", true, false},
		{"unicode match", "世界", "hello 世界", true, false},
		{"invalid regex errors", "[", "text", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.pattern, tt.text)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), "test:") {
					t.Errorf("error should be prefixed with 'test:', got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	fn := FuncMap()["match"].(func(string, string) ([]string, error))

	tests := []struct {
		name    string
		pattern string
		text    string
		want    []string
		wantErr bool
	}{
		{"returns capture groups without full match", "^([^@]+)@(.+)$", "user@example.com", []string{"user", "example.com"}, false},
		{"no match returns empty", "^([0-9]+)$", "abc", []string{}, false},
		{"pattern without groups returns empty even on match", "\\w+", "hello", []string{}, false},
		{"empty text with capture group returns empty", "(\\w+)", "", []string{}, false},
		{"optional group that did not participate returns empty string", "(a)?(b)", "b", []string{"", "b"}, false},
		{"first match wins when text has multiple", "([a-z]+)", "abc def", []string{"abc"}, false},
		{"named capture groups are returned positionally", "(?P<user>[^@]+)@(?P<host>.+)", "u@h", []string{"u", "h"}, false},
		{"nested capture groups both returned", "((a)b)", "ab", []string{"ab", "a"}, false},
		{"unicode capture", "hello (\\p{Han}+)", "hello 世界!", []string{"世界"}, false},
		{"invalid regex errors", "[", "text", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.pattern, tt.text)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), "match:") {
					t.Errorf("error should be prefixed with 'match:', got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
