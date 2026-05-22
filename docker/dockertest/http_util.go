package dockertest

import (
	"fmt"
	"net/http"
)

func writeNotFound(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"message":%q}`, msg)
}
