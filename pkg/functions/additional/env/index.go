// Package env exposes environment-variable access. Unlike sprig's `env`, unset variables return nil (not "") so callers
// can distinguish "not set" from "set to empty string"
package env

import (
	"os"
	"text/template"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"getenv": getenv,
	}
}

// returns the value of environment variable name
//
//   - Set (including empty string) -> the value as string
//   - Unset, default provided      -> the default string
//   - Unset, no default            -> nil
//
// Returning nil (rather than "") on unset lets templates distinguish "not set at all" from "set to empty", and composes
// with sprig's `default` or an explicit `if` check. Caveat: nil renders as "<no value>" when concatenated with a string
// in default missingkey mode - pass an explicit "" default if you want empty-string fallback for missing vars
func getenv(name string, def ...string) any {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return nil
}
