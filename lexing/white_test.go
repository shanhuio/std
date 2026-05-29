package lexing

import (
	"testing"
)

func TestIsWhite(t *testing.T) {
	for _, test := range []struct {
		r    rune
		want bool
	}{
		{' ', true},
		{'\t', true},
		{'\r', true},
		{'\n', false},
		{'a', false},
		{'0', false},
		{'\v', false},
		{'\f', false},
		{0, false},
	} {
		if got := IsWhite(test.r); got != test.want {
			t.Errorf("IsWhite(%q): got %v, want %v", test.r, got, test.want)
		}
	}
}

func TestIsWhiteOrEndl(t *testing.T) {
	for _, test := range []struct {
		r    rune
		want bool
	}{
		{' ', true},
		{'\t', true},
		{'\r', true},
		{'\n', true},
		{'a', false},
		{'0', false},
		{'\v', false},
		{0, false},
	} {
		if got := IsWhiteOrEndl(test.r); got != test.want {
			t.Errorf("IsWhiteOrEndl(%q): got %v, want %v", test.r, got, test.want)
		}
	}
}
