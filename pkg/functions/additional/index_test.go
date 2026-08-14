package additional

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/andrew-grechkin/gotmpl2text/pkg/functions/helm"
)

// guards against two subpackages accidentally registering the same template function name. maps.Copy in FuncMap would
// silently let the later package win, which would be a very hard bug to notice from a rendered template
func TestFuncMapNamesUnique(t *testing.T) {
	owner := map[string]string{}
	for _, sub := range subFuncMaps {
		pkg := funcPackage(sub)
		for name := range sub() {
			if prev, dup := owner[name]; dup {
				t.Errorf("function %q registered by both %q and %q", name, prev, pkg)
				continue
			}
			owner[name] = pkg
		}
	}
}

func TestAggregateContainsAllSubpackageFuncs(t *testing.T) {
	all := FuncMap()
	for _, sub := range subFuncMaps {
		for name := range sub() {
			if _, ok := all[name]; !ok {
				t.Errorf("aggregate is missing %q from %s", name, funcPackage(sub))
			}
		}
	}
}

// accidental shadowing of helm/sprig functions by additional/* packages. Two failure modes:
//
//   - A name registered in additional.FuncMap that also exists in helm.FuncMap() but is NOT in ExpectedOverrides: the
//   additional package is silently overriding an existing function, likely a mistake. Either rename the new function or
//   add its name to ExpectedOverrides with a comment explaining the intent
//
//   - A name in ExpectedOverrides that does NOT exist in helm.FuncMap(): the allowlist is stale, the override no longer
//   collides. Remove the entry
func TestNoUnexpectedOverrides(t *testing.T) {
	helmFm := helm.FuncMap()
	addFm := FuncMap()

	for name := range addFm {
		if _, shadows := helmFm[name]; !shadows {
			continue
		}
		if _, allowed := ExpectedOverrides[name]; !allowed {
			t.Errorf("%s registers %q which shadows a helm/sprig function; either rename or add to ExpectedOverrides with a comment", ownerPackage(name), name)
		}
	}

	for name := range ExpectedOverrides {
		if _, shadows := helmFm[name]; !shadows {
			t.Errorf("ExpectedOverrides[%q] is stale - no such function in helm.FuncMap() to override; remove the entry", name)
		}
	}
}

// returns the additional/* subpackage that registered name, for use in error messages. Walks the same subFuncMaps the
// aggregate does
func ownerPackage(name string) string {
	for _, sub := range subFuncMaps {
		if _, ok := sub()[name]; ok {
			return funcPackage(sub)
		}
	}
	return "<unknown>"
}

// returns the import path of the package that defined fn, so uniqueness errors can point at the culprit. Uses
// reflect+runtime to fish the fully-qualified name from the function pointer
func funcPackage(fn func() template.FuncMap) string {
	pc := reflect.ValueOf(fn).Pointer()
	name := runtime.FuncForPC(pc).Name()
	// e.g. "github.com/.../datetime.FuncMap" -> "github.com/.../datetime"
	if i := strings.LastIndex(name, "."); i > 0 {
		return name[:i]
	}
	return name
}
