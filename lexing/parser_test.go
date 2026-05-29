package lexing

import (
	"testing"
)

const tokSemi = 3

func newTestParser(toks ...*Token) *Parser {
	return NewParser(&sliceTokener{toks: toks}, NewTypes())
}

func TestParserSeeAndToken(t *testing.T) {
	p := newTestParser(tok(tokIdent, "a"), tok(tokKeyword, "b"))

	if p.Token().Lit != "a" {
		t.Fatalf("first token: got %q, want a", p.Token().Lit)
	}
	if !p.See(tokIdent) {
		t.Errorf("See(tokIdent): got false, want true")
	}
	if p.See(tokKeyword) {
		t.Errorf("See(tokKeyword): got true, want false")
	}
	if !p.SeeLit(tokIdent, "a") {
		t.Errorf("SeeLit(tokIdent, a): got false, want true")
	}
	if p.SeeLit(tokIdent, "z") {
		t.Errorf("SeeLit(tokIdent, z): got true, want false")
	}
}

func TestParserAccept(t *testing.T) {
	p := newTestParser(tok(tokIdent, "a"), tok(tokKeyword, "b"))

	if p.Accept(tokKeyword) {
		t.Errorf("Accept(tokKeyword) on ident: got true, want false")
	}
	if p.Token().Lit != "a" {
		t.Errorf("after failed Accept, token moved: got %q", p.Token().Lit)
	}
	if !p.Accept(tokIdent) {
		t.Errorf("Accept(tokIdent): got false, want true")
	}
	if p.Token().Lit != "b" {
		t.Errorf("after Accept, token: got %q, want b", p.Token().Lit)
	}
}

func TestParserShiftAndNext(t *testing.T) {
	p := newTestParser(tok(tokIdent, "a"), tok(tokIdent, "b"))

	shifted := p.Shift()
	if shifted.Lit != "a" {
		t.Errorf("Shift returned %q, want a", shifted.Lit)
	}
	if p.Token().Lit != "b" {
		t.Errorf("after Shift, current %q, want b", p.Token().Lit)
	}
	if got := p.Next(); got.Type != EOF {
		t.Errorf("Next past end: got type %d, want EOF", got.Type)
	}
}

func TestParserExpect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "a"), tok(tokIdent, "b"))
		got := p.Expect(tokIdent)
		if got == nil || got.Lit != "a" {
			t.Fatalf("Expect returned %v, want token a", got)
		}
		if p.InError() {
			t.Errorf("InError after success: got true, want false")
		}
		if p.Token().Lit != "b" {
			t.Errorf("current after Expect: got %q, want b", p.Token().Lit)
		}
	})

	t.Run("failure enters error state", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "a"))
		if got := p.Expect(tokKeyword); got != nil {
			t.Errorf("Expect mismatch: got %v, want nil", got)
		}
		if !p.InError() {
			t.Errorf("InError after failure: got false, want true")
		}
		if errs := p.Errs(); len(errs) != 1 {
			t.Fatalf("got %d errors, want 1", len(errs))
		} else if errs[0].Code != "lexing.unexpected" {
			t.Errorf("error code: got %q, want lexing.unexpected", errs[0].Code)
		}
	})

	t.Run("no-op while in error", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "a"))
		p.Expect(tokKeyword) // enter error, 1 error
		if got := p.Expect(tokIdent); got != nil {
			t.Errorf("Expect while in error: got %v, want nil", got)
		}
		if len(p.Errs()) != 1 {
			t.Errorf("error count should stay 1, got %d", len(p.Errs()))
		}
	})
}

func TestParserExpectLit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "if"), tok(tokIdent, "x"))
		got := p.ExpectLit(tokIdent, "if")
		if got == nil || got.Lit != "if" {
			t.Fatalf("ExpectLit returned %v, want token if", got)
		}
		if p.Token().Lit != "x" {
			t.Errorf("current after ExpectLit: got %q, want x", p.Token().Lit)
		}
	})

	t.Run("wrong literal fails", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "else"))
		if got := p.ExpectLit(tokIdent, "if"); got != nil {
			t.Errorf("ExpectLit mismatch: got %v, want nil", got)
		}
		if !p.InError() {
			t.Errorf("InError after failure: got false, want true")
		}
	})

	t.Run("no-op while in error", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "if"))
		p.Jail()
		if got := p.ExpectLit(tokIdent, "if"); got != nil {
			t.Errorf("ExpectLit while in error: got %v, want nil", got)
		}
	})
}

