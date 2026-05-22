package docker_test

import (
	"fmt"
	"net/http"
	"testing"

	"shanhu.io/std/docker/dockertest"
)

func TestPing(t *testing.T) {
	d := dockertest.New(t)

	var gotMethod, gotPath string
	d.Handle("GET", "/_ping", func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		fmt.Fprint(w, "OK")
	})

	if err := d.Client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method: got %q, want GET", gotMethod)
	}
	if want := "/v1.40/_ping"; gotPath != want {
		t.Errorf("path: got %q, want %q", gotPath, want)
	}
}
