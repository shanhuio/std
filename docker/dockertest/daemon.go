// Package dockertest provides a fake Docker daemon for testing docker.Client.
package dockertest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"shanhu.io/std/docker"
)

// FakeDaemon is an in-process Docker daemon stand-in. It implements (a subset
// of) the Docker Engine API and listens on a per-test unix domain socket the
// same shape as a real local daemon. Tests configure its state via the
// helper methods and exercise behavior through the exported docker.Client
// API; they should not register HTTP handlers on it directly.
type FakeDaemon struct {
	t      *testing.T
	mux    *http.ServeMux
	server *http.Server

	// SockPath is the unix socket the fake daemon listens on.
	SockPath string

	// Client is a docker.Client wired to talk to this fake daemon.
	Client *docker.Client

	mu      sync.Mutex
	version docker.VersionInfo
	conts   []*Container
	nets    []*Network
}

// New starts a FakeDaemon. The server is shut down when the test ends.
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

	d := &FakeDaemon{
		t:        t,
		mux:      mux,
		server:   srv,
		SockPath: sockPath,
		Client:   docker.NewUnixClient(sockPath),
	}
	d.registerRoutes()
	return d
}

func (d *FakeDaemon) handle(method, p string, h http.HandlerFunc) {
	d.mux.HandleFunc(method+" "+path.Join(docker.APIVersion, p), h)
}

func (d *FakeDaemon) registerRoutes() {
	d.handle("GET", "_ping", d.servePing)
	d.handle("HEAD", "_ping", d.servePing)
	d.handle("GET", "version", d.serveVersion)
	d.handle("GET", "containers/json", d.serveListContainers)
	d.handle("GET", "networks/{name}", d.serveInspectNetwork)
}

// SetVersion sets the version info returned by GET /version.
func (d *FakeDaemon) SetVersion(v docker.VersionInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.version = v
}

func (d *FakeDaemon) getVersion() docker.VersionInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.version
}

// AddContainer adds c to the set of containers known to the fake daemon. The
// pointer is stored as-is; later mutations to *c are visible to the fake.
func (d *FakeDaemon) AddContainer(c *Container) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.conts = append(d.conts, c)
}

// AddNetwork adds n to the set of networks known to the fake daemon. The
// pointer is stored as-is; later mutations to *n are visible to the fake.
func (d *FakeDaemon) AddNetwork(n *Network) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nets = append(d.nets, n)
}

// getNetwork returns the network matching nameOrID by Name or ID, or nil if
// none match.
func (d *FakeDaemon) getNetwork(nameOrID string) *Network {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, n := range d.nets {
		if n.Name == nameOrID || n.ID == nameOrID {
			return n
		}
	}
	return nil
}

func (d *FakeDaemon) servePing(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "OK")
}

func (d *FakeDaemon) serveVersion(w http.ResponseWriter, _ *http.Request) {
	v := d.getVersion()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		d.t.Errorf("encode version: %v", err)
	}
}

func (d *FakeDaemon) getContainers(wantLabels []string) []*docker.ContListInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	matched := make([]*docker.ContListInfo, 0, len(d.conts))
	for _, c := range d.conts {
		if matchesAllLabels(c.Labels, wantLabels) {
			matched = append(matched, c.toListInfo())
		}
	}
	return matched
}

func (d *FakeDaemon) serveListContainers(w http.ResponseWriter, r *http.Request) {
	var filters map[string][]string
	if s := r.URL.Query().Get("filters"); s != "" {
		if err := json.Unmarshal([]byte(s), &filters); err != nil {
			http.Error(w, "invalid filters", http.StatusBadRequest)
			return
		}
	}
	matched := d.getContainers(filters["label"])

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(matched); err != nil {
		d.t.Errorf("encode containers: %v", err)
	}
}

func (d *FakeDaemon) serveInspectNetwork(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	n := d.getNetwork(name)
	if n == nil {
		writeNotFound(w, fmt.Sprintf("network %s not found", name))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(n.toInfo()); err != nil {
		d.t.Errorf("encode network: %v", err)
	}
}

