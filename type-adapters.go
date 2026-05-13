package main

import (
	"fmt"
	"strconv"
	"strings"
)

// typeAdapter converts string output from template to target type
type typeAdapter func(string) (any, error)

// typeAdapters maps type names to conversion functions
var typeAdapters = map[string]typeAdapter{
	"string": func(s string) (any, error) {
		return s, nil
	},

	"int64": func(s string) (any, error) {
		trimmed := strings.TrimSpace(s)
		i, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int64: %w", s, err)
		}
		return i, nil
	},

	"float64": func(s string) (any, error) {
		trimmed := strings.TrimSpace(s)
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float64: %w", s, err)
		}
		return f, nil
	},

	"bool": func(s string) (any, error) {
		trimmed := strings.TrimSpace(s)
		b, err := strconv.ParseBool(trimmed)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to bool: %w", s, err)
		}
		return b, nil
	},
}
