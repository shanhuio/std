package lexing

import (
	"testing"
)

// sliceTokener emits a predefined sequence of tokens, then EOF forever.
type sliceTokener struct {
	toks []*Token
	i    int
	errs []*Error
}

func (s *sliceTokener) Token() *Token {
	if s.i >= len(s.toks) {
		return &Token{Type: EOF}
	}
	ret := s.toks[s.i]
	s.i++
	return ret
}

func (s *sliceTokener) Errs() []*Error { return s.errs }

const (
	tokIdent   = 1
	tokKeyword = 2
)

func tok(typ int, lit string) *Token { return &Token{Type: typ, Lit: lit} }

func TestRemover(t *testing.T) {
	src := &sliceTokener{toks: []*Token{
		tok(Comment, "// a"),
		tok(tokIdent, "foo"),
		tok(Comment, "// b"),
		tok(Comment, "// c"),
		tok(tokIdent, "bar"),
	}}

	r := NewRemover(src, Comment)
	got := TokenAll(r)

	var lits []string
	for _, tk := range got {
		if tk.Type == EOF {
			continue
		}
		if tk.Type == Comment {
			t.Errorf("comment token leaked through remover: %q", tk.Lit)
		}
		lits = append(lits, tk.Lit)
	}
	if len(lits) != 2 || lits[0] != "foo" || lits[1] != "bar" {
		t.Errorf("filtered lits: got %v, want [foo bar]", lits)
	}
}

func TestRemoverNonComment(t *testing.T) {
	src := &sliceTokener{toks: []*Token{
		tok(tokIdent, "x"),
		tok(tokKeyword, "if"),
		tok(tokIdent, "y"),
	}}

	r := NewRemover(src, tokKeyword)
	got := TokenAll(r)

	var lits []string
	for _, tk := range got {
		if tk.Type != EOF {
			lits = append(lits, tk.Lit)
		}
	}
	if len(lits) != 2 || lits[0] != "x" || lits[1] != "y" {
		t.Errorf("filtered lits: got %v, want [x y]", lits)
	}
}

func TestNewCommentRemover(t *testing.T) {
	src := &sliceTokener{toks: []*Token{
		tok(Comment, "// drop"),
		tok(tokIdent, "keep"),
	}}

	r := NewCommentRemover(src)
	got := TokenAll(r)

	if len(got) != 2 { // "keep" + EOF
		t.Fatalf("got %d tokens, want 2", len(got))
	}
	if got[0].Lit != "keep" {
		t.Errorf("first token: got %q, want keep", got[0].Lit)
	}
}

func TestRemoverErrsRelayed(t *testing.T) {
	wantErr := &Error{Code: "x", Err: nil}
	src := &sliceTokener{errs: []*Error{wantErr}}
	r := NewRemover(src, Comment)

	errs := r.Errs()
	if len(errs) != 1 || errs[0] != wantErr {
		t.Errorf("Errs not relayed from wrapped tokener: got %v", errs)
	}
}

func TestNewRemoverPanicsOnEOF(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewRemover(_, EOF): got no panic, want panic")
		}
	}()
	NewRemover(&sliceTokener{}, EOF)
}
