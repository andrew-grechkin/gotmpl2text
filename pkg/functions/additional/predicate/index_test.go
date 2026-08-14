package predicate

import "testing"

func TestIsString(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{"", true},
		{"hello", true},
		{42, false},
		{3.14, false},
		{true, false},
		{nil, false},
		{[]string{"a"}, false},
	}
	for _, c := range cases {
		if got := isString(c.v); got != c.want {
			t.Errorf("isString(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestIsNumber(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{0, true},
		{int64(42), true},
		{uint(1), true},
		{3.14, true},
		{float32(1.5), true},
		{"42", false},
		{true, false},
		{false, false}, // bool must NOT count as number even though it's numeric-ish
		{nil, false},
		{[]int{1}, false},
	}
	for _, c := range cases {
		if got := isNumber(c.v); got != c.want {
			t.Errorf("isNumber(%v %T) = %v, want %v", c.v, c.v, got, c.want)
		}
	}
}

func TestIsBool(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{true, true},
		{false, true},
		{1, false},
		{"true", false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isBool(c.v); got != c.want {
			t.Errorf("isBool(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestIsSlice(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{[]int{1, 2, 3}, true},
		{[]string{}, true},
		{[3]int{1, 2, 3}, true}, // arrays count
		{"abc", false},
		{map[string]int{"a": 1}, false},
		{nil, false},
		{[]any(nil), true}, // typed nil slice is still a slice
	}
	for _, c := range cases {
		if got := isSlice(c.v); got != c.want {
			t.Errorf("isSlice(%v %T) = %v, want %v", c.v, c.v, got, c.want)
		}
	}
}

func TestIsMap(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{map[string]int{"a": 1}, true},
		{map[any]any{}, true},
		{[]int{1}, false},
		{"abc", false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isMap(c.v); got != c.want {
			t.Errorf("isMap(%v %T) = %v, want %v", c.v, c.v, got, c.want)
		}
	}
}

func TestIsNil(t *testing.T) {
	var nilSlice []int
	var nilMap map[string]int
	var nilPtr *int
	var nilChan chan int
	var nilFunc func()

	cases := []struct {
		name string
		v    any
		want bool
	}{
		{"untyped nil", nil, true},
		{"nil slice", nilSlice, true},
		{"nil map", nilMap, true},
		{"nil pointer", nilPtr, true},
		{"nil chan", nilChan, true},
		{"nil func", nilFunc, true},
		{"empty string", "", false},
		{"zero int", 0, false},
		{"false", false, false},
		{"empty slice", []int{}, false},
		{"empty map", map[string]int{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isNil(c.v); got != c.want {
				t.Errorf("isNil(%v) = %v, want %v", c.v, got, c.want)
			}
		})
	}
}
