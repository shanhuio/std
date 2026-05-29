package lexing

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestErrorError(t *testing.T) {
	t.Run("with position", func(t *testing.T) {
		e := &Error{
			Pos: &Pos{File: "a.go", Line: 4, Col: 2},
			Err: errors.New("boom"),
		}
		if got, want := e.Error(), "a.go:4: boom"; got != want {
			t.Errorf("Error(): got %q, want %q", got, want)
		}
	})

	t.Run("without position", func(t *testing.T) {
		e := &Error{Err: errors.New("boom")}
		if got, want := e.Error(), "boom"; got != want {
			t.Errorf("Error(): got %q, want %q", got, want)
		}
	})
}

func TestErrorRelFile(t *testing.T) {
	t.Run("nil position falls back to Error", func(t *testing.T) {
		e := &Error{Err: errors.New("boom")}
		if got := e.ErrorRelFile("/work"); got != "boom" {
			t.Errorf("got %q, want boom", got)
		}
	})

	t.Run("empty workDir falls back to Error", func(t *testing.T) {
		e := &Error{Pos: &Pos{File: "/work/a.go", Line: 1, Col: 1}, Err: errors.New("boom")}
		if got := e.ErrorRelFile(""); got != "/work/a.go:1: boom" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("relative to workDir", func(t *testing.T) {
		e := &Error{Pos: &Pos{File: "/work/pkg/a.go", Line: 7, Col: 1}, Err: errors.New("boom")}
		if got, want := e.ErrorRelFile("/work"), "pkg/a.go:7: boom"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to absolute when Rel fails", func(t *testing.T) {
		// A relative workDir against an absolute file path cannot be made
		// relative, so the original file path is used.
		e := &Error{Pos: &Pos{File: "/abs/a.go", Line: 2, Col: 1}, Err: errors.New("boom")}
		if got, want := e.ErrorRelFile("rel"), "/abs/a.go:2: boom"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestErrorJSON(t *testing.T) {
	t.Run("with position", func(t *testing.T) {
		e := &Error{
			Pos:  &Pos{File: "a.go", Line: 3, Col: 9},
			Err:  errors.New("boom"),
			Code: "lexing.x",
		}
		bs, err := json.Marshal(e.JSON())
		if err != nil {
			t.Fatal(err)
		}
		got := string(bs)
		for _, want := range []string{
			`"file":"a.go"`, `"line":3`, `"col":9`,
			`"code":"lexing.x"`, `"err":"boom"`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("JSON %s missing %s", got, want)
			}
		}
	})

	t.Run("without position", func(t *testing.T) {
		e := &Error{Err: errors.New("boom")}
		bs, err := json.Marshal(e.JSON())
		if err != nil {
			t.Fatal(err)
		}
		got := string(bs)
		if !strings.Contains(got, `"file":""`) || !strings.Contains(got, `"line":0`) {
			t.Errorf("JSON without pos: got %s", got)
		}
	})
}

func TestCodeErrorfConstructor(t *testing.T) {
	e := CodeErrorf("lexing.code", "bad %d", 5)
	if e.Code != "lexing.code" {
		t.Errorf("Code: got %q, want lexing.code", e.Code)
	}
	if e.Err.Error() != "bad 5" {
		t.Errorf("Err: got %q, want bad 5", e.Err.Error())
	}
}

func TestErrorfConstructor(t *testing.T) {
	e := Errorf("bad %d", "", 7)
	if e.Code != "" {
		t.Errorf("Code: got %q, want empty", e.Code)
	}
	if e.Err.Error() != "bad 7" {
		t.Errorf("Err: got %q, want bad 7", e.Err.Error())
	}
}

func TestFprintErrs(t *testing.T) {
	errs := []*Error{
		{Pos: &Pos{File: "/work/a.go", Line: 1, Col: 1}, Err: errors.New("first")},
		{Err: errors.New("second")},
	}
	var buf bytes.Buffer
	FprintErrs(&buf, errs, "/work")

	want := "a.go:1: first\nsecond\n"
	if got := buf.String(); got != want {
		t.Errorf("FprintErrs: got %q, want %q", got, want)
	}
}
