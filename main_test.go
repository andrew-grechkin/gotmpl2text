package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSplitTemplateData(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantTmpl   string
		wantBlocks []string
	}{
		{
			name:       "single __DATA__ block",
			content:    "Hello {{ .name }}\n{{/* __DATA__\nname: world\n*/}}",
			wantTmpl:   "Hello {{ .name }}\n",
			wantBlocks: []string{"name: world"},
		},
		{
			name:       "multiple blocks",
			content:    "Hello\n{{/* __DATA__\na: 1\n*/}}\nWorld\n{{/* __DATA__\nb: 2\n*/}}",
			wantTmpl:   "Hello\nWorld\n",
			wantBlocks: []string{"a: 1", "b: 2"},
		},
		{
			name:       "no blocks",
			content:    "Hello {{ .name }}",
			wantTmpl:   "Hello {{ .name }}",
			wantBlocks: nil,
		},
		{
			name:       "whitespace variations",
			content:    "Hello\n\n\n{{/*   __DATA__  \nspaced\n*/}}\n\n\n",
			wantTmpl:   "Hello\n",
			wantBlocks: []string{"spaced"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTmpl, gotBlocks := splitTemplateData(tt.content)

			if gotTmpl != tt.wantTmpl {
				t.Errorf("splitTemplateData() gotTmpl = %q, want %q", gotTmpl, tt.wantTmpl)
			}

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
			stdin := strings.NewReader(tt.template)
			var stdout bytes.Buffer

			err := run(tt.args, stdin, &stdout)
			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}

			got := stdout.String()
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

			stdin := strings.NewReader(tt.template)
			var stdout bytes.Buffer

			err := run([]string{"gotmpl2text"}, stdin, &stdout)
			if tt.wantErr {
				if err == nil {
					t.Errorf("run() expected error for missing key, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}

			got := stdout.String()
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

			stdin := strings.NewReader(tt.template)
			var stdout bytes.Buffer

			err := run([]string{"gotmpl2text"}, stdin, &stdout)
			if tt.wantErr {
				if err == nil {
					t.Errorf("run() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}

			got := stdout.String()
			if got != tt.want {
				t.Errorf("run() got output %q, want %q", got, tt.want)
			}
		})
	}
}

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
			stdin := strings.NewReader(tt.template)
			var stdout bytes.Buffer

			err := run([]string{"gotmpl2text"}, stdin, &stdout)
			if tt.wantErr {
				if err == nil {
					t.Errorf("run() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}

			got := stdout.String()
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

	// Should work fine without custom functions file
	stdin := strings.NewReader(`{{ "test" | upper }}`)
	var stdout bytes.Buffer

	err := run([]string{"gotmpl2text"}, stdin, &stdout)
	if err != nil {
		t.Fatalf("run() failed when custom functions file doesn't exist: %v", err)
	}

	got := stdout.String()
	want := "TEST"
	if got != want {
		t.Errorf("run() got output %q, want %q", got, want)
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

			stdin := strings.NewReader(tt.template)
			var stdout bytes.Buffer

			err := run([]string{"gotmpl2text"}, stdin, &stdout)
			if tt.wantErr {
				if err == nil {
					t.Errorf("run() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}

			got := stdout.String()
			if got != tt.want {
				t.Errorf("run() got output %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPreloadTemplatesFileNotFound(t *testing.T) {
	os.Setenv("GOTMPL_PRELOAD", "/nonexistent/file.tmpl")
	defer os.Unsetenv("GOTMPL_PRELOAD")

	stdin := strings.NewReader(`{{ "test" }}`)
	var stdout bytes.Buffer

	err := run([]string{"gotmpl2text"}, stdin, &stdout)
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
