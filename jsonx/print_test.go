package jsonx

import (
	"testing"

	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

func TestMarshal_loopback(t *testing.T) {
	for _, obj := range []any{
		"something",
		1.234,
		1234,
		nil,
		struct{ A, B string }{A: "a", B: "b"},
		map[string]string{
			"a.com": "a:8888",
			"b.com": "b:7777",
		},
		[]int{1, 2, 3},
	} {
		want, err := json.Marshal(obj)
		if err != nil {
			t.Fatalf("marshal %v: %v", obj, err)
		}

		x, err := Marshal(obj)
		if err != nil {
			t.Errorf("format %v: %v", obj, err)
			continue
		}

		var box any
		if err := Unmarshal(x, &box); err != nil {
			t.Errorf("unmarshal %q: %v", x, err)
			continue
		}

		got, err := json.Marshal(box)
		if err != nil {
			t.Fatalf("marshal jsonx-gen %v: %v", obj, err)
		}

		if !bytes.Equal(want, got) {
			t.Errorf(
				"format test failed %v: got %q, want %q",
				obj, got, want,
			)
		}
	}
}

func TestSprint(t *testing.T) {
	for _, test := range []struct {
		in   any
		want string
	}{
		{1234, "1234\n"},
		{"hi", "\"hi\"\n"},
		{true, "true\n"},
		{nil, "null\n"},
		{[]int{1, 2}, "[\n  1,\n  2,\n]\n"},
		{
			map[string]any{"a": 1, "b": "x"},
			"{\n  a: 1,\n  b: \"x\",\n}\n",
		},
	} {
		got, err := Sprint(test.in)
		if err != nil {
			t.Errorf("Sprint(%v): %v", test.in, err)
			continue
		}
		if got != test.want {
			t.Errorf("Sprint(%v): got %q, want %q", test.in, got, test.want)
		}
	}
}

func TestSprintError(t *testing.T) {
	// A value JSON cannot marshal yields an error and an empty string.
	got, err := Sprint(make(chan int))
	if err == nil {
		t.Errorf("Sprint(chan): got nil error, want one")
	}
	if got != "" {
		t.Errorf("Sprint(chan): got %q, want empty string", got)
	}
}

func TestSprintPreservesFieldOrder(t *testing.T) {
	// Struct fields print in declaration order, not sorted alphabetically.
	v := struct {
		Zebra int
		Apple int
		Mango int
	}{1, 2, 3}

	got, err := Sprint(v)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  Zebra: 1,\n  Apple: 2,\n  Mango: 3,\n}\n"
	if got != want {
		t.Errorf("Sprint: got %q, want %q", got, want)
	}
}

func TestSprintNestedOrder(t *testing.T) {
	// Order is preserved at every level of nesting.
	type inner struct {
		Y int
		X int
	}
	v := struct {
		Second inner
		Items  []int
		First  int
	}{Second: inner{Y: 1, X: 2}, Items: []int{7, 8}, First: 3}

	got, err := Sprint(v)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  Second: {\n    Y: 1,\n    X: 2,\n  },\n" +
		"  Items: [\n    7,\n    8,\n  ],\n  First: 3,\n}\n"
	if got != want {
		t.Errorf("Sprint: got %q, want %q", got, want)
	}
}

func TestSprintMapStillSorted(t *testing.T) {
	// Maps have no field order; json.Marshal sorts their keys, and that order
	// is preserved, so map output stays deterministic (sorted).
	v := map[string]any{"b": 1, "a": 2, "c": 3}

	got, err := Sprint(v)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  a: 2,\n  b: 1,\n  c: 3,\n}\n"
	if got != want {
		t.Errorf("Sprint: got %q, want %q", got, want)
	}
}

func TestSprintLargeInt(t *testing.T) {
	// Integers beyond 2^53 are preserved exactly, since numbers no longer pass
	// through float64.
	got, err := Sprint(int64(9007199254740993))
	if err != nil {
		t.Fatal(err)
	}
	if want := "9007199254740993\n"; got != want {
		t.Errorf("Sprint: got %q, want %q", got, want)
	}
}

func ExamplePrint() {
	Print(map[string]any{"name": "svc", "port": 8080})
	// Output:
	// {
	//   name: "svc",
	//   port: 8080,
	// }
}

func TestWriteFile(t *testing.T) {
	type config struct {
		Name string
		Port int
	}
	want := config{Name: "svc", Port: 8080}
	path := filepath.Join(t.TempDir(), "out.jsonx")

	if err := WriteFile(path, want); err != nil {
		t.Fatal(err)
	}

	// The file round-trips back to the same value.
	var got config
	if err := ReadFile(path, &got); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got != want {
		t.Errorf("round trip: got %+v, want %+v", got, want)
	}
}

func TestWriteFileMarshalError(t *testing.T) {
	// A value JSON cannot marshal makes WriteFile fail before writing, so no
	// file is created.
	path := filepath.Join(t.TempDir(), "out.jsonx")
	if err := WriteFile(path, make(chan int)); err == nil {
		t.Errorf("WriteFile with unmarshalable value: got nil error, want one")
	}
	if _, err := os.Stat(path); err == nil {
		t.Errorf("WriteFile should not create a file on marshal error")
	}
}

func TestWriteFileBadPath(t *testing.T) {
	// The parent path component is a regular file, not a directory, so the
	// write fails.
	notDir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notDir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(notDir, "out.jsonx")
	if err := WriteFile(bad, 1234); err == nil {
		t.Errorf("WriteFile to bad path: got nil error, want one")
	}
}
