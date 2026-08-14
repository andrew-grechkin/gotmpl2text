package text

import (
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple lowercase", "hello", "hello"},
		{"uppercase to lowercase", "HELLO", "hello"},
		{"mixed case", "HelloWorld", "helloworld"},
		{"space separator", "hello world", "hello-world"},
		{"multiple spaces collapse", "hello    world", "hello-world"},
		{"mixed punctuation", "Hello, World!", "hello-world"},
		{"digits kept", "top 10 hits 2024", "top-10-hits-2024"},
		{"trim trailing separators", "hello!!!", "hello"},
		{"trim leading separators", "!!!hello", "hello"},
		{"unicode is treated as separator (ASCII-only), remaining ASCII kept", "café", "caf"},
		{"consecutive non-alphanumeric become single hyphen", "a__b--c!!d", "a-b-c-d"},
		{"internal underscores become hyphens", "my_var_name", "my-var-name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := slug(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("slug(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSlugErrorsOnEmptyResult(t *testing.T) {
	// Empty input, all-punctuation input, and all-unicode input all fail to produce any alphanumeric character - loud
	// error is the right behavior
	cases := []string{"", "!!!", "   ", "---", "世界"}
	for _, s := range cases {
		if _, err := slug(s); err == nil {
			t.Errorf("expected error for %q, got nil", s)
		} else if !strings.Contains(err.Error(), "slug:") {
			t.Errorf("error should be prefixed with 'slug:', got: %v", err)
		}
	}
}
