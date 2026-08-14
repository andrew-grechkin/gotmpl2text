// Package additional aggregates every subpackage's FuncMap into one call so binaries can wire the full "additional" set
// with a single line.
//
// Each subpackage owns its topic (datetime, dict, env, host, predicate, proc, regex, text, uuid); this file only
// composes them and does not add functions of its own
package additional

import (
	"maps"
	"text/template"

	"github.com/andrew-grechkin/gotmpl2text/pkg/functions/additional/uuid"
)

// lists template function names that this package intentionally shadows once composed after helm.FuncMap() in the
// render pipeline. Every name here MUST actually exist in helm.FuncMap() (else the allowlist is stale) and any name in
// this package NOT here that ALSO exists in helm.FuncMap() is an accidental shadow that must be renamed or explicitly
// added here. TestNoUnexpectedOverrides enforces both halves
var ExpectedOverrides = map[string]struct{}{
}

// lists every topic's FuncMap constructor in a stable order so tests can iterate them. Later maps override earlier ones
// on name collision, which is what TestFuncMapNamesUnique guards against
var subFuncMaps = []func() template.FuncMap{
	uuid.FuncMap,
}

// FuncMap returns the union of every subpackage's FuncMap
func FuncMap() template.FuncMap {
	fm := template.FuncMap{}
	for _, sub := range subFuncMaps {
		maps.Copy(fm, sub())
	}
	return fm
}
