package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

// runTemplate is a test helper that executes a template and returns the output
func runTemplate(t *testing.T, template string, args ...string) (string, error) {
	t.Helper()
	fullArgs := append([]string{"gotmpl2text"}, args...)
	stdin := strings.NewReader(template)
	var stdout bytes.Buffer
	err := run(fullArgs, stdin, &stdout)
	return stdout.String(), err
}

func TestSplitTemplateData(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantBlocks []string
	}{
		{
			name:       "single __DATA__ block",
			content:    "Hello {{ .name }}\n{{/* __DATA__\nname: world\n*/}}",
			wantBlocks: []string{"name: world"},
		},
		{
			name:       "multiple blocks",
			content:    "Hello\n{{/* __DATA__\na: 1\n*/}}\nWorld\n{{/* __DATA__\nb: 2\n*/}}",
			wantBlocks: []string{"a: 1", "b: 2"},
		},
		{
			name:       "no blocks",
			content:    "Hello {{ .name }}",
			wantBlocks: nil,
		},
		{
			name:       "whitespace variations",
			content:    "Hello\n\n\n{{/*   __DATA__  \nspaced\n*/}}\n\n\n",
			wantBlocks: []string{"spaced"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotBlocks := splitTemplateData(tt.content)

			if !reflect.DeepEqual(gotBlocks, tt.wantBlocks) {
				t.Errorf("splitTemplateData() gotBlocks = %q, want %q", gotBlocks, tt.wantBlocks)
			}
		})
	}
}

