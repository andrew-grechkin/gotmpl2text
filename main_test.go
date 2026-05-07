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
			wantTmpl:   "Hello\n\nWorld\n",
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
			content:    "Hello\n{{/*   __DATA__  \nspaced\n*/}}",
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
