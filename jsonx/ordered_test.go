package jsonx

import (
	"encoding/json"
	"testing"
)

func TestDecodeOrdered(t *testing.T) {
	// Object key order from the input is preserved, and the various value
	// kinds decode to the expected Go types.
	v, err := decodeOrdered([]byte(`{"z":1,"a":[true,null,"s"],"m":{"y":2,"x":3}}`))
	if err != nil {
		t.Fatal(err)
	}

	obj, ok := v.(orderedObject)
	if !ok {
		t.Fatalf("top level: got %T, want orderedObject", v)
	}

	var keys []string
	for _, m := range obj {
		keys = append(keys, m.key)
	}
	if want := []string{"z", "a", "m"}; !equalStrings(keys, want) {
		t.Errorf("key order: got %v, want %v", keys, want)
	}

	// "z" is a json.Number.
	if n, ok := obj[0].value.(json.Number); !ok || n.String() != "1" {
		t.Errorf("z: got %#v, want json.Number 1", obj[0].value)
	}

	// "a" is an array carrying a bool, a nil, and a string.
	arr, ok := obj[1].value.([]any)
	if !ok || len(arr) != 3 {
		t.Fatalf("a: got %#v, want a 3-element []any", obj[1].value)
	}
	if arr[0] != true || arr[1] != nil {
		t.Errorf("a elements: got %#v", arr)
	}
	if s, ok := arr[2].(string); !ok || s != "s" {
		t.Errorf("a[2]: got %#v, want string s", arr[2])
	}

	// "m" is a nested ordered object, also order-preserving.
	nested, ok := obj[2].value.(orderedObject)
	if !ok || len(nested) != 2 || nested[0].key != "y" || nested[1].key != "x" {
		t.Errorf("m: got %#v, want ordered {y, x}", obj[2].value)
	}
}

func TestDecodeOrderedEmpty(t *testing.T) {
	obj, err := decodeOrdered([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if o, ok := obj.(orderedObject); !ok || len(o) != 0 {
		t.Errorf("empty object: got %#v", obj)
	}

	arr, err := decodeOrdered([]byte(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := arr.([]any); !ok || len(a) != 0 {
		t.Errorf("empty array: got %#v", arr)
	}
}

func TestDecodeOrderedErrors(t *testing.T) {
	for _, in := range []string{
		``,         // no token at all
		`{`,        // truncated object
		`[`,        // truncated array
		`{"a":}`,   // missing value
		`{"a":1`,   // unterminated object
		`[1,2`,     // unterminated array
		`{"a":[1}`, // mismatched nesting
	} {
		if _, err := decodeOrdered([]byte(in)); err == nil {
			t.Errorf("decodeOrdered(%q): got nil error, want one", in)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
