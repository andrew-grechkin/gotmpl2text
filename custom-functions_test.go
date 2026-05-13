package main

import (
	"os"
	"strings"
	"testing"
)

func TestCustomFunctions(t *testing.T) {
	// Set GOTMPL_FUNCTIONS to point to our example functions.yaml
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	functionsFile := absPath + "/examples/functions.yaml"

	os.Setenv("GOTMPL_FUNCTIONS", functionsFile)
	defer os.Unsetenv("GOTMPL_FUNCTIONS")

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "toHarnessId function",
			template: `{{ "my-service_name" | toHarnessId }}`,
			want:     "my_service__name",
			wantErr:  false,
		},
		{
			name:     "toHarnessId with complex input",
			template: `{{ "foo-bar_baz@123" | toHarnessId }}`,
			want:     "foo_bar__baz_123",
			wantErr:  false,
		},
		{
			name:     "shout function",
			template: `{{ "hello world" | shout }}`,
			want:     "HELLO WORLD",
			wantErr:  false,
		},
		{
			name:     "withPrefix function",
			template: `{{ "myvalue" | withPrefix }}`,
			want:     "prefix_myvalue",
			wantErr:  false,
		},
		{
			name:     "slugify function",
			template: `{{ "Hello World! 123" | slugify }}`,
			want:     "hello-world-123",
			wantErr:  false,
		},
		{
			name:     "multiple custom functions",
			template: `{{ "test" | withPrefix | shout }}`,
			want:     "PREFIX_TEST",
			wantErr:  false,
		},
		{
			name:     "custom function with embedded data",
			template: "{{ .name | toHarnessId }}\n{{/* __DATA__\nname: my-app_v2\n*/}}",
			want:     "my_app__v2\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template)
			if tt.wantErr {
				if err == nil {
					t.Errorf("run() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}

			if got != tt.want {
				t.Errorf("run() got output %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCustomFunctionsNotFound(t *testing.T) {
	// Set XDG_CONFIG_HOME to a non-existent directory
	os.Setenv("XDG_CONFIG_HOME", "/tmp/nonexistent-gotmpl2text-test")
	defer os.Unsetenv("XDG_CONFIG_HOME")

	// Make sure GOTMPL_FUNCTIONS and GOTMPL_PRELOAD are not set from previous tests or environment
	os.Unsetenv("GOTMPL_FUNCTIONS")
	os.Unsetenv("GOTMPL_PRELOAD")

	// Should work fine without custom functions file
	got, err := runTemplate(t, `{{ "test" | upper }}`)
	if err != nil {
		t.Fatalf("run() failed when custom functions file doesn't exist: %v", err)
	}

	want := "TEST"
	if got != want {
		t.Errorf("run() got output %q, want %q", got, want)
	}
}

func TestCustomFunctionTypes(t *testing.T) {
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	functionsFile := absPath + "/test/fixtures/custom-functions/typed-functions.yaml"

	t.Setenv("GOTMPL_FUNCTIONS", functionsFile)

	tests := []struct {
		name     string
		template string
		wantOut  string
		wantErr  bool
	}{
		{
			name:     "string type explicit",
			template: `{{ "hello" | toUpperStr }}`,
			wantOut:  "HELLO",
			wantErr:  false,
		},
		{
			name:     "string type implicit default",
			template: `{{ "WORLD" | toLowerStr }}`,
			wantOut:  "world",
			wantErr:  false,
		},
		{
			name:     "string preserves whitespace",
			template: `{{ "  spaced  " | toLowerStr }}`,
			wantOut:  "  spaced  ",
			wantErr:  false,
		},
		{
			name:     "int64 type returns integer",
			template: `{{ $len := "hello" | getLength }}{{ if gt $len 3 }}pass{{ end }}`,
			wantOut:  "pass",
			wantErr:  false,
		},
		{
			name:     "int64 arithmetic",
			template: `{{ $len := "test" | getLength }}{{ add $len 10 }}`,
			wantOut:  "14",
			wantErr:  false,
		},
		{
			name:     "bool type in conditional",
			template: `{{ if "hello" | isLong }}yes{{ else }}no{{ end }}`,
			wantOut:  "no",
			wantErr:  false,
		},
		{
			name:     "bool type true",
			template: `{{ if "hello world" | isLong }}yes{{ else }}no{{ end }}`,
			wantOut:  "yes",
			wantErr:  false,
		},
		{
			name:     "float64 type",
			template: `{{ $half := 10 | halfValue }}{{ printf "%.1f" $half }}`,
			wantOut:  "5.0",
			wantErr:  false,
		},
		{
			name:     "float64 comparison",
			template: `{{ $val := 10 | halfValue }}{{ if gt $val 4.0 }}pass{{ end }}`,
			wantOut:  "pass",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.wantOut {
				t.Errorf("got %q, want %q", got, tt.wantOut)
			}
		})
	}
}

func TestCustomFunctionTypeErrors(t *testing.T) {
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	tests := []struct {
		name       string
		yamlFile   string
		template   string
		wantErrMsg string
	}{
		{
			name:       "unknown type",
			yamlFile:   absPath + "/test/fixtures/custom-functions/error-unknown-type.yaml",
			template:   `{{ 1 | badType }}`,
			wantErrMsg: `unknown type "uint32" for custom function "badType"`,
		},
		{
			name:       "int64 conversion error",
			yamlFile:   absPath + "/test/fixtures/custom-functions/error-int64-conversion.yaml",
			template:   `{{ "not-a-number" | toInt }}`,
			wantErrMsg: `custom function "toInt": cannot convert "not-a-number" to int64`,
		},
		{
			name:       "float64 conversion error",
			yamlFile:   absPath + "/test/fixtures/custom-functions/error-float64-conversion.yaml",
			template:   `{{ "invalid" | toFloat }}`,
			wantErrMsg: `custom function "toFloat": cannot convert "invalid" to float64`,
		},
		{
			name:       "bool conversion error",
			yamlFile:   absPath + "/test/fixtures/custom-functions/error-bool-conversion.yaml",
			template:   `{{ "maybe" | toBool }}`,
			wantErrMsg: `custom function "toBool": cannot convert "maybe" to bool`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GOTMPL_FUNCTIONS", tt.yamlFile)

			_, err := runTemplate(t, tt.template)
			if err == nil {
				t.Fatalf("expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error message %q doesn't contain expected %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
