package main

import (
	"strings"
	"testing"
)

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
