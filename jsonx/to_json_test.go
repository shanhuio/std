package jsonx

import "testing"

// The tests in this file convert JSONx snippets to canonical JSON with ToJSON
// and double as executable documentation of the dialect: each checkOK pins
// down an accepted form and its JSON output, and each checkErr pins down an
// input the parser rejects. They are intentionally exhaustive about the edges
// where JSONx differs from plain JSON.

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

// TestScalars documents that JSON scalars pass through unchanged.
func TestScalars(t *testing.T) {
	checkOK(t, `1234`, `1234`)
	checkOK(t, `true`, `true`)
	checkOK(t, `false`, `false`)
	checkOK(t, `null`, `null`)
	checkOK(t, `"string"`, `"string"`)

	// An empty input has no value and is an error.
	checkErr(t, ``)
	// An unknown character is rejected.
	checkErr(t, `@`)
	// A bare or dotted identifier is not a value.
	checkErr(t, `foo`)
	checkErr(t, `a.b.c`)
}

// TestComments documents that both line and block comments are accepted
// anywhere whitespace is, and are dropped from the output.
func TestComments(t *testing.T) {
	checkOK(t, "42 // trailing line comment", "42")
	checkOK(t, "/* leading block */ 42", "42")
	checkOK(t, "{a:/* inline */ 42}", `{"a":42}`)
	checkOK(t, "{\n// a field\na:42,\n}", `{"a":42}`)
}

// TestUnquotedKeys documents that object keys may be written as bare Go-style
// identifiers; quoted string keys remain valid too.
func TestUnquotedKeys(t *testing.T) {
	checkOK(t, `{value:42}`, `{"value":42}`)
	checkOK(t, `{a:1,b:2}`, `{"a":1,"b":2}`)
	checkOK(t, `{"quoted":1}`, `{"quoted":1}`)
}

// TestObjectsAndLists documents object and list values, including nesting and
// the empty object, plus the malformed forms the parser rejects.
func TestObjectsAndLists(t *testing.T) {
	checkOK(t, `{}`, `{}`)
	checkOK(t, `{value:null}`, `{"value":null}`)
	checkOK(t, `{bool:false}`, `{"bool":false}`)
	checkOK(t, `{a:42,b:true}`, `{"a":42,"b":true}`)
	checkOK(t, `{a:{a:{a:42}}}`, `{"a":{"a":{"a":42}}}`)
	checkOK(t, `{"a":"a","b":"b"}`, `{"a":"a","b":"b"}`)

	checkErr(t, `}`)       // not an operand
	checkErr(t, `{a:}`)    // missing value
	checkErr(t, `{a 1}`)   // missing colon between key and value
	checkErr(t, `{1:2}`)   // key is neither identifier nor string
	checkErr(t, `{a:1`)    // unterminated object
	checkErr(t, `[1 2]`)   // missing comma between list entries
	checkErr(t, `[1,2`)    // unterminated list
	checkErr(t, `{a:1,,}`) // stray comma where an entry is expected
}

// TestTrailingComma documents the end-of-line comma rule: a value that ends a
// line must be followed by a comma, so every entry in a multi-line object or
// array — including the last — carries a trailing comma. On a single line the
// closing bracket terminates the final entry, so none is needed.
func TestTrailingComma(t *testing.T) {
	// Single line: closing bracket ends the last entry, trailing comma optional.
	checkOK(t, `{a:1,b:2}`, `{"a":1,"b":2}`)
	checkOK(t, `{a:42,}`, `{"a":42}`)
	checkOK(t, `[1,2]`, `[1,2]`)
	checkOK(t, `[1,2,]`, `[1,2]`)

	// Multi-line: every entry must end with a comma.
	checkOK(t, "{a:42,\n}", `{"a":42}`)
	checkOK(t, "{\na:42,\n}", `{"a":42}`)
	checkOK(t, "{\na:1,\nb:2,\n}", `{"a":1,"b":2}`)
	checkOK(t, "[\n1,\n2,\n]", `[1,2]`)

	// A value ending a line without a comma is rejected, even the last one.
	checkErr(t, "{\na:1\nb:2\n}")  // missing comma after a:1
	checkErr(t, "{\na:1,\nb:2\n}") // missing comma after last entry b:2
	checkErr(t, "[\n1\n2\n]")
}

// TestStrings documents the two supported string forms and that single quotes
// are not a string syntax.
func TestStrings(t *testing.T) {
	// Double-quoted strings work as in JSON, with escapes.
	checkOK(t, `"hi"`, `"hi"`)

	// Backtick raw strings work as in Go: no escapes, may span lines.
	checkOK(t, "`hi\nthere`", `"hi\nthere"`)
	checkOK(t, "`string\n`", `"string\n"`)

	checkErr(t, `'hi'`)          // single-quoted strings are not supported
	checkErr(t, `"unterminated`) // unterminated string
}

// TestNumbers documents that a leading '+' or '-' sign is accepted on numbers;
// a bare '+' is normalized away. A sign must be followed by a number.
func TestNumbers(t *testing.T) {
	checkOK(t, `{a:-42}`, `{"a":-42}`)
	checkOK(t, `{a:+42}`, `{"a":42}`)
	checkErr(t, `+x`)
}
