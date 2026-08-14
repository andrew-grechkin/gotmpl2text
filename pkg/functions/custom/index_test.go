package custom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestLoadNoConfigReturnsEmptyMap(t *testing.T) {
	// No config path resolvable: unset all env inputs and point HOME at a
	// nonexistent directory so getPath() returns "".
	t.Setenv("GOTMPL_FUNCTIONS", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	got, err := Load(template.FuncMap{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil map; expected empty map")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

func TestLoadMissingFileReturnsEmptyMap(t *testing.T) {
	t.Setenv("GOTMPL_FUNCTIONS", "/nonexistent/gotmpl2text/functions.yaml")

	got, err := Load(template.FuncMap{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil map; expected empty map")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

func TestLoadCompilesFunctionsFromYAML(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "functions.yaml")
	yaml := `functions:
  - name: shout
    template: "{{ . | upper }}"
  - name: getLength
    template: "{{ . | len }}"
    type: int64
  - name: isEmpty
    template: "{{ if eq . \"\" }}true{{ else }}false{{ end }}"
    type: bool
`
	if err := os.WriteFile(cfg, []byte(yaml), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("GOTMPL_FUNCTIONS", cfg)

	// baseFuncMap must expose the primitives the templates use (upper, len).
	base := template.FuncMap{
		"upper": strings.ToUpper,
		"len":   func(s string) int { return len(s) },
	}

	fm, err := Load(base, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	shout := fm["shout"].(func(any) (any, error))
	out, err := shout("hello")
	if err != nil {
		t.Fatalf("shout: %v", err)
	}
	if out != "HELLO" {
		t.Errorf("shout: got %v, want HELLO", out)
	}

	getLength := fm["getLength"].(func(any) (any, error))
	out, err = getLength("hello")
	if err != nil {
		t.Fatalf("getLength: %v", err)
	}
	if got, ok := out.(int64); !ok || got != 5 {
		t.Errorf("getLength: got %v (%T), want int64(5)", out, out)
	}

	isEmpty := fm["isEmpty"].(func(any) (any, error))
	out, err = isEmpty("")
	if err != nil {
		t.Fatalf("isEmpty: %v", err)
	}
	if got, ok := out.(bool); !ok || got != true {
		t.Errorf("isEmpty(\"\"): got %v (%T), want true", out, out)
	}
}

func TestLoadUnknownTypeErrors(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "functions.yaml")
	yaml := `functions:
  - name: bad
    template: "{{ . }}"
    type: uint32
`
	if err := os.WriteFile(cfg, []byte(yaml), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("GOTMPL_FUNCTIONS", cfg)

	_, err := Load(template.FuncMap{}, false)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), `unknown type "uint32"`) {
		t.Errorf("error should mention unknown type, got: %v", err)
	}
}

func TestLoadMalformedYAMLErrors(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "functions.yaml")
	if err := os.WriteFile(cfg, []byte(":\n  not: valid: yaml:"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("GOTMPL_FUNCTIONS", cfg)

	_, err := Load(template.FuncMap{}, false)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parsing custom functions YAML") {
		t.Errorf("error should mention YAML parsing, got: %v", err)
	}
}

func TestStringAdapter(t *testing.T) {
	adapter := typeAdapters["string"]
	if adapter == nil {
		t.Fatal("string adapter not found")
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "preserves leading whitespace",
			input: "  hello",
			want:  "  hello",
		},
		{
			name:  "preserves trailing whitespace",
			input: "hello  ",
			want:  "hello  ",
		},
		{
			name:  "preserves all whitespace",
			input: "  hello world  ",
			want:  "  hello world  ",
		},
		{
			name:  "preserves newlines",
			input: "hello\nworld",
			want:  "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, ok := result.(string)
			if !ok {
				t.Fatalf("result is not a string: %T", result)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInt64Adapter(t *testing.T) {
	adapter := typeAdapters["int64"]
	if adapter == nil {
		t.Fatal("int64 adapter not found")
	}

	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:    "positive integer",
			input:   "42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   "-123",
			want:    -123,
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "large number",
			input:   "9223372036854775807",
			want:    9223372036854775807,
			wantErr: false,
		},
		{
			name:    "trims leading whitespace",
			input:   "  42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "trims trailing whitespace",
			input:   "42  ",
			want:    42,
			wantErr: false,
		},
		{
			name:    "trims both whitespace",
			input:   "  42  ",
			want:    42,
			wantErr: false,
		},
		{
			name:    "invalid - not a number",
			input:   "hello",
			wantErr: true,
		},
		{
			name:    "invalid - float",
			input:   "42.5",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid - mixed",
			input:   "42abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if !strings.Contains(err.Error(), "cannot convert") {
					t.Errorf("error message should contain 'cannot convert', got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, ok := result.(int64)
			if !ok {
				t.Fatalf("result is not int64: %T", result)
			}

			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFloat64Adapter(t *testing.T) {
	adapter := typeAdapters["float64"]
	if adapter == nil {
		t.Fatal("float64 adapter not found")
	}

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "integer",
			input:   "42",
			want:    42.0,
			wantErr: false,
		},
		{
			name:    "float",
			input:   "3.14",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "negative float",
			input:   "-2.5",
			want:    -2.5,
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0.0",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "scientific notation",
			input:   "1.23e10",
			want:    1.23e10,
			wantErr: false,
		},
		{
			name:    "trims leading whitespace",
			input:   "  3.14",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "trims trailing whitespace",
			input:   "3.14  ",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "trims both whitespace",
			input:   "  3.14  ",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "invalid - not a number",
			input:   "hello",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid - mixed",
			input:   "3.14abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if !strings.Contains(err.Error(), "cannot convert") {
					t.Errorf("error message should contain 'cannot convert', got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, ok := result.(float64)
			if !ok {
				t.Fatalf("result is not float64: %T", result)
			}

			if got != tt.want {
				t.Errorf("got %f, want %f", got, tt.want)
			}
		})
	}
}

func TestBoolAdapter(t *testing.T) {
	adapter := typeAdapters["bool"]
	if adapter == nil {
		t.Fatal("bool adapter not found")
	}

	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:    "true lowercase",
			input:   "true",
			want:    true,
			wantErr: false,
		},
		{
			name:    "false lowercase",
			input:   "false",
			want:    false,
			wantErr: false,
		},
		{
			name:    "true uppercase",
			input:   "TRUE",
			want:    true,
			wantErr: false,
		},
		{
			name:    "false uppercase",
			input:   "FALSE",
			want:    false,
			wantErr: false,
		},
		{
			name:    "true mixed case",
			input:   "True",
			want:    true,
			wantErr: false,
		},
		{
			name:    "1 is true",
			input:   "1",
			want:    true,
			wantErr: false,
		},
		{
			name:    "0 is false",
			input:   "0",
			want:    false,
			wantErr: false,
		},
		{
			name:    "trims leading whitespace",
			input:   "  true",
			want:    true,
			wantErr: false,
		},
		{
			name:    "trims trailing whitespace",
			input:   "false  ",
			want:    false,
			wantErr: false,
		},
		{
			name:    "trims both whitespace",
			input:   "  true  ",
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid - not a bool",
			input:   "maybe",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid - number other than 0 or 1",
			input:   "2",
			wantErr: true,
		},
		{
			name:    "invalid - yes",
			input:   "yes",
			wantErr: true,
		},
		{
			name:    "invalid - no",
			input:   "no",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if !strings.Contains(err.Error(), "cannot convert") {
					t.Errorf("error message should contain 'cannot convert', got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, ok := result.(bool)
			if !ok {
				t.Fatalf("result is not bool: %T", result)
			}

			if got != tt.want {
				t.Errorf("got %t, want %t", got, tt.want)
			}
		})
	}
}

func TestTypeAdaptersExist(t *testing.T) {
	expectedTypes := []string{"string", "int64", "float64", "bool"}

	for _, typeName := range expectedTypes {
		t.Run(typeName, func(t *testing.T) {
			adapter, ok := typeAdapters[typeName]
			if !ok {
				t.Errorf("type adapter %q not found", typeName)
			}
			if adapter == nil {
				t.Errorf("type adapter %q is nil", typeName)
			}
		})
	}
}
