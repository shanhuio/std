package jsonx

import (
	"strings"
	"testing"

	"shanhu.io/std/lexing"
)

// rawTokens drives lexJSONX directly, before the semicolon inserter and
// keyworder transform the stream, so each test sees exactly what lex.go emits.
func rawTokens(t *testing.T, in string) ([]*lexing.Token, []*lexing.Error) {
	t.Helper()
	x := lexing.MakeLexer("test.jsonx", strings.NewReader(in), lexJSONX)
	return lexing.Tokens(x)
}

func TestLexJSONXTokens(t *testing.T) {
	type tok struct {
		typ int
		lit string
	}
	for _, test := range []struct {
		in   string
		want []tok
	}{
		// Single-char operators keep their literal.
		{"{", []tok{{tokOperator, "{"}}},
		{"}", []tok{{tokOperator, "}"}}},
		{"[", []tok{{tokOperator, "["}}},
		{"]", []tok{{tokOperator, "]"}}},
		{",", []tok{{tokOperator, ","}}},
		{":", []tok{{tokOperator, ":"}}},
		{"+", []tok{{tokOperator, "+"}}},
		{"-", []tok{{tokOperator, "-"}}},
		{".", []tok{{tokOperator, "."}}},
		// A lone '/' (not //, not /*) is just an operator.
		{"/", []tok{{tokOperator, "/"}}},
		// Semicolon and newline are their own token types.
		{";", []tok{{tokSemi, ";"}}},
		{"\n", []tok{{tokEndl, "\n"}}},
		// Strings: double-quoted and raw backtick.
		{`"abc"`, []tok{{tokString, `"abc"`}}},
		{"`abc`", []tok{{tokString, "`abc`"}}},
		// Numbers: integer vs float.
		{"42", []tok{{tokInt, "42"}}},
		{"4.2", []tok{{tokFloat, "4.2"}}},
		{"0xff", []tok{{tokInt, "0xff"}}},
		// Identifiers. Keyword mapping happens later, so true is an ident here.
		{"abc", []tok{{tokIdent, "abc"}}},
		{"true", []tok{{tokIdent, "true"}}},
		// Comments lex to a single Comment token.
		{"// hi", []tok{{lexing.Comment, "// hi"}}},
		{"/* hi */", []tok{{lexing.Comment, "/* hi */"}}},
		// A small mixed sequence exercises whitespace skipping and chaining.
		{"a:1", []tok{
			{tokIdent, "a"}, {tokOperator, ":"}, {tokInt, "1"},
		}},
	} {
		toks, errs := rawTokens(t, test.in)
		if len(errs) != 0 {
			t.Errorf("lex %q: unexpected errors %v", test.in, errs)
			continue
		}
		// Drop the trailing EOF token before comparing.
		if n := len(toks); n == 0 || toks[n-1].Type != lexing.EOF {
			t.Errorf("lex %q: missing trailing EOF", test.in)
			continue
		}
		got := toks[:len(toks)-1]
		if len(got) != len(test.want) {
			t.Errorf("lex %q: got %d tokens, want %d", test.in, len(got), len(test.want))
			continue
		}
		for i, w := range test.want {
			if got[i].Type != w.typ || got[i].Lit != w.lit {
				t.Errorf("lex %q token %d: got (%d,%q), want (%d,%q)",
					test.in, i, got[i].Type, got[i].Lit, w.typ, w.lit)
			}
		}
	}
}

func TestLexJSONXIllegalChar(t *testing.T) {
	toks, errs := rawTokens(t, "@")
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Code != "jsonx.illegalChar" {
		t.Errorf("error code: got %q, want jsonx.illegalChar", errs[0].Code)
	}
	if toks[0].Type != lexing.Illegal {
		t.Errorf("first token type: got %d, want Illegal", toks[0].Type)
	}
}

func TestLexJSONXPanicsOnWhiteStart(t *testing.T) {
	// lexJSONX must be called on a non-white rune; the lexer normally
	// guarantees this by skipping whitespace first.
	x := lexing.MakeLexer("test.jsonx", strings.NewReader("   "), lexJSONX)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("lexJSONX on whitespace: got no panic, want panic")
		}
	}()
	lexJSONX(x)
}

// TestTokenerKeywordsAndSemi checks the full tokener chain: keywords are
// recognized and a semicolon is inserted at a line break after a value.
func TestTokenerKeywordsAndSemi(t *testing.T) {
	x := tokener("test.jsonx", strings.NewReader("true\nfalse"))
	toks, errs := lexing.Tokens(x)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	var got []struct {
		typ int
		lit string
	}
	for _, tk := range toks {
		if tk.Type == lexing.EOF {
			continue
		}
		got = append(got, struct {
			typ int
			lit string
		}{tk.Type, tk.Lit})
	}

	want := []struct {
		typ int
		lit string
	}{
		{tokKeyword, "true"},
		{tokSemi, "\n"}, // newline after a value inserts a semicolon
		{tokKeyword, "false"},
		{tokSemi, ""}, // EOF after a value inserts a final semicolon too
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens %v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		if got[i].typ != w.typ || got[i].lit != w.lit {
			t.Errorf("token %d: got (%d,%q), want (%d,%q)",
				i, got[i].typ, got[i].lit, w.typ, w.lit)
		}
	}
}
