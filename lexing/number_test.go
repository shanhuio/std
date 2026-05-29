package lexing

import (
	"strings"
	"testing"
)

const (
	numInt   = 1
	numFloat = 2
)

// lexNumbers lexes a whitespace-separated stream of numbers and returns them.
func lexNumbers(t *testing.T, input string) []*Token {
	t.Helper()
	tokener := NewTokener("test.txt", strings.NewReader(input), func(x *Lexer) *Token {
		return LexNumber(x, numInt, numFloat)
	}, IsWhite)

	toks, errs := Tokens(tokener)
	if len(errs) != 0 {
		t.Fatalf("unexpected lex errors: %v", errs)
	}

	var ret []*Token
	for _, tok := range toks {
		if tok.Type == EOF {
			continue
		}
		ret = append(ret, tok)
	}
	return ret
}

func TestLexNumber(t *testing.T) {
	for _, test := range []struct {
		input   string
		wantLit string
		wantTyp int
	}{
		{"0", "0", numInt},
		{"42", "42", numInt},
		{"007", "007", numInt},
		{"0xff", "0xff", numInt},
		{"0x1A2b", "0x1A2b", numInt},
		{"0x", "0x", numInt},
		{"3.14", "3.14", numFloat},
		{"10.", "10.", numFloat},
		{"1e10", "1e10", numFloat},
		{"2E5", "2E5", numFloat},
		{"1.5e-3", "1.5e-3", numFloat},
		{"6e-2", "6e-2", numFloat},
	} {
		toks := lexNumbers(t, test.input)
		if len(toks) != 1 {
			t.Errorf("LexNumber(%q): got %d tokens, want 1", test.input, len(toks))
			continue
		}
		got := toks[0]
		if got.Lit != test.wantLit {
			t.Errorf("LexNumber(%q): lit got %q, want %q", test.input, got.Lit, test.wantLit)
		}
		if got.Type != test.wantTyp {
			t.Errorf("LexNumber(%q): type got %d, want %d", test.input, got.Type, test.wantTyp)
		}
	}
}

func TestLexNumberSequence(t *testing.T) {
	toks := lexNumbers(t, "1 2.0 0xa")
	want := []struct {
		lit string
		typ int
	}{
		{"1", numInt},
		{"2.0", numFloat},
		{"0xa", numInt},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(toks), len(want))
	}
	for i, w := range want {
		if toks[i].Lit != w.lit || toks[i].Type != w.typ {
			t.Errorf("token %d: got (%q,%d), want (%q,%d)",
				i, toks[i].Lit, toks[i].Type, w.lit, w.typ)
		}
	}
}

func TestLexNumberPanicsOnNonDigit(t *testing.T) {
	x := NewLexer("test.txt", strings.NewReader("abc"))

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("LexNumber on non-digit start: got no panic, want panic")
		}
	}()
	LexNumber(x, numInt, numFloat)
}
