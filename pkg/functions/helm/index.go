package helm

import (
	"fmt"
	"maps"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

// indentLines adds padding to non-empty lines in a string
func indentLines(spaces int, v string) string {
	padding := strings.Repeat(" ", spaces)
	var b strings.Builder
	b.Grow(len(v) + strings.Count(v, "\n")*spaces)
	for i, line := range strings.Split(v, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		if line != "" {
			b.WriteString(padding)
			b.WriteString(line)
		}
	}
	return b.String()
}

// FuncMap returns a FuncMap with Sprig functions plus Helm-specific functions
func FuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	// Add Helm-specific functions
	helmFuncs := template.FuncMap{
		// include is a stub here and replaced in run() after template parsing
		// because it needs access to the parsed template object (circular dependency)
		"include": func(name string, data any) (string, error) {
			return "", fmt.Errorf("include function not properly initialized")
		},

		"required": func(msg string, val any) (any, error) {
			if val == nil {
				return nil, fmt.Errorf("required value not provided: %s", msg)
			}
			if str, ok := val.(string); ok && str == "" {
				return nil, fmt.Errorf("required value not provided: %s", msg)
			}
			return val, nil
		},

		"toYaml": func(v any) (string, error) {
			data, err := yaml.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("toYaml: %w", err)
			}
			return strings.TrimSuffix(string(data), "\n"), nil
		},

		"fromYaml": func(str string) (any, error) {
			var v any
			if err := yaml.Unmarshal([]byte(str), &v); err != nil {
				return nil, fmt.Errorf("fromYaml: %w", err)
			}
			return v, nil
		},

		"nindent": func(spaces int, v string) string {
			return "\n" + indentLines(spaces, v)
		},

		"indent": func(spaces int, v string) string {
			return indentLines(spaces, v)
		},
	}

	maps.Copy(funcMap, helmFuncs)

	return funcMap
}
