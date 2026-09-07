package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// returns the absolute path to the repo root, resolved from this source file's location so tests keep working
// regardless of the caller's cwd or the file being moved within the package
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// test helper that executes a template and returns the output. dataFiles are passed straight through to Run (no
// program-name prefix)
func runTemplate(t *testing.T, template string, dataFiles ...string) (string, error) {
	t.Helper()
	stdin := strings.NewReader(template)
	var stdout bytes.Buffer
	err := Run(dataFiles, stdin, &stdout)
	return stdout.String(), err
}

func TestRunWhitespaceControl(t *testing.T) {
	// These tests verify that standard Go text/template whitespace controls behave correctly and are not mangled by
	// data splitting or reading logic
	dir := t.TempDir()
	testDataFile := filepath.Join(dir, "testdata.yaml")
	if err := os.WriteFile(testDataFile, []byte("name: api\nreplicas: 3"), 0644); err != nil {
		t.Fatalf("Failed to write test data file: %v", err)
	}

	tests := []struct {
		name     string
		template string
		args     []string
		want     string
	}{
		{
			name:     "without trailing dash (keeps newline)",
			template: "{{ .name }}: {{ .replicas }}\n",
			args:     []string{testDataFile},
			want:     "api: 3\n",
		},
		{
			name:     "with trailing dash (strips newline)",
			template: "{{ .name }}: {{ .replicas -}}\n",
			args:     []string{testDataFile},
			want:     "api: 3",
		},
		{
			name:     "embedded data without trailing dash (keeps newline)",
			template: "{{ .name }}: {{ .replicas }}\n{{/* __DATA__\nname: api\nreplicas: 3\n*/}}",
			args:     nil,
			want:     "api: 3\n",
		},
		{
			name:     "embedded data with trailing dash (strips newline)",
			template: "{{ .name }}: {{ .replicas -}}\n{{/* __DATA__\nname: api\nreplicas: 3\n*/}}",
			args:     nil,
			want:     "api: 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template, tt.args...)
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
			t.Setenv("GOTMPL_ALLOW_MISSING", tt.allowMissing)

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
			t.Setenv("GOTMPL_IGNORE_EMBED", tt.ignoreEmbed)

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
	root := projectRoot(t)

	helpersFile := root + "/test/fixtures/helpers.tmpl"
	commonFile := root + "/test/fixtures/common.tmpl"

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
			preload:  helpersFile + preloadSeparator + commonFile,
			template: `{{ include "greeting" "Test" }} {{ include "banner" "Title" }}`,
			want: `Hello, Test! =================================
Title
=================================`,
			wantErr: false,
		},
		{
			name:     "preload with spaces in list",
			preload:  helpersFile + " " + preloadSeparator + " " + commonFile,
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
			t.Setenv("GOTMPL_PRELOAD", tt.preload)

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

func TestPreloadTemplatesEmbeddedData(t *testing.T) {
	// __DATA__ blocks embedded in preload template files should be extracted and merged into the template data, the
	// same way __DATA__ blocks in the STDIN template are
	dir := t.TempDir()
	preloadFile := filepath.Join(dir, "preload-with-data.tmpl")
	preloadContent := `{{- define "greeting" -}}Hello, {{ .name }}!{{- end -}}
{{/* __DATA__
name: from-preload
*/}}`
	if err := os.WriteFile(preloadFile, []byte(preloadContent), 0644); err != nil {
		t.Fatalf("Failed to write preload file: %v", err)
	}

	t.Setenv("GOTMPL_PRELOAD", preloadFile)

	got, err := runTemplate(t, `{{ include "greeting" . }}`)
	if err != nil {
		t.Fatalf("run() failed: %v", err)
	}

	want := "Hello, from-preload!"
	if got != want {
		t.Errorf("run() got output %q, want %q", got, want)
	}
}

func TestPreloadFunctionErrorReportsBothLocations(t *testing.T) {
	// When a STDIN template calls include on a preload-defined template and something inside that preload template
	// errors at execution time, the user must see BOTH sites: the include call in STDIN (with line:col) and the failing
	// expression inside the preload file (with its path + line:col)
	//
	// Expected shape:
	//   template: STDIN:<line>:<col>: executing "STDIN" at <include ...>:
	//     error calling include:
	//       template: <preload path>:<line>:<col>: executing "<name>" at <.field>:
	//         <root cause>
	dir := t.TempDir()
	preloadPath := filepath.Join(dir, "helpers.tmpl")
	preload := "" +
		"{{- define \"greeting\" -}}\n" + // line 1
		"Hi {{ .name }},\n" + // line 2
		"{{ required \"email is required\" .email }}\n" + // line 3 - errors here
		"{{- end -}}\n"
	if err := os.WriteFile(preloadPath, []byte(preload), 0644); err != nil {
		t.Fatalf("write preload: %v", err)
	}
	t.Setenv("GOTMPL_PRELOAD", preloadPath)

	stdin := "" +
		"Report:\n" + // line 1
		"  {{ include \"greeting\" . }}\n" + // line 2 - calls include here
		"{{/* __DATA__\nname: Alice\n*/}}\n"

	_, err := runTemplate(t, stdin)
	if err == nil {
		t.Fatal("expected error but got none")
	}
	msg := err.Error()

	// loadPreloadFiles converts absolute paths to cwd-relative so the error message shows a clean path anchored to the
	// project. Compute the same relative path here so the assertion doesn't depend on where the test process happens to
	// be running from
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	relPreload, err := filepath.Rel(cwd, preloadPath)
	if err != nil {
		t.Fatalf("rel: %v", err)
	}

	// Every frame must land on its own line and start with "template:" so vim errorformat / any `path:line:col:` grep
	// parses each as a separate quickfix entry
	lines := strings.Split(msg, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 error lines (stdin frame + preload frame), got %d:\n%s", len(lines), msg)
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, "template: ") {
			t.Errorf("line %d must start with %q, got: %s", i, "template: ", line)
		}
	}

	// Line 1: STDIN site with the include call, ending in the wrapper
	wantLine1 := []string{
		`template: STDIN:2:5:`,
		`executing "STDIN" at <include "greeting" .>`,
		`error calling include`,
	}
	for _, want := range wantLine1 {
		if !strings.Contains(lines[0], want) {
			t.Errorf("line 1 missing %q; got: %s", want, lines[0])
		}
	}

	// Line 2: preload site with the failing expression and root cause
	wantLine2 := []string{
		`template: ` + relPreload + `:3:`,
		`executing "greeting" at <.email>`,
		`map has no entry for key "email"`,
	}
	for _, want := range wantLine2 {
		if !strings.Contains(lines[1], want) {
			t.Errorf("line 2 missing %q; got: %s", want, lines[1])
		}
	}
}

// proves the multi-line format is not hardcoded to 2 frames. A STDIN template calls "outer" (defined in preload A)
// which calls "inner" (defined in preload B) which errors - three frames, three lines, each parseable independently
func TestPreloadFunctionErrorReportsThreeFrames(t *testing.T) {
	dir := t.TempDir()

	outerPath := filepath.Join(dir, "outer.tmpl")
	outer := "" +
		"{{- define \"outer\" -}}\n" + // line 1
		"outer: {{ include \"inner\" . }}\n" + // line 2 - calls inner
		"{{- end -}}\n"
	if err := os.WriteFile(outerPath, []byte(outer), 0644); err != nil {
		t.Fatalf("write outer: %v", err)
	}

	innerPath := filepath.Join(dir, "inner.tmpl")
	inner := "" +
		"{{- define \"inner\" -}}\n" + // line 1
		"inner: {{ .missing }}\n" + // line 2 - errors here
		"{{- end -}}\n"
	if err := os.WriteFile(innerPath, []byte(inner), 0644); err != nil {
		t.Fatalf("write inner: %v", err)
	}

	t.Setenv("GOTMPL_PRELOAD", outerPath+preloadSeparator+innerPath)

	stdin := "" +
		"Report:\n" + // line 1
		"{{ include \"outer\" . }}\n" // line 2 - calls outer

	_, err := runTemplate(t, stdin)
	if err == nil {
		t.Fatal("expected error but got none")
	}
	msg := err.Error()

	lines := strings.Split(msg, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 error lines (stdin -> outer -> inner), got %d:\n%s", len(lines), msg)
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, "template: ") {
			t.Errorf("line %d must start with %q, got: %s", i, "template: ", line)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	relOuter, err := filepath.Rel(cwd, outerPath)
	if err != nil {
		t.Fatalf("rel outer: %v", err)
	}
	relInner, err := filepath.Rel(cwd, innerPath)
	if err != nil {
		t.Fatalf("rel inner: %v", err)
	}

	// Frame 1: STDIN calls include "outer" on line 2
	wantLine1 := []string{
		`template: STDIN:2:`,
		`executing "STDIN" at <include "outer" .>`,
		`error calling include`,
	}
	// Frame 2: outer.tmpl calls include "inner" on line 2
	wantLine2 := []string{
		`template: ` + relOuter + `:2:`,
		`executing "outer" at <include "inner" .>`,
		`error calling include`,
	}
	// Frame 3: inner.tmpl fails on <.missing> at line 2
	wantLine3 := []string{
		`template: ` + relInner + `:2:`,
		`executing "inner" at <.missing>`,
		`map has no entry for key "missing"`,
	}

	for _, frame := range []struct {
		idx  int
		want []string
	}{{0, wantLine1}, {1, wantLine2}, {2, wantLine3}} {
		for _, want := range frame.want {
			if !strings.Contains(lines[frame.idx], want) {
				t.Errorf("line %d missing %q; got: %s", frame.idx+1, want, lines[frame.idx])
			}
		}
	}
}

// pins the no-op behaviour for errors that don't have Go's template chain wrapper (preload errors, yaml parse errors,
// etc). They stay single-line and unchanged
func TestFormatTemplateErrorLeavesNonTemplateErrorsAlone(t *testing.T) {
	orig := errors.New("error reading data file foo.yaml: open foo.yaml: no such file or directory")
	got := formatTemplateError(orig)
	if got != orig {
		t.Errorf("expected identical error (no wrapper found); got a different value")
	}
	if strings.Contains(got.Error(), "\n") {
		t.Errorf("no wrapper present, error must remain single-line; got:\n%s", got.Error())
	}
}

// verifies that a 3-frame chain (include-within-include) also splits cleanly
func TestFormatTemplateErrorHandlesDeepChain(t *testing.T) {
	flat := errors.New(
		`template: STDIN:1:1: at <include "b" .>: error calling include: ` +
			`template: b.tmpl:2:2: at <include "c" .>: error calling include: ` +
			`template: c.tmpl:3:3: at <.x>: map has no entry for key "x"`,
	)
	got := formatTemplateError(flat).Error()
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines for a 3-frame chain, got %d:\n%s", len(lines), got)
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, "template: ") {
			t.Errorf("line %d must start with %q, got: %s", i, "template: ", line)
		}
	}
}

func TestPreloadTemplatesFileNotFound(t *testing.T) {
	t.Setenv("GOTMPL_PRELOAD", "/nonexistent/file.tmpl")

	_, err := runTemplate(t, `{{ "test" }}`)
	if err == nil {
		t.Errorf("run() expected error when preload file doesn't exist, but got none")
		return
	}

	// Verify it's a PreloadError (via errors.As so wrapped errors still work)
	var pe *PreloadError
	if !errors.As(err, &pe) {
		t.Errorf("run() expected PreloadError, got %T", err)
	}

	// Verify error message contains the file name
	if !strings.Contains(err.Error(), "/nonexistent/file.tmpl") {
		t.Errorf("run() error message should contain file name, got: %v", err)
	}
}

func TestHelmFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "toYaml on string strips trailing newline",
			template: `[{{ "hello" | toYaml }}]`,
			want:     "[hello]",
		},
		{
			name:     "fromYaml then toYaml roundtrip",
			template: `{{ "a: 1\nb: 2" | fromYaml | toYaml }}`,
			want:     "a: 1\nb: 2",
		},
		{
			name:     "nindent adds leading newline and indents",
			template: `x:{{ "hello" | nindent 2 }}`,
			want:     "x:\n  hello",
		},
		{
			name:     "indent has no leading newline",
			template: `{{ "hello" | indent 4 }}`,
			want:     "    hello",
		},
		{
			name:     "required passes non-empty value through",
			template: `{{ required "x must be set" "y" }}`,
			want:     "y",
		},
		{
			name:     "required errors on empty string",
			template: `{{ required "x must be set" "" }}`,
			wantErr:  true,
		},
		{
			name:     "required errors on nil value",
			template: "{{ required \"x must be set\" .nilval }}\n{{/* __DATA__\nnilval: null\n*/}}",
			wantErr:  true,
		},
		{
			name:     "include executes named template",
			template: `{{- define "greet" -}}Hi {{ . }}{{- end -}}{{ include "greet" "Bob" }}`,
			want:     "Hi Bob",
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

func TestRunMultipleDataFilesMerge(t *testing.T) {
	// Verifies deep-merge semantics across multiple data files: later files override earlier ones, but nested keys not
	// present in the later file are preserved from the earlier file
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	override := filepath.Join(dir, "override.yaml")

	baseContent := "name: base\nreplicas: 3\nconfig:\n  timeout: 30\n  debug: false\n"
	if err := os.WriteFile(base, []byte(baseContent), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}

	overrideContent := "replicas: 5\nconfig:\n  debug: true\n  cache: enabled\n"
	if err := os.WriteFile(override, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	template := `{{ .name }}|{{ .replicas }}|{{ .config.timeout }}|{{ .config.debug }}|{{ .config.cache }}`
	got, err := runTemplate(t, template, base, override)
	if err != nil {
		t.Fatalf("run() failed: %v", err)
	}

	// name: only in base → preserved
	// replicas: in both → override wins
	// config.timeout: only in base → preserved (deep merge)
	// config.debug: in both → override wins
	// config.cache: only in override → added
	want := "base|5|30|true|enabled"
	if got != want {
		t.Errorf("run() got output %q, want %q", got, want)
	}
}

// The tests below are integration tests for GOTMPL_FUNCTIONS handling: they verify that examples/functions.yaml and
// test/fixtures/custom-functions/* work end-to-end through cli.Run. Unit-level custom-function behavior is tested in
// pkg/functions/custom; unit-level uuid behavior in pkg/functions/additional/uuid. These tests exercise the pipeline
// that composes them

func TestCustomFunctions(t *testing.T) {
	root := projectRoot(t)
	t.Setenv("GOTMPL_FUNCTIONS", root+"/examples/functions.yaml")
	t.Setenv("GOTMPL_PRELOAD", "")

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{"toHarnessId function", `{{ "my-service_name" | toHarnessId }}`, "my_service__name", false},
		{"toHarnessId with complex input", `{{ "foo-bar_baz@123" | toHarnessId }}`, "foo_bar__baz_123", false},
		{"shout function", `{{ "hello world" | shout }}`, "HELLO WORLD", false},
		{"withPrefix function", `{{ "myvalue" | withPrefix }}`, "prefix_myvalue", false},
		{"slugify function", `{{ "Hello World! 123" | slugify }}`, "hello-world-123", false},
		{"multiple custom functions", `{{ "test" | withPrefix | shout }}`, "PREFIX_TEST", false},
		{"custom function with embedded data", "{{ .name | toHarnessId }}\n{{/* __DATA__\nname: my-app_v2\n*/}}", "my_app__v2\n", false},
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
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCustomFunctionsNotFound(t *testing.T) {
	// No config file resolvable: cli.Run must still render successfully
	t.Setenv("XDG_CONFIG_HOME", "/tmp/nonexistent-gotmpl2text-test")
	t.Setenv("GOTMPL_FUNCTIONS", "")
	t.Setenv("GOTMPL_PRELOAD", "")

	got, err := runTemplate(t, `{{ "test" | upper }}`)
	if err != nil {
		t.Fatalf("run() failed when custom functions file doesn't exist: %v", err)
	}
	if got != "TEST" {
		t.Errorf("got %q, want %q", got, "TEST")
	}
}

func TestCustomFunctionTypes(t *testing.T) {
	root := projectRoot(t)
	t.Setenv("GOTMPL_FUNCTIONS", root+"/test/fixtures/custom-functions/typed-functions.yaml")
	t.Setenv("GOTMPL_PRELOAD", "")

	tests := []struct {
		name     string
		template string
		wantOut  string
	}{
		{"string type explicit", `{{ "hello" | toUpperStr }}`, "HELLO"},
		{"string type implicit default", `{{ "WORLD" | toLowerStr }}`, "world"},
		{"string preserves whitespace", `{{ "  spaced  " | toLowerStr }}`, "  spaced  "},
		{"int64 type returns integer", `{{ $len := "hello" | getLength }}{{ if gt $len 3 }}pass{{ end }}`, "pass"},
		{"int64 arithmetic", `{{ $len := "test" | getLength }}{{ add $len 10 }}`, "14"},
		{"bool type in conditional", `{{ if "hello" | isLong }}yes{{ else }}no{{ end }}`, "no"},
		{"bool type true", `{{ if "hello world" | isLong }}yes{{ else }}no{{ end }}`, "yes"},
		{"float64 type", `{{ $half := 10 | halfValue }}{{ printf "%.1f" $half }}`, "5.0"},
		{"float64 comparison", `{{ $val := 10 | halfValue }}{{ if gt $val 4.0 }}pass{{ end }}`, "pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template)
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
	root := projectRoot(t)
	t.Setenv("GOTMPL_PRELOAD", "")

	tests := []struct {
		name       string
		yamlFile   string
		template   string
		wantErrMsg string
	}{
		{
			name:       "unknown type",
			yamlFile:   root + "/test/fixtures/custom-functions/error-unknown-type.yaml",
			template:   `{{ 1 | badType }}`,
			wantErrMsg: `unknown type "uint32" for custom function "badType"`,
		},
		{
			name:       "int64 conversion error",
			yamlFile:   root + "/test/fixtures/custom-functions/error-int64-conversion.yaml",
			template:   `{{ "not-a-number" | toInt }}`,
			wantErrMsg: `custom function "toInt": cannot convert "not-a-number" to int64`,
		},
		{
			name:       "float64 conversion error",
			yamlFile:   root + "/test/fixtures/custom-functions/error-float64-conversion.yaml",
			template:   `{{ "invalid" | toFloat }}`,
			wantErrMsg: `custom function "toFloat": cannot convert "invalid" to float64`,
		},
		{
			name:       "bool conversion error",
			yamlFile:   root + "/test/fixtures/custom-functions/error-bool-conversion.yaml",
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

func TestParseRunArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantHelm     bool
		wantFiles    []dataFile
		wantErrMatch string
	}{
		{
			name:      "no flags, all positional",
			args:      []string{"a.yaml", "b.yaml"},
			wantFiles: []dataFile{{path: "a.yaml"}, {path: "b.yaml"}},
		},
		{
			name:      "long form space-separated wraps files after it",
			args:      []string{"--wrap", "Values", "chart/values.yaml"},
			wantFiles: []dataFile{{path: "chart/values.yaml", wrapKey: "Values"}},
		},
		{
			name:      "short form space-separated",
			args:      []string{"-w", "Values", "chart/values.yaml"},
			wantFiles: []dataFile{{path: "chart/values.yaml", wrapKey: "Values"}},
		},
		{
			name:      "long form with equals",
			args:      []string{"--wrap=Values", "a.yaml"},
			wantFiles: []dataFile{{path: "a.yaml", wrapKey: "Values"}},
		},
		{
			name:      "short form with equals",
			args:      []string{"-w=Values", "a.yaml"},
			wantFiles: []dataFile{{path: "a.yaml", wrapKey: "Values"}},
		},
		{
			name:      "wrap positional: files before wrap unaffected, files after wrapped",
			args:      []string{"a.yaml", "--wrap", "Values", "b.yaml", "c.yaml"},
			wantFiles: []dataFile{{path: "a.yaml"}, {path: "b.yaml", wrapKey: "Values"}, {path: "c.yaml", wrapKey: "Values"}},
		},
		{
			name:      "wrap positional: two --wrap keys apply to different files",
			args:      []string{"--wrap", "A", "a.yaml", "--wrap", "B", "b.yaml"},
			wantFiles: []dataFile{{path: "a.yaml", wrapKey: "A"}, {path: "b.yaml", wrapKey: "B"}},
		},
		{
			name:     "helm flag alone",
			args:     []string{"--helm"},
			wantHelm: true,
		},
		{
			name:      "helm plus positional wrap plus files",
			args:      []string{"--helm", "ctx.yaml", "--wrap", "Values", "values.yaml"},
			wantHelm:  true,
			wantFiles: []dataFile{{path: "ctx.yaml"}, {path: "values.yaml", wrapKey: "Values"}},
		},
		{
			name:         "missing value after --wrap",
			args:         []string{"--wrap"},
			wantErrMatch: "--wrap requires a key argument",
		},
		{
			name:         "missing value after -w",
			args:         []string{"a.yaml", "-w"},
			wantErrMatch: "-w requires a key argument",
		},
		{
			name:         "empty value in --wrap=",
			args:         []string{"--wrap="},
			wantErrMatch: "--wrap= requires a non-empty key argument",
		},
		{
			name:         "empty value in -w=",
			args:         []string{"-w="},
			wantErrMatch: "-w= requires a non-empty key argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helm, files, err := parseRunArgs(tt.args)
			if tt.wantErrMatch != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrMatch)
				}
				if !strings.Contains(err.Error(), tt.wantErrMatch) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrMatch)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if helm != tt.wantHelm {
				t.Errorf("helm = %v, want %v", helm, tt.wantHelm)
			}
			if !reflect.DeepEqual(files, tt.wantFiles) {
				t.Errorf("files = %+v, want %+v", files, tt.wantFiles)
			}
		})
	}
}

func TestRunWrap(t *testing.T) {
	// The wrap flag lifts each data file's content under a top-level key so template snippets that reference
	// `.Values.foo` can be tested against a chart's flat values.yaml with no pre-processing. Embedded __DATA__ blocks
	// must NOT be wrapped: they are the template author's own data, not external values
	dir := t.TempDir()
	valuesPath := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte("name: api\nreplicas: 3\n"), 0644); err != nil {
		t.Fatalf("write values: %v", err)
	}
	overridePath := filepath.Join(dir, "override.yaml")
	if err := os.WriteFile(overridePath, []byte("replicas: 5\n"), 0644); err != nil {
		t.Fatalf("write override: %v", err)
	}
	emptyPath := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	// context file: unwrapped, would be placed BEFORE --wrap on the command line so it stays at the root
	contextPath := filepath.Join(dir, "context.yaml")
	if err := os.WriteFile(contextPath, []byte("Release:\n  Name: custom-release\n"), 0644); err != nil {
		t.Fatalf("write context: %v", err)
	}
	// pre-wrapped: file already has "Values:" at top. With --wrap Values applied, this gets DOUBLE-wrapped now (the
	// no-double-wrap escape hatch is gone; user expresses intent positionally instead)
	preWrappedPath := filepath.Join(dir, "prewrapped.yaml")
	if err := os.WriteFile(preWrappedPath, []byte("Values:\n  replicas: 7\n"), 0644); err != nil {
		t.Fatalf("write prewrapped: %v", err)
	}
	// two flat files with distinct top-level keys, used to test that two --wrap flags in one command apply to
	// different files
	aFile := filepath.Join(dir, "a.yaml")
	if err := os.WriteFile(aFile, []byte("x: 1\n"), 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	bFile := filepath.Join(dir, "b.yaml")
	if err := os.WriteFile(bFile, []byte("y: 2\n"), 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	tests := []struct {
		name     string
		template string
		args     []string
		want     string
		wantErr  bool
	}{
		{
			name:     "wrap single file under Values",
			template: `{{ .Values.name }}:{{ .Values.replicas }}`,
			args:     []string{"--wrap", "Values", valuesPath},
			want:     "api:3",
		},
		{
			name:     "wrap short flag",
			template: `{{ .Values.name }}:{{ .Values.replicas }}`,
			args:     []string{"-w", "Values", valuesPath},
			want:     "api:3",
		},
		{
			name:     "wrap with equals form",
			template: `{{ .Values.name }}:{{ .Values.replicas }}`,
			args:     []string{"--wrap=Values", valuesPath},
			want:     "api:3",
		},
		{
			name:     "wrap applies to every file and deep-merges",
			template: `{{ .Values.name }}:{{ .Values.replicas }}`,
			args:     []string{"--wrap", "Values", valuesPath, overridePath},
			want:     "api:5",
		},
		{
			name:     "no wrap without flag leaves data at root",
			template: `{{ .name }}:{{ .replicas }}`,
			args:     []string{valuesPath},
			want:     "api:3",
		},
		{
			name:     "wrap does not touch embedded __DATA__ blocks",
			template: "{{ .Values.name }}:{{ .kind }}\n{{/* __DATA__\nkind: Deployment\n*/}}",
			args:     []string{"--wrap", "Values", valuesPath},
			want:     "api:Deployment\n",
		},
		{
			name:     "empty data file with wrap does not clobber earlier data",
			template: `{{ .Values.name }}:{{ .Values.replicas }}`,
			args:     []string{"--wrap", "Values", valuesPath, emptyPath},
			want:     "api:3",
		},
		{
			name:     "wrap with missing key errors when template asks for unwrapped path",
			template: `{{ .name }}`,
			args:     []string{"--wrap", "Values", valuesPath},
			wantErr:  true,
		},
		{
			// files before --wrap stay unwrapped, files after are wrapped. This is the canonical way to mix a
			// hand-written context file (Release: at root) with a flat chart values.yaml
			name:     "positional wrap: unwrapped file precedes wrapped file",
			template: `{{ .Release.Name }}:{{ .Values.name }}`,
			args:     []string{contextPath, "--wrap", "Values", valuesPath},
			want:     "custom-release:api",
		},
		{
			// two --wrap flags in one invocation, applied to different files
			name:     "positional wrap: two distinct wrap keys",
			template: `{{ .A.x }}:{{ .B.y }}`,
			args:     []string{"--wrap", "A", aFile, "--wrap", "B", bFile},
			want:     "1:2",
		},
		{
			// pre-wrapped file with "Values:" at top, run under --wrap Values. Double-wrap is honored now: the parsed
			// content becomes {Values: {Values: {replicas: 7}}}. Users who don't want this write their invocation as
			// positional wrap (put the pre-wrapped file BEFORE --wrap)
			name:     "double wrap is honored (no escape hatch); template must dig through both levels",
			template: `{{ .Values.Values.replicas }}`,
			args:     []string{"--wrap", "Values", preWrappedPath},
			want:     "7",
		},
		{
			// same pre-wrapped file, but placed BEFORE --wrap on the command line: stays at root, no double-wrap
			name:     "pre-wrapped file placed before --wrap stays single-level",
			template: `{{ .Values.replicas }}`,
			args:     []string{preWrappedPath, "--wrap", "Values", emptyPath},
			want:     "7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template, tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (output=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunHelm(t *testing.T) {
	// --helm injects dummy Release/Chart/Capabilities/Template context as a defaults layer. Templates that only
	// reference these standard Helm objects at the scalar/map level work without any data file. Downstream data
	// (data files, embedded __DATA__) can override individual fields; method-shaped accessors (`.Files.Get`,
	// `.Capabilities.APIVersions.Has`) are intentionally not supported and errors from them are the expected signal
	// that the template needs `helm template` proper
	dir := t.TempDir()
	releaseOverride := filepath.Join(dir, "release-override.yaml")
	if err := os.WriteFile(releaseOverride, []byte("Release:\n  Name: my-release\n"), 0644); err != nil {
		t.Fatalf("write release-override: %v", err)
	}
	valuesFile := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(valuesFile, []byte("name: api\nreplicas: 3\n"), 0644); err != nil {
		t.Fatalf("write values: %v", err)
	}

	tests := []struct {
		name     string
		template string
		args     []string
		want     string
		wantErr  bool
	}{
		{
			name:     "Release.Name dummy",
			template: `{{ .Release.Name }}`,
			args:     []string{"--helm"},
			want:     "release-name",
		},
		{
			name:     "Release.Namespace dummy",
			template: `{{ .Release.Namespace }}`,
			args:     []string{"--helm"},
			want:     "release-namespace",
		},
		{
			name:     "Chart.Version dummy",
			template: `{{ .Chart.Version }}`,
			args:     []string{"--helm"},
			want:     "0.0.0-dummy",
		},
		{
			name:     "Chart.Name dummy",
			template: `{{ .Chart.Name }}`,
			args:     []string{"--helm"},
			want:     "chart-name",
		},
		{
			name:     "Capabilities.KubeVersion.Major dummy",
			template: `{{ .Capabilities.KubeVersion.Major }}`,
			args:     []string{"--helm"},
			want:     "1",
		},
		{
			name:     "Template.BasePath dummy",
			template: `{{ .Template.BasePath }}`,
			args:     []string{"--helm"},
			want:     "chart-name/templates",
		},
		{
			// dummies are DEFAULTS: user file overrides individual fields via deep-merge, other dummies survive
			name:     "user data file overrides Release.Name, Chart.Version dummy still visible",
			template: `{{ .Release.Name }}:{{ .Chart.Version }}`,
			args:     []string{"--helm", releaseOverride},
			want:     "my-release:0.0.0-dummy",
		},
		{
			// --helm plays with positional --wrap: unwrapped override file first, then --wrap Values for values.yaml
			name:     "helm + positional wrap: mixed override and wrapped values",
			template: `{{ .Release.Name }}:{{ .Values.name }}:{{ .Values.replicas }}`,
			args:     []string{"--helm", releaseOverride, "--wrap", "Values", valuesFile},
			want:     "my-release:api:3",
		},
		{
			// without --helm, .Release.Name is a missing key
			name:     "no --helm means Release is missing",
			template: `{{ .Release.Name }}`,
			args:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template, tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (output=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUUIDv7DerivedFunctions verifies that examples/functions.yaml correctly layers custom functions on top of our base
// uuid functions - a full-pipeline check that the documented example config actually works
func TestUUIDv7DerivedFunctions(t *testing.T) {
	root := projectRoot(t)
	t.Setenv("GOTMPL_FUNCTIONS", root+"/examples/functions.yaml")
	t.Setenv("GOTMPL_PRELOAD", "")

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{"uuidv7ToEpochMs converts nanoseconds to milliseconds", `{{ "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpochMs }}`, "1778593723465"},
		{"uuidv7ToEpoch converts nanoseconds to seconds", `{{ "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpoch }}`, "1778593723"},
		{"milliseconds to seconds conversion is consistent", `{{ $ms := "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpochMs }}{{ $sec := "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpoch }}{{ eq (div $ms 1000) $sec }}`, "true"},
		{"uuidv7 pipeline to milliseconds", `{{ $ms := uuidv7 | uuidv7ToEpochMs }}{{ if gt $ms 1000000000000 }}PASS{{ end }}`, "PASS"},
		{"uuidv7 pipeline to seconds", `{{ $sec := uuidv7 | uuidv7ToEpoch }}{{ if gt $sec 1000000000 }}PASS{{ end }}`, "PASS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runTemplate(t, tt.template)
			if err != nil {
				t.Fatalf("run() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