func TestParserErrorHelpers(t *testing.T) {
	for _, test := range []struct {
		name string
		fn   func(p *Parser)
		code string
	}{
		{"Errorf", func(p *Parser) { p.Errorf(nil, "boom %d", 1) }, ""},
		{"ErrorfHere", func(p *Parser) { p.ErrorfHere("boom") }, ""},
		{"CodeErrorf", func(p *Parser) { p.CodeErrorf(nil, "c1", "boom") }, "c1"},
		{"CodeErrorfHere", func(p *Parser) { p.CodeErrorfHere("c2", "boom") }, "c2"},
	} {
		p := newTestParser(tok(tokIdent, "a"))
		test.fn(p)
		if !p.InError() {
			t.Errorf("%s: InError got false, want true", test.name)
		}
		errs := p.Errs()
		if len(errs) != 1 {
			t.Errorf("%s: got %d errors, want 1", test.name, len(errs))
			continue
		}
		if errs[0].Code != test.code {
			t.Errorf("%s: code got %q, want %q", test.name, errs[0].Code, test.code)
		}
	}
}

func TestParserJailAndBailOut(t *testing.T) {
	p := newTestParser(tok(tokIdent, "a"))
	if p.InError() {
		t.Errorf("fresh parser InError: got true, want false")
	}
	p.Jail()
	if !p.InError() {
		t.Errorf("after Jail InError: got false, want true")
	}
	p.BailOut()
	if p.InError() {
		t.Errorf("after BailOut InError: got true, want false")
	}
}

func TestParserTypeStr(t *testing.T) {
	p := newTestParser(tok(tokIdent, "a"))
	if got := p.TypeStr(EOF); got != "eof" {
		t.Errorf("TypeStr(EOF): got %q, want eof", got)
	}
	if got := p.TypeStr(99); got != "<T99>" {
		t.Errorf("TypeStr(99): got %q, want <T99>", got)
	}
}

func TestParserSkipErrStmt(t *testing.T) {
	t.Run("not in error is a no-op", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "a"), tok(tokSemi, ";"))
		if p.SkipErrStmt(tokSemi) {
			t.Errorf("SkipErrStmt while not in error: got true, want false")
		}
		if p.Token().Lit != "a" {
			t.Errorf("token moved during no-op skip: got %q", p.Token().Lit)
		}
	})

	t.Run("skips to separator and bails out", func(t *testing.T) {
		p := newTestParser(
			tok(tokIdent, "a"),
			tok(tokIdent, "b"),
			tok(tokSemi, ";"),
			tok(tokIdent, "c"),
		)
		p.Jail()
		if !p.SkipErrStmt(tokSemi) {
			t.Fatalf("SkipErrStmt: got false, want true")
		}
		if p.InError() {
			t.Errorf("still in error after SkipErrStmt")
		}
		if p.Token().Lit != "c" {
			t.Errorf("token after skip: got %q, want c", p.Token().Lit)
		}
	})

	t.Run("stops at EOF when separator is missing", func(t *testing.T) {
		p := newTestParser(tok(tokIdent, "a"), tok(tokIdent, "b"))
		p.Jail()
		if !p.SkipErrStmt(tokSemi) {
			t.Fatalf("SkipErrStmt: got false, want true")
		}
		if p.InError() {
			t.Errorf("still in error after SkipErrStmt")
		}
		if !p.See(EOF) {
			t.Errorf("token after skip: got %q, want EOF", p.Token().Lit)
		}
	})
}

func TestParserErrsFromTokener(t *testing.T) {
	// When the underlying tokener has errors, Errs returns those instead of
	// the parser's own error list.
	lexErr := &Error{Code: "lex"}
	src := &sliceTokener{toks: []*Token{tok(tokIdent, "a")}, errs: []*Error{lexErr}}
	p := NewParser(src, NewTypes())
	p.ErrorfHere("parser-level error")

	errs := p.Errs()
	if len(errs) != 1 || errs[0] != lexErr {
		t.Errorf("Errs should relay tokener errors, got %v", errs)
	}
}
