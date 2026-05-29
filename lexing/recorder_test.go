package lexing

import (
	"strings"
	"testing"
)

func newIdentTokener(input string) Tokener {
	return NewTokener("test.txt", strings.NewReader(input), func(x *Lexer) *Token {
		return LexIdent(x, identType)
	}, IsWhite)
}

func TestRecorderToken(t *testing.T) {
	rec := NewRecorder(newIdentTokener("foo bar"))

	// Pull tokens one at a time; the recorder should relay and record each.
	t1 := rec.Token()
	t2 := rec.Token()
	if t1.Lit != "foo" || t2.Lit != "bar" {
		t.Fatalf("relayed tokens: got %q, %q; want foo, bar", t1.Lit, t2.Lit)
	}

	recorded := rec.Tokens()
	if len(recorded) != 2 {
		t.Fatalf("recorded %d tokens, want 2", len(recorded))
	}
	if recorded[0] != t1 || recorded[1] != t2 {
		t.Errorf("recorded tokens do not match relayed tokens")
	}
}

func TestRecorderEmpty(t *testing.T) {
	rec := NewRecorder(newIdentTokener(""))
	if got := rec.Tokens(); got != nil {
		t.Errorf("Tokens before reading: got %v, want nil", got)
	}
}

func TestTokenAll(t *testing.T) {
	toks := TokenAll(newIdentTokener("a b c"))

	// TokenAll records every fetched token, including the trailing EOF.
	if len(toks) != 4 {
		t.Fatalf("got %d tokens, want 4 (3 idents + EOF)", len(toks))
	}
	for i, want := range []string{"a", "b", "c"} {
		if toks[i].Lit != want {
			t.Errorf("token %d: got %q, want %q", i, toks[i].Lit, want)
		}
	}
	if last := toks[len(toks)-1]; last.Type != EOF {
		t.Errorf("last token type: got %d, want EOF", last.Type)
	}
}

func TestTokenAllEmpty(t *testing.T) {
	toks := TokenAll(newIdentTokener(""))
	if len(toks) != 1 || toks[0].Type != EOF {
		t.Errorf("TokenAll on empty input: got %v, want a single EOF token", toks)
	}
}
