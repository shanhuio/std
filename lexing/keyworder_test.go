package lexing

import (
	"testing"
)

func TestKeywordSet(t *testing.T) {
	set := KeywordSet("if", "else", "for")
	for _, w := range []string{"if", "else", "for"} {
		if _, ok := set[w]; !ok {
			t.Errorf("KeywordSet missing %q", w)
		}
	}
	if _, ok := set["while"]; ok {
		t.Errorf("KeywordSet contains unexpected %q", "while")
	}
	if got := len(KeywordSet()); got != 0 {
		t.Errorf("empty KeywordSet: got len %d, want 0", got)
	}
}

func TestKeyworder(t *testing.T) {
	src := newStaticTokener(
		tok(tokIdent, "if"),
		tok(tokIdent, "x"),
		tok(tokIdent, "else"),
	)

	kw := NewKeyworder(src)
	kw.Keywords = KeywordSet("if", "else")
	kw.Ident = tokIdent
	kw.Keyword = tokKeyword

	got := TokenAll(kw)

	want := []struct {
		lit string
		typ int
	}{
		{"if", tokKeyword},
		{"x", tokIdent},
		{"else", tokKeyword},
	}
	for i, w := range want {
		if got[i].Lit != w.lit || got[i].Type != w.typ {
			t.Errorf("token %d: got (%q,%d), want (%q,%d)",
				i, got[i].Lit, got[i].Type, w.lit, w.typ)
		}
	}
}

func TestKeyworderNilKeywords(t *testing.T) {
	// With no keyword set, every ident passes through unchanged.
	src := newStaticTokener(tok(tokIdent, "if"))
	kw := NewKeyworder(src)
	kw.Ident = tokIdent
	kw.Keyword = tokKeyword

	got := kw.Token()
	if got.Type != tokIdent {
		t.Errorf("type got %d, want %d (unchanged)", got.Type, tokIdent)
	}
}

func TestKeyworderNonIdentUnchanged(t *testing.T) {
	// A token whose type is not Ident is never reclassified, even if its
	// literal is in the keyword set.
	src := newStaticTokener(tok(tokKeyword, "if"))
	kw := NewKeyworder(src)
	kw.Keywords = KeywordSet("if")
	kw.Ident = tokIdent
	kw.Keyword = tokKeyword

	got := kw.Token()
	if got.Type != tokKeyword {
		t.Errorf("type got %d, want %d (unchanged)", got.Type, tokKeyword)
	}
}

func TestKeyworderErrsRelayed(t *testing.T) {
	wantErr := &Error{Code: "x"}
	src := newStaticTokener()
	src.setErrors(wantErr)
	kw := NewKeyworder(src)

	errs := kw.Errs()
	if len(errs) != 1 || errs[0] != wantErr {
		t.Errorf("Errs not relayed: got %v", errs)
	}
}
