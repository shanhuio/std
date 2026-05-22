package dockertest

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func setJSONContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

// writeJSON sets the JSON content type and encodes body to w.
func writeJSON(w http.ResponseWriter, body any) error {
	setJSONContentType(w)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	return nil
}

// readJSON decodes the JSON body of r into target. On error it writes a 400
// Bad Request response and returns false; the caller should return.
func readJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		http.Error(w, "decode body: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeNotFound(w http.ResponseWriter, msg string) {
	setJSONContentType(w)
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"message":%q}`, msg)
}
