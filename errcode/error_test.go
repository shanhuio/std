package errcode

import (
	"errors"
	"strings"
	"testing"
)

func TestCommonError(t *testing.T) {
	for _, test := range []struct {
		f   func(err error) bool
		err error
	}{
		{IsNotFound, NotFoundf("not found")},
		{IsInvalidArg, InvalidArgf("invalid arg")},
		{IsUnauthorized, Unauthorizedf("unauthorized")},
		{IsInternal, Internalf("internal")},
		{IsTimeOut, TimeOutf("time out")},
	} {
		if !test.f(test.err) {
			t.Errorf("test failed for error: %s", test.err)
		}
	}
}

func TestAnnotate(t *testing.T) {
	t.Run("preserves error code", func(t *testing.T) {
		annotated := Annotate(NotFoundf("missing"), "while doing X")
		if !IsNotFound(annotated) {
			t.Errorf("IsNotFound after Annotate: got false, want true")
		}
		msg := annotated.Error()
		if !strings.Contains(msg, "while doing X") {
			t.Errorf("missing prefix: got %q", msg)
		}
		if !strings.Contains(msg, "missing") {
			t.Errorf("missing original message: got %q", msg)
		}
	})

	t.Run("plain error stays codeless", func(t *testing.T) {
		annotated := Annotate(errors.New("oops"), "context")
		if Of(annotated) != "" {
			t.Errorf("Of: got %q, want empty", Of(annotated))
		}
		if got := annotated.Error(); got != "context: oops" {
			t.Errorf("error: got %q, want %q", got, "context: oops")
		}
	})

	t.Run("supports errors.Is", func(t *testing.T) {
		sentinel := errors.New("sentinel")
		if !errors.Is(Annotate(sentinel, "ctx"), sentinel) {
			t.Errorf("errors.Is should find sentinel through Annotate")
		}
	})
}

func TestAnnotatef(t *testing.T) {
	t.Run("format args", func(t *testing.T) {
		annotated := Annotatef(InvalidArgf("bad value"), "parse %q", "input")
		if !IsInvalidArg(annotated) {
			t.Errorf("IsInvalidArg: got false, want true")
		}
		if msg := annotated.Error(); !strings.Contains(msg, `parse "input"`) {
			t.Errorf("got %q", msg)
		}
	})

	t.Run("nested annotation preserves code and order", func(t *testing.T) {
		once := Annotatef(NotFoundf("file"), "in %s", "dir1")
		twice := Annotatef(once, "in %s", "dir2")

		if !IsNotFound(twice) {
			t.Errorf("nested annotation should preserve NotFound code")
		}

		msg := twice.Error()
		idx := func(s string) int { return strings.Index(msg, s) }
		i1, i2, i3 := idx("dir2"), idx("dir1"), idx("file")
		if i1 < 0 || i2 < 0 || i3 < 0 {
			t.Fatalf("missing one of dir2/dir1/file in %q", msg)
		}
		if !(i1 < i2 && i2 < i3) {
			t.Errorf("expected outer-to-inner order (dir2 < dir1 < file) in %q", msg)
		}
	})
}
