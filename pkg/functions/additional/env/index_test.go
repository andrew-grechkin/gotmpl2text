package env

import "testing"

func TestGetenvSet(t *testing.T) {
	t.Setenv("GOTMPL_TEST_VAR", "hello")
	if got := getenv("GOTMPL_TEST_VAR"); got != "hello" {
		t.Errorf("expected \"hello\", got %v", got)
	}
}

func TestGetenvSetToEmptyReturnsEmptyString(t *testing.T) {
	// A var set to "" is distinguishable from unset - must return "" as a real string value, and any default must be
	// ignored (matches shell ${VAR-default}, not ${VAR:-default})
	t.Setenv("GOTMPL_TEST_VAR", "")
	if got := getenv("GOTMPL_TEST_VAR", "fallback"); got != "" {
		t.Errorf("expected \"\" (set-to-empty beats default), got %v (%T)", got, got)
	}
}

func TestGetenvUnsetNoDefaultReturnsNil(t *testing.T) {
	if got := getenv("GOTMPL_TEST_DEFINITELY_UNSET_VAR_XYZ"); got != nil {
		t.Errorf("expected nil for unset var, got %v (%T)", got, got)
	}
}

func TestGetenvUnsetWithDefault(t *testing.T) {
	if got := getenv("GOTMPL_TEST_DEFINITELY_UNSET_VAR_XYZ", "fallback"); got != "fallback" {
		t.Errorf("expected \"fallback\", got %v", got)
	}
}
