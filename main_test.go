package main

import "testing"

func TestEmbeds(t *testing.T) {
	if len(helpText) == 0 {
		t.Error("helpText is empty; help.txt was not embedded")
	}
	if len(readmeContent) == 0 {
		t.Error("readmeContent is empty; README.md was not embedded")
	}
}
