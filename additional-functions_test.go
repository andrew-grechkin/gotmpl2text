package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestUUIDv7ToEpochNs(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "valid UUID v7",
			template: `{{ "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpochNs }}`,
			want:     "1778593723465000000",
			wantErr:  false,
		},
		{
			name:     "generated UUID v7 has valid timestamp",
			template: `{{ $uuid := uuidv7 }}{{ $ns := $uuid | uuidv7ToEpochNs }}{{ if gt $ns 1000000000000000000 }}PASS{{ end }}`,
			want:     "PASS",
			wantErr:  false,
		},
		{
			name:     "invalid UUID",
			template: `{{ 42 | uuidv7ToEpochNs }}`,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "empty string",
			template: `{{ "" | uuidv7ToEpochNs }}`,
			want:     "",
			wantErr:  true,
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

func TestUUIDv7DerivedFunctions(t *testing.T) {
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
			name:     "uuidv7ToEpochMs converts nanoseconds to milliseconds",
			template: `{{ "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpochMs }}`,
			want:     "1778593723465",
			wantErr:  false,
		},
		{
			name:     "uuidv7ToEpoch converts nanoseconds to seconds",
			template: `{{ "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpoch }}`,
			want:     "1778593723",
			wantErr:  false,
		},
		{
			name:     "milliseconds to seconds conversion is consistent",
			template: `{{ $ms := "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpochMs }}{{ $sec := "019e1c72-7449-7195-b65b-b7c7f94ed77e" | uuidv7ToEpoch }}{{ eq (div $ms 1000) $sec }}`,
			want:     "true",
			wantErr:  false,
		},
		{
			name:     "uuidv7 pipeline to milliseconds",
			template: `{{ $ms := uuidv7 | uuidv7ToEpochMs }}{{ if gt $ms 1000000000000 }}PASS{{ end }}`,
			want:     "PASS",
			wantErr:  false,
		},
		{
			name:     "uuidv7 pipeline to seconds",
			template: `{{ $sec := uuidv7 | uuidv7ToEpoch }}{{ if gt $sec 1000000000 }}PASS{{ end }}`,
			want:     "PASS",
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
