// Package text provides string-transformation helpers
package text

import (
	"fmt"
	"strings"
	"text/template"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"slug": slug,
	}
}

// converts an arbitrary string into a URL/filename-friendly slug: lowercase, ASCII letters and digits kept, everything
// else collapsed to a single hyphen, no leading or trailing hyphens.
//
// Errors if no ASCII alphanumeric characters survive - an empty slug is silent misuse (broken URL path, empty filename)
// and downstream code has no way to detect it. Halt at the source instead
//
// NOTE: this is an ASCII-only implementation. Non-ASCII characters (diacritics, CJK, emoji) are treated as separators.
// For proper Unicode normalization ("café" -> "cafe" instead of "caf") add golang.org/x/text and preprocess with
// norm.NFKD.
func slug(s string) (string, error) {
	var b strings.Builder
	b.Grow(len(s))
	lastSep := true
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSep = false
		case !lastSep:
			b.WriteByte('-')
			lastSep = true
		}
	}
	result := strings.TrimRight(b.String(), "-")
	if result == "" {
		return "", fmt.Errorf("slug: input %q produced empty slug (no ASCII alphanumeric characters)", s)
	}
	return result, nil
}
