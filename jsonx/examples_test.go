package jsonx

import "testing"

// The tests in this file double as executable documentation of what the JSONx
// dialect does and does not accept. Each one converts a JSONx snippet to
// canonical JSON with ToJSON and checks either the resulting JSON or that the
// input is rejected. They are intentionally exhaustive about the edges where
// JSONx differs from plain JSON.

// checkOK asserts that in converts to the canonical JSON want.
func checkOK(t *testing.T, in, want string) {
	t.Helper()
	bs, errs := ToJSON([]byte(in))
	if len(errs) != 0 {
		t.Errorf("ToJSON(%q): unexpected error %v", in, errs)
		return
	}
	if got := string(bs); got != want {
		t.Errorf("ToJSON(%q): got %q, want %q", in, got, want)
	}
}

// checkErr asserts that in is rejected by the parser.
func checkErr(t *testing.T, in string) {
	t.Helper()
	if _, errs := ToJSON([]byte(in)); len(errs) == 0 {
		t.Errorf("ToJSON(%q): got no error, want one", in)
	}
}

// TestExampleComments documents that both line and block comments are
// accepted anywhere whitespace is, and are dropped from the output.
func TestExampleComments(t *testing.T) {
	checkOK(t, "42 // trailing line comment", "42")
	checkOK(t, "/* leading block */ 42", "42")
	checkOK(t, "{a:/* inline */ 42}", `{"a":42}`)
	checkOK(t, "{\n// a field\na:42,\n}", `{"a":42}`)
}

// TestExampleUnquotedKeys documents that object keys may be written as bare
// Go-style identifiers; quoted string keys remain valid too.
func TestExampleUnquotedKeys(t *testing.T) {
	checkOK(t, `{value:42}`, `{"value":42}`)
	checkOK(t, `{a:1,b:2}`, `{"a":1,"b":2}`)
	checkOK(t, `{"quoted":1}`, `{"quoted":1}`)
}

// TestExampleTrailingComma documents the end-of-line comma rule: a value that
// ends a line must be followed by a comma, so every entry in a multi-line
// object or array — including the last — carries a trailing comma. On a single
// line the closing bracket terminates the final entry, so none is needed.
func TestExampleTrailingComma(t *testing.T) {
	// Single line: closing bracket ends the last entry, trailing comma optional.
	checkOK(t, `{a:1,b:2}`, `{"a":1,"b":2}`)
	checkOK(t, `{a:1,b:2,}`, `{"a":1,"b":2}`)
	checkOK(t, `[1,2]`, `[1,2]`)
	checkOK(t, `[1,2,]`, `[1,2]`)

	// Multi-line: every entry must end with a comma.
	checkOK(t, "{\na:1,\nb:2,\n}", `{"a":1,"b":2}`)
	checkOK(t, "[\n1,\n2,\n]", `[1,2]`)

	// A value ending a line without a comma is rejected, even the last one.
	checkErr(t, "{\na:1\nb:2\n}")  // missing comma after a:1
	checkErr(t, "{\na:1,\nb:2\n}") // missing comma after last entry b:2
	checkErr(t, "[\n1\n2\n]")
}

// TestExampleStrings documents the two supported string forms and that single
// quotes are not a string syntax.
func TestExampleStrings(t *testing.T) {
	// Double-quoted strings work as in JSON, with escapes.
	checkOK(t, `"hi"`, `"hi"`)

	// Backtick raw strings work as in Go: no escapes, may span lines.
	checkOK(t, "`hi\nthere`", `"hi\nthere"`)

	// Single-quoted strings are not supported.
	checkErr(t, `'hi'`)
}

// TestExampleNumbers documents that a leading '+' or '-' sign is accepted on
// numbers; a bare '+' is normalized away.
func TestExampleNumbers(t *testing.T) {
	checkOK(t, `{a:-42}`, `{"a":-42}`)
	checkOK(t, `{a:+42}`, `{"a":42}`)
}

// TestExampleDottedIdent documents that a dotted identifier path is shorthand
// for an array of its segments.
func TestExampleDottedIdent(t *testing.T) {
	checkOK(t, `a.b.c.d`, `["a","b","c","d"]`)
}
