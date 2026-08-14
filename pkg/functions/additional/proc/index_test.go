package proc

import (
	"os"
	"testing"
)

func TestFuncMapRegistered(t *testing.T) {
	fm := FuncMap()
	for _, name := range []string{"uid", "gid", "cwd"} {
		if _, ok := fm[name]; !ok {
			t.Errorf("expected %q registered", name)
		}
	}
}

func TestCwdMatchesOSGetwd(t *testing.T) {
	got, err := FuncMap()["cwd"].(func() (string, error))()
	if err != nil {
		t.Fatalf("cwd error: %v", err)
	}
	want, _ := os.Getwd()
	if got != want {
		t.Errorf("cwd() = %q, want %q", got, want)
	}
}
