package lexing

import (
	"testing"
)

func TestTokenString(t *testing.T) {
	for _, test := range []struct {
		tok  *Token
		want string
	}{
		{
			&Token{Type: 1, Lit: "foo", Pos: &Pos{File: "a.go", Line: 2, Col: 3}},
			"'foo' (a.go:2:3)",
		},
		{
			&Token{Type: 1, Lit: "bar", Pos: &Pos{Line: 5, Col: 1}},
			"'bar' (5:1)",
		},
		{
			&Token{Type: EOF, Lit: "", Pos: &Pos{File: "a.go", Line: 9, Col: 9}},
			"'' (a.go:9:9)",
		},
	} {
		if got := test.tok.String(); got != test.want {
			t.Errorf("Token.String(): got %q, want %q", got, test.want)
		}
	}
}
