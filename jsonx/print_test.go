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
