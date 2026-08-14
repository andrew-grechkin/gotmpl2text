// Package predicate provides type-check helpers (isString, isNumber, isBool, isSlice, isMap, isNil) that read cleaner
// in template conditionals than sprig's stringly-typed `kindIs "map" .` pattern
package predicate

import (
	"reflect"
	"text/template"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"isString": isString,
		"isNumber": isNumber,
		"isBool":   isBool,
		"isSlice":  isSlice,
		"isMap":    isMap,
		"isNil":    isNil,
	}
}

func isString(v any) bool {
	_, ok := v.(string)
	return ok
}

// reports whether v is any of Go's built-in numeric kinds (signed/unsigned integer or float). Excludes bool
func isNumber(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	}
	return false
}

func isBool(v any) bool {
	_, ok := v.(bool)
	return ok
}

// reports whether v is a slice or array (both are indexable sequences as far as templates are concerned)
func isSlice(v any) bool {
	if v == nil {
		return false
	}
	k := reflect.ValueOf(v).Kind()
	return k == reflect.Slice || k == reflect.Array
}

func isMap(v any) bool {
	if v == nil {
		return false
	}
	return reflect.ValueOf(v).Kind() == reflect.Map
}

// isNil reports whether v is untyped nil or a typed nil (nil pointer, nil slice, nil map, nil chan, nil interface, nil
// func)
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	}
	return false
}
