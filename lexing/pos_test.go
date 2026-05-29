package lexing

import (
	"testing"
)

func TestPosString(t *testing.T) {
	for _, test := range []struct {
		pos  *Pos
		want string
	}{
		{&Pos{File: "main.go", Line: 12, Col: 5}, "main.go:12:5"},
		{&Pos{Line: 3, Col: 7}, "3:7"},
		{&Pos{File: "a/b.go", Line: 1, Col: 1}, "a/b.go:1:1"},
		{&Pos{}, "0:0"},
	} {
		if got := test.pos.String(); got != test.want {
			t.Errorf("Pos%+v.String(): got %q, want %q", *test.pos, got, test.want)
		}
	}
}
