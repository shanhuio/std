package jsonx

import (
	"bytes"
	"errors"
	"testing"

	"shanhu.io/std/lexing"
)

// Token and value constructors for building AST nodes directly, so the encode
// functions can be tested without going through the parser.
func opTok(lit string) *lexing.Token  { return &lexing.Token{Type: tokOperator, Lit: lit} }
func kwTok(lit string) *lexing.Token  { return &lexing.Token{Type: tokKeyword, Lit: lit} }
func intTok(lit string) *lexing.Token { return &lexing.Token{Type: tokInt, Lit: lit} }

func basicInt(lit string) *basic { return &basic{token: intTok(lit)} }

func identKey(name string) *objectKey {
	return &objectKey{token: &lexing.Token{Type: tokIdent, Lit: name}}
}

// twoEntryObject has both an identifier key and a string key, plus a comma
// between entries, so encoding it touches every branch of encodeObject.
func twoEntryObject() *object {
	return &object{entries: []*objectEntry{
		{key: identKey("a"), value: basicInt("1")},
		{
			key:   &objectKey{token: &lexing.Token{Type: tokString, Lit: `"b"`}, value: "b"},
			value: basicInt("2"),
		},
	}}
}

func twoEntryList() *list {
	return &list{entries: []*listEntry{
		{value: basicInt("1")},
		{value: basicInt("2")},
	}}
}

func TestEncodeValue(t *testing.T) {
	for _, test := range []struct {
		name string
		v    value
		want string
	}{
		{"null", &null{token: kwTok("null")}, "null"},
		{"bool true", &boolean{keyword: kwTok("true")}, "true"},
		{"bool false", &boolean{keyword: kwTok("false")}, "false"},
		{"int", basicInt("42"), "42"},
		{"negative int", &basic{lead: opTok("-"), token: intTok("42")}, "-42"},
		{"positive int", &basic{lead: opTok("+"), token: intTok("42")}, "42"},
		{
			"float",
			&basic{token: &lexing.Token{Type: tokFloat, Lit: "1.5"}, value: 1.5},
			"1.5",
		},
		{
			"string",
			&basic{token: &lexing.Token{Type: tokString, Lit: `"x"`}, value: "x"},
			`"x"`,
		},
		{"empty object", &object{}, "{}"},
		{"object", twoEntryObject(), `{"a":1,"b":2}`},
		{"empty list", &list{}, "[]"},
		{"list", twoEntryList(), "[1,2]"},
	} {
		var buf bytes.Buffer
		if err := encodeValue(&buf, test.v); err != nil {
			t.Errorf("%s: encodeValue: %v", test.name, err)
			continue
		}
		if got := buf.String(); got != test.want {
			t.Errorf("%s: got %q, want %q", test.name, got, test.want)
		}
	}
}

func TestEncodeValueErrors(t *testing.T) {
	t.Run("unexpected basic token", func(t *testing.T) {
		// A basic whose token is neither int, float, nor string is invalid.
		if err := encodeValue(&bytes.Buffer{}, &basic{token: opTok("+")}); err == nil {
			t.Errorf("got nil error, want one")
		}
	})

	t.Run("invalid value type", func(t *testing.T) {
		// encodeValue's default branch rejects anything that is not an AST node.
		if err := encodeValue(&bytes.Buffer{}, "not a node"); err == nil {
			t.Errorf("got nil error, want one")
		}
	})

	t.Run("unmarshalable basic value", func(t *testing.T) {
		// encodeJSON fails when the value cannot be marshaled to JSON.
		v := &basic{token: &lexing.Token{Type: tokFloat}, value: make(chan int)}
		if err := encodeValue(&bytes.Buffer{}, v); err == nil {
			t.Errorf("got nil error, want one")
		}
	})
}

// failAfter writes successfully n times, then fails on every subsequent write.
type failAfter struct{ n int }

func (f *failAfter) Write(p []byte) (int, error) {
	if f.n <= 0 {
		return 0, errors.New("write failed")
	}
	f.n--
	return len(p), nil
}

// countWriter counts how many Write calls a value's encoding makes.
type countWriter struct{ n int }

func (c *countWriter) Write(p []byte) (int, error) {
	c.n++
	return len(p), nil
}

// TestEncodeWriteErrors makes the writer fail at each successive write position
// for several value shapes, exercising every write-error return in the encode
// functions.
func TestEncodeWriteErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		v    value
	}{
		{"null", &null{token: kwTok("null")}},
		{"bool", &boolean{keyword: kwTok("true")}},
		{"negative int", &basic{lead: opTok("-"), token: intTok("42")}},
		{"float", &basic{token: &lexing.Token{Type: tokFloat, Lit: "1.5"}, value: 1.5}},
		{"object", twoEntryObject()},
		{"list", twoEntryList()},
	} {
		// Find how many writes a successful encoding performs.
		cw := new(countWriter)
		if err := encodeValue(cw, test.v); err != nil {
			t.Fatalf("%s: unexpected error counting writes: %v", test.name, err)
		}

		// Failing at any of those positions must surface as an error.
		for n := 0; n < cw.n; n++ {
			if err := encodeValue(&failAfter{n: n}, test.v); err == nil {
				t.Errorf("%s: failing at write %d: got nil error, want one",
					test.name, n)
			}
		}
	}
}
