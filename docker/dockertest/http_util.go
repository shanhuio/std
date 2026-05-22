package dockertest

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func setJSONContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

// writeJSON sets the JSON content type and encodes body to w. The returned
// error, if any, is intended to be passed to FakeDaemon.recordErr by the
// caller.
func writeJSON(w http.ResponseWriter, body any) error {
	setJSONContentType(w)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	return nil
}

func writeNotFound(w http.ResponseWriter, msg string) {
	setJSONContentType(w)
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"message":%q}`, msg)
}
