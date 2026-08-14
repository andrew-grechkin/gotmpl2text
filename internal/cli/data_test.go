package cli

import (
	"reflect"
	"testing"
)

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
