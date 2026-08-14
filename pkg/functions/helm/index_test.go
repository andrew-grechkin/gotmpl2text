package helm

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestIndentLines(t *testing.T) {
	tests := []struct {
		name   string
		spaces int
		input  string
		want   string
	}{
		{"empty string", 4, "", ""},
		{"zero spaces", 0, "hello", "hello"},
		{"single line", 2, "hello", "  hello"},
		{"multi-line", 2, "a\nb\nc", "  a\n  b\n  c"},
		{"blank lines preserved without padding", 2, "a\n\nb", "  a\n\n  b"},
		{"trailing newline preserved", 2, "a\n", "  a\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentLines(tt.spaces, tt.input)
			if got != tt.want {
				t.Errorf("indentLines(%d, %q) = %q, want %q", tt.spaces, tt.input, got, tt.want)
			}
		})
	}
}

func TestFuncMapRegistersExpectedNames(t *testing.T) {
	fm := FuncMap()
	// Helm-specific + a couple of sprig samples to catch regressions where the
	// sprig import silently drops.
	for _, name := range []string{"include", "required", "toYaml", "fromYaml", "nindent", "indent", "upper", "lower"} {
		if _, ok := fm[name]; !ok {
			t.Errorf("expected function %q to be registered", name)
		}
	}
}

// render is a small helper that executes a template with the given FuncMap
// and returns the output. It fails the test on any parse/execute error.
func render(t *testing.T, fm template.FuncMap, src string, data any) string {
	t.Helper()
	tmpl, err := template.New("t").Funcs(fm).Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestFuncMapRenderBehaviour(t *testing.T) {
	fm := FuncMap()

	tests := []struct {
		name string
		src  string
		want string
	}{
		{"toYaml on map", `{{ toYaml . }}`, "a: 1\nb: 2"},
		{"fromYaml round-trip", `{{ "x: 1" | fromYaml | toYaml }}`, "x: 1"},
		{"nindent adds leading newline and indents", `x:{{ "hello" | nindent 2 }}`, "x:\n  hello"},
		{"indent has no leading newline", `{{ "hello" | indent 4 }}`, "    hello"},
	}

	data := map[string]any{"a": 1, "b": 2}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := render(t, fm, tt.src, data)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFromYamlErrorSurfaced(t *testing.T) {
	// Regression: fromYaml used to swallow parse errors and return an empty
	// map, silently rendering as if the input were valid-but-empty.
	fm := FuncMap()
	fn := fm["fromYaml"].(func(string) (any, error))
	if _, err := fn(":\n  bad: yaml: colons"); err == nil {
		t.Error("expected error for malformed YAML, got nil")
	}
}

func TestRequired(t *testing.T) {
	fm := FuncMap()
	fn := fm["required"].(func(string, any) (any, error))

	if _, err := fn("must be set", nil); err == nil {
		t.Error("expected error for nil value")
	}
	if _, err := fn("must be set", ""); err == nil {
		t.Error("expected error for empty string")
	}
	got, err := fn("must be set", "value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "value" {
		t.Errorf("got %v, want %q", got, "value")
	}
}

func TestIncludeStubErrorsUntilRebound(t *testing.T) {
	// The stub `include` in this package's FuncMap is intentionally a
	// placeholder - callers must rebind it after parsing (see cli/template.go).
	// This test pins that contract so we notice if someone silently removes it.
	fm := FuncMap()
	include := fm["include"].(func(string, any) (string, error))
	if _, err := include("anything", nil); err == nil || !strings.Contains(err.Error(), "not properly initialized") {
		t.Errorf("expected stub error, got: %v", err)
	}
}