func TestRunWhitespaceControl(t *testing.T) {
	// These tests verify that standard Go text/template whitespace controls
	// behave correctly and are not mangled by data splitting or reading logic.
	tests := []struct {
		name     string
		template string
		args     []string
		want     string
	}{
		{
			name:     "without trailing dash (keeps newline)",
			template: "{{ .name }}: {{ .replicas }}\n",
			args:     []string{"gotmpl2text", "testdata.yaml"},
			want:     "api: 3\n",
		},
		{
			name:     "with trailing dash (strips newline)",
			template: "{{ .name }}: {{ .replicas -}}\n",
			args:     []string{"gotmpl2text", "testdata.yaml"},
			want:     "api: 3",
		},
		{
			name:     "embedded data without trailing dash (keeps newline)",
			template: "{{ .name }}: {{ .replicas }}\n{{/* __DATA__\nname: api\nreplicas: 3\n*/}}",
			args:     []string{"gotmpl2text"},
			want:     "api: 3\n",
		},
		{
			name:     "embedded data with trailing dash (strips newline)",
			template: "{{ .name }}: {{ .replicas -}}\n{{/* __DATA__\nname: api\nreplicas: 3\n*/}}",
			args:     []string{"gotmpl2text"},
			want:     "api: 3",
		},
	}

	testDataFile := "testdata.yaml"
	if err := os.WriteFile(testDataFile, []byte("name: api\nreplicas: 3"), 0644); err != nil {
		t.Fatalf("Failed to write test data file: %v", err)
	}
	defer os.Remove(testDataFile)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template, tt.args[1:]...)
			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}

			if got != tt.want {
				t.Errorf("run() got output %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunMissingKeys(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		allowMissing string
		want         string
		wantErr      bool
	}{
		{
			name:         "missing key error (default)",
			template:     "{{ .missing }}",
			allowMissing: "",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "missing key allowed (env=1)",
			template:     "{{ .missing }}",
			allowMissing: "1",
			want:         "<no value>",
			wantErr:      false,
		},
		{
			name:         "missing key allowed (env=true)",
			template:     "{{ .missing }}",
			allowMissing: "true",
			want:         "<no value>",
			wantErr:      false,
		},
		{
			name:         "missing key error (env=0)",
			template:     "{{ .missing }}",
			allowMissing: "0",
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear/Set env
			if tt.allowMissing != "" {
				os.Setenv("GOTMPL_ALLOW_MISSING", tt.allowMissing)
			} else {
				os.Unsetenv("GOTMPL_ALLOW_MISSING")
			}
			defer os.Unsetenv("GOTMPL_ALLOW_MISSING")

			got, err := runTemplate(t, tt.template)
			if tt.wantErr {
				if err == nil {
					t.Errorf("run() expected error for missing key, but got none")
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

func TestRunIgnoreEmbed(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		ignoreEmbed string
		want        string
		wantErr     bool
	}{
		{
			name:        "use embedded data (default)",
			template:    "Hello {{ .name }}\n{{/* __DATA__\nname: world\n*/}}",
			ignoreEmbed: "",
			want:        "Hello world\n",
			wantErr:     false,
		},
		{
			name:        "ignore embedded data (env=1)",
			template:    "Hello {{ .name }}\n{{/* __DATA__\nname: world\n*/}}",
			ignoreEmbed: "1",
			want:        "",
			wantErr:     true, // Should fail because 'name' is now missing
		},
		{
			name:        "ignore embedded data (env=true)",
			template:    "Hello {{ .name }}\n{{/* __DATA__\nname: world\n*/}}",
			ignoreEmbed: "true",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ignoreEmbed != "" {
				os.Setenv("GOTMPL_IGNORE_EMBED", tt.ignoreEmbed)
			} else {
				os.Unsetenv("GOTMPL_IGNORE_EMBED")
			}
			defer os.Unsetenv("GOTMPL_IGNORE_EMBED")

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

func TestPreloadTemplates(t *testing.T) {
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	helpersFile := absPath + "/test/fixtures/helpers.tmpl"
	commonFile := absPath + "/test/fixtures/common.tmpl"

	tests := []struct {
		name     string
		preload  string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "single preload file",
			preload:  helpersFile,
			template: `{{ include "greeting" "World" }}`,
			want:     "Hello, World!",
			wantErr:  false,
		},
		{
			name:     "multiple preload files",
			preload:  helpersFile + "," + commonFile,
			template: `{{ include "greeting" "Test" }} {{ include "banner" "Title" }}`,
			want: `Hello, Test! =================================
Title
=================================`,
			wantErr: false,
		},
		{
			name:     "preload with spaces in list",
			preload:  helpersFile + " , " + commonFile,
			template: `{{ include "upper" "hello" }}`,
			want:     "HELLO",
			wantErr:  false,
		},
		{
			name:     "preload with embedded data",
			preload:  helpersFile,
			template: "{{ include \"greeting\" .name }}\n{{/* __DATA__\nname: Alice\n*/}}",
			want:     "Hello, Alice!\n",
			wantErr:  false,
		},
		{
			name:     "no preload",
			preload:  "",
			template: `{{ "test" | upper }}`,
			want:     "TEST",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preload != "" {
				os.Setenv("GOTMPL_PRELOAD", tt.preload)
			} else {
				os.Unsetenv("GOTMPL_PRELOAD")
			}
			defer os.Unsetenv("GOTMPL_PRELOAD")

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

func TestPreloadTemplatesFileNotFound(t *testing.T) {
	os.Setenv("GOTMPL_PRELOAD", "/nonexistent/file.tmpl")
	defer os.Unsetenv("GOTMPL_PRELOAD")

	_, err := runTemplate(t, `{{ "test" }}`)
	if err == nil {
		t.Errorf("run() expected error when preload file doesn't exist, but got none")
		return
	}

	// Verify it's a PreloadError
	var preloadErr *PreloadError
	if !reflect.TypeOf(err).ConvertibleTo(reflect.TypeOf(preloadErr)) {
		t.Errorf("run() expected PreloadError, got %T", err)
	}

	// Verify error message contains the file name
	if !strings.Contains(err.Error(), "/nonexistent/file.tmpl") {
		t.Errorf("run() error message should contain file name, got: %v", err)
	}
}
