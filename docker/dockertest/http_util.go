package dockertest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func setJSONContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

// writeJSON sets the JSON content type and encodes body to w. Any encode
// error is reported via t.Errorf.
func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	setJSONContentType(w)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func writeNotFound(w http.ResponseWriter, msg string) {
	setJSONContentType(w)
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"message":%q}`, msg)
}
