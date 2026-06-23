package lexing

import (
	"bytes"
	"errors"
	"testing"
)

func TestErrorListAddAndErrs(t *testing.T) {
	lst := NewErrorList()
	if lst.Errs() != nil {
		t.Errorf("fresh list Errs: got %v, want nil", lst.Errs())
	}

	lst.Errorf(nil, "boom %d", 1)
	lst.CodeErrorf(&Pos{Line: 2}, "code", "bad")

	errs := lst.Errs()
	if len(errs) != 2 {
		t.Fatalf("got %d errors, want 2", len(errs))
	}
	if errs[0].Err.Error() != "boom 1" || errs[0].Code != "" {
		t.Errorf("first error: %+v", errs[0])
	}
	if errs[1].Code != "code" {
		t.Errorf("second error code: got %q, want code", errs[1].Code)
	}
}

func TestErrorListAddNilPanics(t *testing.T) {
	lst := NewErrorList()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Add(nil): got no panic, want panic")
		}
	}()
	lst.Add(nil)
}

func TestErrorListAddAll(t *testing.T) {
	lst := NewErrorList()
	lst.AddAll([]*Error{
		{Err: errors.New("a")},
		{Err: errors.New("b")},
	})
	if len(lst.Errs()) != 2 {
		t.Errorf("got %d errors, want 2", len(lst.Errs()))
	}
	if !lst.InJail() {
		t.Errorf("InJail after AddAll: got false, want true")
	}
}

func TestErrorListMax(t *testing.T) {
	lst := NewErrorList()
	lst.Max = 2
	for i := range 5 {
		lst.Errorf(nil, "err %d", i)
	}
	if got := len(lst.Errs()); got != 2 {
		t.Errorf("capped error count: got %d, want 2", got)
	}
	// Even when capped, the list remains in jail.
	if !lst.InJail() {
		t.Errorf("InJail after capped adds: got false, want true")
	}
}

func TestErrorListJailBailOut(t *testing.T) {
	lst := NewErrorList()
	if lst.InJail() {
		t.Errorf("fresh list InJail: got true, want false")
	}
	lst.Jail()
	if !lst.InJail() {
		t.Errorf("after Jail: got false, want true")
	}
	lst.BailOut()
	if lst.InJail() {
		t.Errorf("after BailOut: got true, want false")
	}
}

func TestErrorListPrint(t *testing.T) {
	lst := NewErrorList()
	lst.Errorf(&Pos{File: "a.go", Line: 1, Col: 1}, "first")
	lst.Errorf(nil, "second")

	var buf bytes.Buffer
	if err := lst.Print(&buf); err != nil {
		t.Fatalf("Print: %v", err)
	}
	want := "a.go:1: first\nsecond\n"
	if got := buf.String(); got != want {
		t.Errorf("Print: got %q, want %q", got, want)
	}
}

// errWriter fails on the first write, to exercise Print's error path.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestErrorListPrintError(t *testing.T) {
	lst := NewErrorList()
	lst.Errorf(nil, "boom")
	if err := lst.Print(errWriter{}); err == nil {
		t.Errorf("Print to failing writer: got nil error, want error")
	}
}

// TestLogErrorWithErrorList exercises LogError against a real ErrorList, which
// is the concrete Logger used throughout the package.
func TestLogErrorWithErrorList(t *testing.T) {
	lst := NewErrorList()
	if LogError(lst, nil) {
		t.Errorf("LogError(nil): got true, want false")
	}
	if len(lst.Errs()) != 0 {
		t.Errorf("LogError(nil) should not log, got %d errors", len(lst.Errs()))
	}

	if !LogError(lst, errors.New("boom")) {
		t.Errorf("LogError(err): got false, want true")
	}
	errs := lst.Errs()
	if len(errs) != 1 || errs[0].Err.Error() != "boom" {
		t.Errorf("LogError did not log correctly: %+v", errs)
	}
}

func TestSingleErr(t *testing.T) {
	errs := SingleErr(errors.New("boom"))
	if len(errs) != 1 || errs[0].Err.Error() != "boom" || errs[0].Code != "" {
		t.Errorf("SingleErr: got %+v", errs)
	}
}

func TestSingleCodeErr(t *testing.T) {
	errs := SingleCodeErr("code", errors.New("boom"))
	if len(errs) != 1 || errs[0].Code != "code" || errs[0].Err.Error() != "boom" {
		t.Errorf("SingleCodeErr: got %+v", errs)
	}
}
