package host

import (
	"runtime"
	"testing"
)

func TestFuncMapRegistered(t *testing.T) {
	fm := FuncMap()
	for _, name := range []string{"hostname", "os", "arch"} {
		if _, ok := fm[name]; !ok {
			t.Errorf("expected %q registered", name)
		}
	}
}

func TestOSAndArchMatchRuntime(t *testing.T) {
	fm := FuncMap()
	if got := fm["os"].(func() string)(); got != runtime.GOOS {
		t.Errorf("os() = %q, want %q", got, runtime.GOOS)
	}
	if got := fm["arch"].(func() string)(); got != runtime.GOARCH {
		t.Errorf("arch() = %q, want %q", got, runtime.GOARCH)
	}
}

func TestHostnameNonEmpty(t *testing.T) {
	h, err := FuncMap()["hostname"].(func() (string, error))()
	if err != nil {
		t.Fatalf("hostname error: %v", err)
	}
	if h == "" {
		t.Error("hostname is empty")
	}
}
