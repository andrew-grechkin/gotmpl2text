// Package regex provides ergonomic regex helpers - substitute, substituteAll, split, test, match - replacing sprig's
// regex family whose argument order and return shape don't compose well in template pipelines
package regex

import (
	"fmt"
	"regexp"
	"text/template"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"substitute":    substitute,
		"substituteAll": substituteAll,
		"splitBy":       splitBy,
		"test":          test,
		"match":         match,
	}
}

// replaces first match using regex (think of text =~ s///r)
func substitute(pattern, repl, text string) (string, error) {
	if pattern == "" {
		return text, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("substitute: invalid regex '%s': %w", pattern, err)
	}
	loc := re.FindStringSubmatchIndex(text)
	if loc == nil {
		return text, nil // No match found
	}
	var result []byte
	result = re.ExpandString(result, repl, text, loc)
	return text[:loc[0]] + string(result) + text[loc[1]:], nil
}

// replaces all matches using regex (think of text =~ s///gr)
func substituteAll(pattern, repl, text string) (string, error) {
	if pattern == "" {
		return text, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("substituteAll: invalid regex '%s': %w", pattern, err)
	}
	return re.ReplaceAllString(text, repl), nil
}

// splits a string by a regular expression pattern
//
// Named "splitBy" (not "split") to leave sprig's own `split` alone - sprig's version returns a map keyed by "_0", "_1",
// ... which is a bad API but a stable one that existing templates rely on. At the call site the pattern argument makes
// the intent obvious: `.text | splitBy "\s+"`.
//
// Supports 2-argument and 3-argument forms:
//   - 2 args (pattern, text): Splits unconditionally
//     Pipeline: {{ .text | splitBy "pattern" }}
//   - 3 args (pattern, max_items, text): Limits the number of returned substrings.
//     Pipeline: {{ .text | splitBy "pattern" 2 }}
func splitBy(pattern string, args ...any) ([]string, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("splitBy: expected 2 or 3 arguments, got %d", len(args)+1)
	}

	text, n, err := splitByParseArgs(args)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("splitBy: invalid regex '%s': %w", pattern, err)
	}

	return re.Split(text, n), nil
}

// extracts the text and split limit from splitBy's variadic args. One arg: text only (unlimited). Two args: [limit,
// text].
func splitByParseArgs(args []any) (text string, n int, err error) {
	n = -1 // Default: unlimited splits

	if len(args) == 1 {
		return fmt.Sprint(args[0]), n, nil
	}

	switch v := args[0].(type) {
	case int:
		n = v
	case int64:
		n = int(v)
	case float64:
		n = int(v)
	default:
		return "", 0, fmt.Errorf("splitBy: max_items must be an integer, got %T", args[0])
	}

	return fmt.Sprint(args[1]), n, nil
}

// checks if a regular expression matches a string.
// Returns true if the pattern matches anywhere in text, false otherwise.
//
// Pipeline usage:
//
//	{{ .text | test "^v[0-9]+" }}
func test(pattern string, text string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("test: invalid regex '%s': %w", pattern, err)
	}
	return re.MatchString(text), nil
}

// extracts captured groups from a regular expression. Returns a slice of strings containing only parenthesized capture
// groups:
//   - index 0 = Capture Group 1 ($1)
//   - index 1 = Capture Group 2 ($2)
//
// Returns an empty slice []string{} if no match occurs or no groups exist.
//
// Pipeline usage:
//
//	{{ .text | match "^([^@]+)@(.+)$" }}
func match(pattern string, text string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("match: invalid regex '%s': %w", pattern, err)
	}

	submatches := re.FindStringSubmatch(text)
	if len(submatches) <= 1 {
		return []string{}, nil
	}

	// Strip index 0 (full match) so returned array has size of amount of capture groups in pattern
	return submatches[1:], nil
}
