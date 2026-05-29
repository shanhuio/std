package lexing

import (
	"testing"

	"strings"
)

func TestLexComment(t *testing.T) {
	for _, s := range []string{
		"",
		"     ",
		"// abc",
		"/*abc*/",
		"   // abc",
		"   /* abc \n abc */",
	} {
		t.Logf("parsing %q", s)
		x := NewCommentLexer("a.txt", strings.NewReader(s))
		toks, errs := Tokens(x)
		if len(errs) != 0 {
			t.Errorf("unxpected errors: %v", errs)
		}
		if len(toks) > 2 {
			t.Errorf("want 0 or 1 token, got %d", len(toks)-1)
		}
	}

	for _, s := range []string{
		"/*//*a/",    // bad end
		"/*\n\n",     // bad end
		"//abc\nabc", // cross line
		"/*abc*/abc", // closed
		"/abc",       // invalid start
		"/",          //
	} {
		t.Logf("parsing %q", s)
		x := NewCommentLexer("a.txt", strings.NewReader(s))
		_, errs := Tokens(x)
		if len(errs) == 0 {
			t.Errorf("expect error, but got nothing")
		}
	}
}

func TestLexCommentPanicsWithoutSlash(t *testing.T) {
	// LexComment requires a '/' already buffered as a precondition.
	x := NewLexer("a.txt", strings.NewReader("abc"))

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("LexComment without buffered '/': got no panic, want panic")
		}
	}()
	LexComment(x)
}
