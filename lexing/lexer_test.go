package lexing

import (
	"errors"
	"strings"
	"testing"
)

// errReader returns a fixed error on every Read.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestLexerTokenNilLexFunc(t *testing.T) {
	// With no LexFunc set, Token emits an Illegal token for each rune.
	x := NewLexer("test.txt", strings.NewReader("ab"))

	tok := x.Token()
	if tok.Type != Illegal {
		t.Errorf("first token type: got %d, want Illegal", tok.Type)
	}

	// The stream still terminates with EOF.
	tok = x.Token()
	if tok.Type != Illegal {
		t.Errorf("second token type: got %d, want Illegal", tok.Type)
	}
	if eof := x.Token(); eof.Type != EOF {
		t.Errorf("final token type: got %d, want EOF", eof.Type)
	}
}

func TestLexerErrsReadError(t *testing.T) {
	// A non-EOF read error surfaces through Errs as a single error wrapping
	// the underlying reader error.
	sentinel := errors.New("read failed")
	x := NewLexer("test.txt", errReader{err: sentinel})

	errs := x.Errs()
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Err != sentinel {
		t.Errorf("Errs[0].Err: got %v, want %v", errs[0].Err, sentinel)
	}
}

func TestLexerErrsCleanEOF(t *testing.T) {
	// A clean EOF is not reported as an error.
	x := NewLexer("test.txt", strings.NewReader(""))
	if tok := x.Token(); tok.Type != EOF {
		t.Fatalf("token on empty input: got %d, want EOF", tok.Type)
	}
	if errs := x.Errs(); errs != nil {
		t.Errorf("clean EOF Errs: got %v, want nil", errs)
	}
}
