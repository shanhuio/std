package jsonx

import (
	"errors"
	"testing"

	"shanhu.io/std/lexing"
)

// fakeTokener emits a fixed list of tokens, then EOF forever. It lets the
// semi-inserter tests feed exact token streams without going through the lexer.
type fakeTokener struct {
	toks []*lexing.Token
	i    int
	errs []*lexing.Error
}

func (f *fakeTokener) Token() *lexing.Token {
	if f.i >= len(f.toks) {
		return &lexing.Token{Type: lexing.EOF}
	}
	t := f.toks[f.i]
	f.i++
	return t
}

func (f *fakeTokener) Errs() []*lexing.Error { return f.errs }

func mkTok(typ int, lit string) *lexing.Token {
	return &lexing.Token{Type: typ, Lit: lit}
}

// runSemi feeds toks through the semi-inserter and returns the resulting
// stream, including the trailing EOF.
func runSemi(toks ...*lexing.Token) []*lexing.Token {
	si := newSemiInserter(&fakeTokener{toks: toks})
	var out []*lexing.Token
	for {
		t := si.Token()
		out = append(out, t)
		if t.Type == lexing.EOF {
			break
		}
	}
	return out
}

type wantTok struct {
	typ int
	lit string
}

func checkStream(t *testing.T, got []*lexing.Token, want []wantTok) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d tokens %v, want %d", len(got), dumpToks(got), len(want))
	}
	for i, w := range want {
		if got[i].Type != w.typ || got[i].Lit != w.lit {
			t.Errorf("token %d: got (%d,%q), want (%d,%q)",
				i, got[i].Type, got[i].Lit, w.typ, w.lit)
		}
	}
}

func dumpToks(toks []*lexing.Token) []wantTok {
	out := make([]wantTok, len(toks))
	for i, t := range toks {
		out[i] = wantTok{t.Type, t.Lit}
	}
	return out
}

// A value followed by a newline turns that newline into a semicolon.
func TestSemiInserterEndlAfterValue(t *testing.T) {
	got := runSemi(
		mkTok(tokInt, "1"),
		mkTok(tokEndl, "\n"),
		mkTok(tokInt, "2"),
	)
	checkStream(t, got, []wantTok{
		{tokInt, "1"},
		{tokSemi, "\n"},
		{tokInt, "2"},
		{tokSemi, ""}, // EOF after a value inserts a final semicolon
		{lexing.EOF, ""},
	})
}

// A newline with no preceding value (insertSemi is false) is dropped.
func TestSemiInserterEndlWithoutValue(t *testing.T) {
	got := runSemi(
		mkTok(tokEndl, "\n"),
		mkTok(tokInt, "1"),
	)
	checkStream(t, got, []wantTok{
		{tokInt, "1"},
		{tokSemi, ""},
		{lexing.EOF, ""},
	})
}

// Closing brackets request a semicolon; other operators suppress it.
func TestSemiInserterOperators(t *testing.T) {
	// "}" -> a following newline becomes a semicolon.
	got := runSemi(
		mkTok(tokOperator, "}"),
		mkTok(tokEndl, "\n"),
	)
	checkStream(t, got, []wantTok{
		{tokOperator, "}"},
		{tokSemi, "\n"},
		{lexing.EOF, ""},
	})

	// "{" -> a following newline is dropped.
	got = runSemi(
		mkTok(tokOperator, "{"),
		mkTok(tokEndl, "\n"),
	)
	checkStream(t, got, []wantTok{
		{tokOperator, "{"},
		{lexing.EOF, ""},
	})
}

// An explicit semicolon passes through and clears the pending insert, so a
// following newline is dropped rather than doubled.
func TestSemiInserterExplicitSemi(t *testing.T) {
	got := runSemi(
		mkTok(tokInt, "1"),
		mkTok(tokSemi, ";"),
		mkTok(tokEndl, "\n"),
	)
	checkStream(t, got, []wantTok{
		{tokInt, "1"},
		{tokSemi, ";"},
		{lexing.EOF, ""},
	})
}

// Comments are passed through without disturbing the pending-semicolon state,
// so a value-comment-newline sequence still inserts a semicolon.
func TestSemiInserterComment(t *testing.T) {
	got := runSemi(
		mkTok(tokInt, "1"),
		mkTok(lexing.Comment, "// hi"),
		mkTok(tokEndl, "\n"),
	)
	checkStream(t, got, []wantTok{
		{tokInt, "1"},
		{lexing.Comment, "// hi"},
		{tokSemi, "\n"},
		{lexing.EOF, ""},
	})
}

// With no pending semicolon, EOF is returned directly with nothing inserted.
func TestSemiInserterEOFNoInsert(t *testing.T) {
	got := runSemi(mkTok(tokOperator, "{"))
	checkStream(t, got, []wantTok{
		{tokOperator, "{"},
		{lexing.EOF, ""},
	})
}

// Errs are relayed from the underlying tokener.
func TestSemiInserterErrsRelayed(t *testing.T) {
	want := errors.New("boom")
	si := newSemiInserter(&fakeTokener{
		errs: []*lexing.Error{{Err: want}},
	})
	errs := si.Errs()
	if len(errs) != 1 || errs[0].Err != want {
		t.Errorf("Errs: got %v, want one error wrapping %v", errs, want)
	}
}
