package errcode

import (
	"errors"
	"os"
	"testing"
)

// timeoutErr satisfies the implicit `interface{ Timeout() bool }` that
// os.IsTimeout looks for.
type timeoutErr struct{ msg string }

func (t *timeoutErr) Error() string { return t.msg }
func (t *timeoutErr) Timeout() bool { return true }

func TestFromOS(t *testing.T) {
	t.Run("nil passes through", func(t *testing.T) {
		if got := FromOS(nil); got != nil {
			t.Errorf("FromOS(nil): got %v, want nil", got)
		}
	})

	t.Run("not-exist", func(t *testing.T) {
		got := FromOS(os.ErrNotExist)
		if !IsNotFound(got) {
			t.Errorf("IsNotFound: got false, want true (err=%v)", got)
		}
	})

	t.Run("PathError wrapping not-exist", func(t *testing.T) {
		base := &os.PathError{Op: "open", Path: "/missing", Err: os.ErrNotExist}
		got := FromOS(base)
		if !IsNotFound(got) {
			t.Errorf("IsNotFound: got false, want true (err=%v)", got)
		}
	})

	t.Run("permission", func(t *testing.T) {
		got := FromOS(os.ErrPermission)
		if !IsUnauthorized(got) {
			t.Errorf("IsUnauthorized: got false, want true (err=%v)", got)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		got := FromOS(&timeoutErr{msg: "deadline"})
		if !IsTimeOut(got) {
			t.Errorf("IsTimeOut: got false, want true (err=%v)", got)
		}
	})

	t.Run("plain error passes through", func(t *testing.T) {
		base := errors.New("plain")
		got := FromOS(base)
		if got != base {
			t.Errorf("FromOS(plain): got %v, want same instance", got)
		}
		if Of(got) != "" {
			t.Errorf("Of(plain): got %q, want empty", Of(got))
		}
	})
}
