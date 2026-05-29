package lexing

import (
	"strings"
	"testing"
)

const strType = 1

// lexSingle lexes input with lex func f, returning the single token and any
// lexing errors.
func lexSingle(input string, f LexFunc) (*Token, []*Error) {
	x := MakeLexer("test.txt", strings.NewReader(input), f)
	tok := x.Token()
	return tok, x.Errs()
}

func lexStr(input string, q rune) (*Token, []*Error) {
	return lexSingle(input, func(x *Lexer) *Token {
		return LexString(x, strType, q)
	})
}

func TestLexStringValid(t *testing.T) {
	for _, input := range []string{
		`"hello"`,
		`""`,
		`"a\tb\n"`,
		`"\x41"`,
		`"\101"`,
		`"A"`,
		`"\U00000041"`,
		`"\\"`,
		`"quote \" inside"`,
	} {
		tok, errs := lexStr(input, '"')
		if len(errs) != 0 {
			t.Errorf("LexString(%q): unexpected errors %v", input, errs)
			continue
		}
		if tok.Lit != input {
			t.Errorf("LexString(%q): lit got %q", input, tok.Lit)
		}
	}
}

func TestLexStringErrors(t *testing.T) {
	for _, test := range []struct {
		input string
		code  string
	}{
		{`"abc`, "lexing.unexpectedEOF"},
		{"\"ab\ncd\"", "lexing.unexpectedEndl"},
		{`"\q"`, "lexing.unknownESC"},
	} {
		_, errs := lexStr(test.input, '"')
		if len(errs) == 0 {
			t.Errorf("LexString(%q): want error %q, got none", test.input, test.code)
			continue
		}
		if errs[0].Code != test.code {
			t.Errorf("LexString(%q): code got %q, want %q",
				test.input, errs[0].Code, test.code)
		}
	}
}

func TestLexStringEscapeErrors(t *testing.T) {
	// These report human-friendly messages without a specific code.
	for _, input := range []string{
		`"\xZZ"`,   // illegal escape char
		`"\uD800"`, // surrogate, invalid code point
		`"\`,       // escape not terminated at EOF
	} {
		_, errs := lexStr(input, '"')
		if len(errs) == 0 {
			t.Errorf("LexString(%q): want an error, got none", input)
		}
	}
}

func TestLexStringCharLiteral(t *testing.T) {
	t.Run("single char ok", func(t *testing.T) {
		_, errs := lexStr(`'a'`, '\'')
		if len(errs) != 0 {
			t.Errorf("LexString('a'): unexpected errors %v", errs)
		}
	})

	for _, input := range []string{`''`, `'ab'`} {
		_, errs := lexStr(input, '\'')
		if len(errs) == 0 {
			t.Errorf("LexString(%q): want illegal char literal error", input)
			continue
		}
		if errs[0].Code != "lexing.illegalCharLit" {
			t.Errorf("LexString(%q): code got %q, want lexing.illegalCharLit",
				input, errs[0].Code)
		}
	}
}

func TestLexStringPanics(t *testing.T) {
	t.Run("unsupported quote", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("got no panic, want panic")
			}
		}()
		x := NewLexer("test.txt", strings.NewReader("|x|"))
		LexString(x, strType, '|')
	})

	t.Run("not at quote", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("got no panic, want panic")
			}
		}()
		x := NewLexer("test.txt", strings.NewReader("abc"))
		LexString(x, strType, '"')
	})
}

func TestLexRawString(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		input := "`raw\nstring`"
		tok, errs := lexSingle(input, func(x *Lexer) *Token {
			return LexRawString(x, strType)
		})
		if len(errs) != 0 {
			t.Errorf("LexRawString: unexpected errors %v", errs)
		}
		if tok.Lit != input {
			t.Errorf("LexRawString: lit got %q, want %q", tok.Lit, input)
		}
	})

	t.Run("unterminated", func(t *testing.T) {
		_, errs := lexSingle("`unterminated", func(x *Lexer) *Token {
			return LexRawString(x, strType)
		})
		if len(errs) == 0 || errs[0].Code != "lexing.unexpectedEOF" {
			t.Errorf("LexRawString: want lexing.unexpectedEOF, got %v", errs)
		}
	})

	t.Run("not at backtick panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("got no panic, want panic")
			}
		}()
		x := NewLexer("test.txt", strings.NewReader("abc"))
		LexRawString(x, strType)
	})
}
