package jsonx

import (
	"testing"

	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// writeTempFile writes content to a fresh temp file and returns its path.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.jsonx")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDecoder(t *testing.T) {
	input := strings.NewReader(`"a""b";"c"`)

	dec := NewDecoder(input)
	var got []string
	for dec.More() {
		var s string
		if err := dec.Decode(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}

	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUnmarshal_error(t *testing.T) {
	var s string
	if err := Unmarshal([]byte(`"missing`), &s); err == nil {
		t.Errorf("parse incomplete string passed")
	}
}

func TestUnmarshal(t *testing.T) {
	var v int
	if err := Unmarshal([]byte("1234"), &v); err != nil {
		t.Fatal(err)
	}
	if v != 1234 {
		t.Errorf("got %d, want 1234", v)
	}
}

func TestDecoder_series(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		`str "string"`,
		`num 3`,
		`struct {Field: "value"}`,
	}, "\n"))

	type structType struct {
		Field string
	}

	dec := NewDecoder(input)
	tm := func(t string) any {
		switch t {
		case "str":
			return new(string)
		case "num":
			return new(int)
		case "struct":
			return new(structType)
		}
		return nil
	}
	list, errs := dec.DecodeSeries(tm)
	for _, err := range errs {
		t.Error(err)
	}

	strVal := "string"
	numVal := 3
	want := []*Typed{
		{Type: "str", V: &strVal},
		{Type: "num", V: &numVal},
		{Type: "struct", V: &structType{Field: "value"}},
	}
	if len(list) != len(want) {
		t.Errorf("got %d entries, want %d", len(list), len(want))
	} else {
		for i, got := range list {
			w := want[i]
			if got.Type != w.Type {
				t.Errorf(
					"entry #%d, got type %q, want %q",
					i, got.Type, w.Type,
				)
			}
			if !reflect.DeepEqual(got.V, w.V) {
				t.Errorf(
					"entry value #%d, got %+v, want %+v",
					i, got.V, w.V,
				)
			}
		}
	}
}

func TestReadFile(t *testing.T) {
	type config struct {
		Name string
		Port int
	}
	path := writeTempFile(t, "{\nname: \"svc\",\nport: 8080,\n}")

	var cfg config
	if err := ReadFile(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "svc" || cfg.Port != 8080 {
		t.Errorf("ReadFile: got %+v, want {svc 8080}", cfg)
	}
}

func TestReadFileMissing(t *testing.T) {
	var v any
	missing := filepath.Join(t.TempDir(), "nope.jsonx")
	if err := ReadFile(missing, &v); err == nil {
		t.Errorf("ReadFile on missing file: got nil error, want one")
	}
}

func TestReadFileTrailingContent(t *testing.T) {
	// A file with more than one value is rejected: the first value decodes,
	// then unmarshalFile sees there is more.
	path := writeTempFile(t, "1 2")
	var v int
	if err := ReadFile(path, &v); err == nil {
		t.Errorf("ReadFile with trailing content: got nil error, want one")
	}
}

func TestReadFileMaybeJSON(t *testing.T) {
	type config struct{ A, B int }

	// Valid JSONx is read directly.
	t.Run("jsonx", func(t *testing.T) {
		path := writeTempFile(t, "{a:1,b:2,}")
		var got config
		if err := ReadFileMaybeJSON(path, &got); err != nil {
			t.Fatal(err)
		}
		if got != (config{1, 2}) {
			t.Errorf("got %+v, want {1 2}", got)
		}
	})

	// Plain multi-line JSON is not valid JSONx (no end-of-line trailing
	// commas), but is accepted through the JSON fallback.
	t.Run("json fallback", func(t *testing.T) {
		path := writeTempFile(t, "{\n\"a\": 1,\n\"b\": 2\n}")
		if _, errs := ToJSON([]byte("{\n\"a\": 1,\n\"b\": 2\n}")); errs == nil {
			t.Fatal("input unexpectedly valid as JSONx; test no longer covers fallback")
		}
		var got config
		if err := ReadFileMaybeJSON(path, &got); err != nil {
			t.Fatal(err)
		}
		if got != (config{1, 2}) {
			t.Errorf("got %+v, want {1 2}", got)
		}
	})

	// Input that is neither valid JSONx nor valid JSON fails.
	t.Run("invalid", func(t *testing.T) {
		path := writeTempFile(t, "@@@")
		var got config
		if err := ReadFileMaybeJSON(path, &got); err == nil {
			t.Errorf("ReadFileMaybeJSON on garbage: got nil error, want one")
		}
	})

	// A missing file errors before any parsing.
	t.Run("missing", func(t *testing.T) {
		var got config
		missing := filepath.Join(t.TempDir(), "nope.jsonx")
		if err := ReadFileMaybeJSON(missing, &got); err == nil {
			t.Errorf("ReadFileMaybeJSON on missing file: got nil error, want one")
		}
	})
}

func TestReadSeriesFile(t *testing.T) {
	path := writeTempFile(t, strings.Join([]string{
		`str "hello"`,
		`num 7`,
	}, "\n"))

	tm := func(s string) any {
		switch s {
		case "str":
			return new(string)
		case "num":
			return new(int)
		}
		return nil
	}

	list, errs := ReadSeriesFile(path, tm)
	if len(errs) != 0 {
		t.Fatalf("ReadSeriesFile: unexpected errors %v", errs)
	}

	strVal := "hello"
	numVal := 7
	want := []*Typed{
		{Type: "str", V: &strVal},
		{Type: "num", V: &numVal},
	}
	if len(list) != len(want) {
		t.Fatalf("got %d entries, want %d", len(list), len(want))
	}
	for i, got := range list {
		if got.Type != want[i].Type {
			t.Errorf("entry #%d type: got %q, want %q", i, got.Type, want[i].Type)
		}
		if !reflect.DeepEqual(got.V, want[i].V) {
			t.Errorf("entry #%d value: got %+v, want %+v", i, got.V, want[i].V)
		}
	}
}

func TestReadSeriesFileMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.jsonx")
	_, errs := ReadSeriesFile(missing, func(string) any { return nil })
	if len(errs) == 0 {
		t.Errorf("ReadSeriesFile on missing file: got no errors, want one")
	}
}

func TestDecoder_series_error(t *testing.T) {
	s := strings.Join([]string{
		`t {`,
		`	a:"x/**`,
		`}`,
	}, "\n")
	input := strings.NewReader(s)
	dec := NewDecoder(input)
	type structType struct{}
	tm := func(t string) any {
		return new(structType)
	}
	if _, errs := dec.DecodeSeries(tm); errs == nil {
		t.Errorf("decode %q got no error", s)
	} else {
		for _, err := range errs {
			t.Log(err)
		}
	}
}
