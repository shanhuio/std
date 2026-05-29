package lexing

import (
	"testing"
)

func TestTypesDefaults(t *testing.T) {
	types := NewTypes()

	for _, test := range []struct {
		t    int
		want string
	}{
		{EOF, "eof"},
		{Comment, "comment"},
		{Illegal, "illegal"},
	} {
		if got := types.Name(test.t); got != test.want {
			t.Errorf("Name(%d): got %q, want %q", test.t, got, test.want)
		}
	}
}

func TestTypesRegisterAndName(t *testing.T) {
	types := NewTypes()
	types.Register(1, "ident")
	types.Register(2, "number")

	for _, test := range []struct {
		t    int
		want string
	}{
		{1, "ident"},
		{2, "number"},
	} {
		if got := types.Name(test.t); got != test.want {
			t.Errorf("Name(%d): got %q, want %q", test.t, got, test.want)
		}
	}
}

func TestTypesNameUnregistered(t *testing.T) {
	types := NewTypes()
	if got, want := types.Name(42), "<T42>"; got != want {
		t.Errorf("Name(42): got %q, want %q", got, want)
	}
}

func TestTypesRegisterDuplicatePanics(t *testing.T) {
	types := NewTypes()
	types.Register(7, "first")

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Register of duplicate type: got no panic, want panic")
		}
	}()
	types.Register(7, "second")
}

func TestTypesRegisterDefaultPanics(t *testing.T) {
	types := NewTypes()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Register over default token: got no panic, want panic")
		}
	}()
	types.Register(EOF, "again")
}
