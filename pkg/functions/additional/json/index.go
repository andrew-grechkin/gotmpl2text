// Package json overrides sprig's silent JSON serialization functions (toJson, toPrettyJson, toRawJson, fromJson) with
// variants that halt template execution on marshal/unmarshal errors
//
// Rationale: a serialization failure has no meaningful continuation - either there's a hole in the output (failed
// marshal) or the "parsed" value is a map-with-Error-key that downstream field accesses can't distinguish from
// legitimate data (failed unmarshal). Rendering must stop at the point of the failure so the author sees exactly what
// went wrong. Sprig's must* variants were already loud; we're making that the default
package json

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"toJson":       toJSON,
		"toPrettyJson": toPrettyJSON,
		"toRawJson":    toRawJSON,
		"toJsonL":      toJSONL,
		"fromJson":     fromJSON,
	}
}

// toJSON marshals v to compact JSON
func toJSON(v any) (string, error) {
	data, err := stdjson.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("toJson: %w", err)
	}
	return string(data), nil
}

// marshals v with a 2-space indent (matches sprig's default)
func toPrettyJSON(v any) (string, error) {
	data, err := stdjson.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("toPrettyJson: %w", err)
	}
	return string(data), nil
}

// marshals v without HTML-escaping (< > & stay as-is), matching sprig's SetEscapeHTML(false) behavior. Encoder appends
// a trailing newline which we strip
func toRawJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := stdjson.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("toRawJson: %w", err)
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// marshals a slice/array as JSON Lines (NDJSON): one compact JSON value per line, no trailing newline. Common format
// for streaming logs and batch record output. Errors if the input isn't an array or slice - "one value per line" has no
// meaning for scalars or maps
func toJSONL(v any) (string, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return "", fmt.Errorf("toJsonL: expected array or slice, got %T", v)
	}
	var b strings.Builder
	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		data, err := stdjson.Marshal(rv.Index(i).Interface())
		if err != nil {
			return "", fmt.Errorf("toJsonL[%d]: %w", i, err)
		}
		b.Write(data)
	}
	return b.String(), nil
}

// parses JSON text into any value (object, array, scalar). Unlike sprig's fromJson which only accepts objects and
// stashes errors in a map["Error"] key, this returns the polymorphic value and errors on failure - matching sprig's own
// mustFromJson shape
func fromJSON(s string) (any, error) {
	var v any
	if err := stdjson.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("fromJson: %w", err)
	}
	return v, nil
}
