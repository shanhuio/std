package lexing

import (
	"strings"
	"testing"
)

func TestIsIdentLetter(t *testing.T) {
	for _, r := range "_abzABZ" {
		if !IsIdentLetter(r) {
			t.Errorf("%q should be an ident letter", r)
		}
	}
	for _, r := range "013-%~ \t.@" {
		if IsIdentLetter(r) {
			t.Errorf("%q should not be an ident letter", r)
		}
	}
}

const identType = 1

// lexIdents lexes a whitespace-separated stream of identifiers and returns
// their literals.
func lexIdents(t *testing.T, input string) []string {
	t.Helper()
	x := MakeLexer("test.txt", strings.NewReader(input), func(x *Lexer) *Token {
		return LexIdent(x, identType)
	})

	var lits []string
	for {
		tok := x.Token()
		if tok.Type == EOF {
			break
		}
		lits = append(lits, tok.Lit)
	}
	if errs := x.Errs(); len(errs) != 0 {
		t.Fatalf("unexpected lex errors: %v", errs)
	}
	return lits
}

func TestLexIdent(t *testing.T) {
	for _, test := range []struct {
		input string
		want  []string
	}{
		{"foo", []string{"foo"}},
		{"foo bar baz", []string{"foo", "bar", "baz"}},
		{"_x9", []string{"_x9"}},
		{"a1b2 _under_score", []string{"a1b2", "_under_score"}},
		{"  spaced\tout  ", []string{"spaced", "out"}},
	} {
		got := lexIdents(t, test.input)
		if len(got) != len(test.want) {
			t.Errorf("LexIdent(%q): got %v, want %v", test.input, got, test.want)
			continue
		}
		for i := range got {
			if got[i] != test.want[i] {
				t.Errorf("LexIdent(%q): got %v, want %v", test.input, got, test.want)
				break
			}
		}
	}
}

func TestLexIdentPanicsOnNonLetter(t *testing.T) {
	x := NewLexer("test.txt", strings.NewReader("123"))

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("LexIdent on digit start: got no panic, want panic")
		}
	}()
	LexIdent(x, identType)
}
