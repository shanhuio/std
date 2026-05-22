// Package dockertest provides a fake Docker daemon for testing docker.Client.
package dockertest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"shanhu.io/std/docker"
)

// FakeDaemon is an in-process stand-in for the Docker daemon. It listens on
// a unix domain socket inside the test's temporary directory, the same shape
// as a real local Docker daemon. Tests register route handlers on it and
// exercise the exported docker.Client API.
type FakeDaemon struct {
	t      *testing.T
	mux    *http.ServeMux
	server *http.Server

	// SockPath is the unix socket the fake daemon listens on.
	SockPath string

	// Client is a docker.Client wired to talk to this fake daemon.
	Client *docker.Client
}

// New starts a FakeDaemon with no routes registered. The server is shut down
// when the test ends.
func New(t *testing.T) *FakeDaemon {
	t.Helper()
	sockPath := filepath.Join(t.TempDir(), "docker.sock")
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen unix %q: %v", sockPath, err)
	}

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	return &FakeDaemon{
		t:        t,
		mux:      mux,
		server:   srv,
		SockPath: sockPath,
		Client:   docker.NewUnixClient(sockPath),
	}
}

// Handle registers a handler for `method p` under docker.APIVersion. p must
// start with "/".
func (d *FakeDaemon) Handle(method, p string, h http.HandlerFunc) {
	d.t.Helper()
	d.mux.HandleFunc(method+" "+docker.APIVersion+p, h)
}

// HandleJSON registers a route that returns body encoded as JSON with 200 OK.
func (d *FakeDaemon) HandleJSON(method, p string, body any) {
	d.t.Helper()
	d.Handle(method, p, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			d.t.Errorf("encode response for %s %s: %v", method, p, err)
		}
	})
}

// HandleError registers a route that returns the given HTTP status with a
// Docker-style JSON error body.
func (d *FakeDaemon) HandleError(method, p string, status int, msg string) {
	d.t.Helper()
	d.Handle(method, p, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"message":%q}`, msg)
	})
}
