package dockertest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetJSONContentType(t *testing.T) {
	w := httptest.NewRecorder()
	setJSONContentType(w)
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", got, "application/json")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	body := map[string]any{"name": "alpha", "n": 3}
	if err := writeJSON(w, body); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", got, "application/json")
	}
	got := strings.TrimSpace(w.Body.String())
	want := `{"n":3,"name":"alpha"}`
	if got != want {
		t.Errorf("body: got %q, want %q", got, want)
	}
}

func TestWriteJSONEncodeError(t *testing.T) {
	w := httptest.NewRecorder()
	// chan values cannot be marshaled by encoding/json.
	err := writeJSON(w, make(chan int))
	if err == nil {
		t.Fatal("writeJSON: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "encode response") {
		t.Errorf("error: got %q, want it to mention 'encode response'", err)
	}
}

func TestWriteNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeNotFound(w, "thing missing")
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", got, "application/json")
	}
	got := strings.TrimSpace(w.Body.String())
	want := `{"message":"thing missing"}`
	if got != want {
		t.Errorf("body: got %q, want %q", got, want)
	}
}
