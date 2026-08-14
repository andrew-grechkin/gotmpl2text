// Package dict provides dotted-path helpers over nested map values: valueAt (fetch), existsAt (present, possibly nil),
// definedAt (present and non-nil)
package dict

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"text/template"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"valueAt":     valueAt,
		"existsAt":    existsAt,
		"definedAt":   definedAt,
		"toEntries":   toEntries,
		"fromEntries": fromEntries,
	}
}

// walks a dotted path through nested map values
//
// Signature is variadic so the function is pipeline-friendly. Go template
// pipelines feed the piped value as the LAST argument, so both of these work:
//
//	{{ .data | valueAt "a.b.c" }}           -> valueAt("a.b.c", .data)             args=[.data]
//	{{ .data | valueAt "a.b.c" "default" }} -> valueAt("a.b.c", "default", .data)  args=["default", .data]
//	{{ valueAt "a.b.c" .data }}                                                    args=[.data]
//	{{ valueAt "a.b.c" "default" .data }}                                          args=["default", .data]
//
// Returns nil when any segment is missing (or the default if provided).
// A present-but-nil value returns nil (not the default) - use definedAt to
// distinguish "missing" from "present-but-nil".
//
// Errors loudly if called with the wrong number of arguments so template
// authors see the mistake instead of a silent nil
func valueAt(path string, args ...any) (any, error) {
	var def, data any
	switch len(args) {
	case 1:
		data = args[0]
	case 2:
		def, data = args[0], args[1]
	default:
		return nil, fmt.Errorf("valueAt: expected 2 or 3 arguments, got %d", len(args)+1)
	}
	v, ok := walk(path, data)
	if !ok {
		return def, nil
	}
	return v, nil
}

// reports whether path can be resolved in data. A present-but-nil
// value counts as existing - only truly missing segments return false
//
//	{{ if .data | existsAt "server.tls" }}...{{ end }}
func existsAt(path string, data any) bool {
	_, ok := walk(path, data)
	return ok
}

// reports whether path exists AND the value at path is not nil
//
//	{{ if .data | definedAt "server.tls.host" }}...{{ end }}
func definedAt(path string, data any) bool {
	v, ok := walk(path, data)
	return ok && v != nil
}

// resolves a dotted path against nested maps. Returns (value, true) if every segment resolves; (nil, false) if any
// segment is missing or any intermediate value isn't a map
func walk(path string, data any) (any, bool) {
	cur := data
	for seg := range strings.SplitSeq(path, ".") {
		v, ok := lookup(cur, seg)
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// converts a map to a slice of {"key": ..., "value": ...} maps. Behavior matches jq's to_entries
//
// Keys are sorted lexicographically for deterministic output - map iteration in Go is randomized, so without sorting
// the same input would produce different bytes across runs, breaking diffs and reproducible config generation
func toEntries(m any) ([]map[string]any, error) {
	rv := reflect.ValueOf(m)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil, fmt.Errorf("toEntries: expected map, got %T", m)
	}
	keys := rv.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	result := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		result = append(result, map[string]any{
			"key":   k.Interface(),
			"value": rv.MapIndex(k).Interface(),
		})
	}
	return result, nil
}

// inverse of toEntries: converts a slice of {"key": ..., "value": ...} records back into a map. Matches jq's
// from_entries (name in Go camelCase)
//
// A missing "value" field is treated as nil (matches jq). "key" must be a string. Errors on non-slice input, non-map
// entries, or non-string keys
func fromEntries(v any) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, fmt.Errorf("fromEntries: expected array or slice, got %T", v)
	}
	result := make(map[string]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		entry := rv.Index(i).Interface()
		if k := reflect.ValueOf(entry).Kind(); k != reflect.Map {
			return nil, fmt.Errorf("fromEntries[%d]: expected map, got %T", i, entry)
		}
		keyRaw, ok := lookup(entry, "key")
		if !ok {
			return nil, fmt.Errorf("fromEntries[%d]: missing 'key' field", i)
		}
		keyStr, ok := keyRaw.(string)
		if !ok {
			return nil, fmt.Errorf("fromEntries[%d]: 'key' must be a string, got %T", i, keyRaw)
		}
		val, _ := lookup(entry, "value") // missing "value" -> nil
		result[keyStr] = val
	}
	return result, nil
}

// returns m[key] handling the two map shapes that come out of YAML/JSON unmarshal into interface{}
func lookup(m any, key string) (any, bool) {
	switch mm := m.(type) {
	case map[string]any:
		v, ok := mm[key]
		return v, ok
	case map[any]any:
		v, ok := mm[key]
		return v, ok
	}
	return nil, false
}
