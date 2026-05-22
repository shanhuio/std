// Package dockertest provides a fake Docker daemon for testing docker.Client.
package dockertest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"shanhu.io/std/docker"
)

// FakeDaemon is an in-process Docker daemon stand-in. It implements (a subset
// of) the Docker Engine API and listens on a unix domain socket the same
// shape as a real local daemon. Tests configure its state via the helper
// methods and exercise behavior through the exported docker.Client API.
// Encode and other internal errors are recorded and returned by Err.
type FakeDaemon struct {
	mux    *http.ServeMux
	server *http.Server
	tmpDir string

	// SockPath is the unix socket the fake daemon listens on.
	SockPath string

	// Client is a docker.Client wired to talk to this fake daemon.
	Client *docker.Client

	mu      sync.Mutex
	version docker.VersionInfo
	conts   []*Container
	nets    []*Network
	vols    []*Volume
	imgs    []*Image
	errs    []error
}

// New starts a FakeDaemon listening on a unix socket inside a temporary
// directory. Call Close when done.
func New() (*FakeDaemon, error) {
	tmpDir, err := os.MkdirTemp("", "dockertest-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	sockPath := filepath.Join(tmpDir, "docker.sock")
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("listen unix %q: %w", sockPath, err)
	}

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	d := &FakeDaemon{
		mux:      mux,
		server:   srv,
		tmpDir:   tmpDir,
		SockPath: sockPath,
		Client:   docker.NewUnixClient(sockPath),
	}
	d.registerRoutes()
	go func() { _ = srv.Serve(l) }()
	return d, nil
}

// Close shuts down the fake daemon and removes its temporary directory.
func (d *FakeDaemon) Close() error {
	shutdownErr := d.server.Shutdown(context.Background())
	rmErr := os.RemoveAll(d.tmpDir)
	if shutdownErr != nil {
		return shutdownErr
	}
	return rmErr
}

// Err returns the joined internal errors recorded by the fake daemon, or
// nil if none have been recorded. Typically checked via a deferred call at
// the end of a test.
func (d *FakeDaemon) Err() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.errs) == 0 {
		return nil
	}
	return errors.Join(d.errs...)
}

func (d *FakeDaemon) recordErr(err error) {
	if err == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.errs = append(d.errs, err)
}

// writeJSON wraps the package-level writeJSON, recording any encode error
// via recordErr.
func (d *FakeDaemon) writeJSON(w http.ResponseWriter, body any) {
	d.recordErr(writeJSON(w, body))
}

func (d *FakeDaemon) handle(method, p string, h http.HandlerFunc) {
	d.mux.HandleFunc(method+" "+path.Join(docker.APIVersion, p), h)
}

func (d *FakeDaemon) registerRoutes() {
	d.handle("GET", "_ping", d.servePing)
	d.handle("HEAD", "_ping", d.servePing)
	d.handle("GET", "version", d.serveVersion)
	d.handle("GET", "containers/json", d.serveListContainers)
	d.handle("GET", "containers/{id}/json", d.serveInspectContainer)
	d.handle("GET", "networks/{name}", d.serveInspectNetwork)
	d.handle("GET", "volumes", d.serveListVolumes)
	d.handle("GET", "volumes/{name}", d.serveInspectVolume)
	d.handle("GET", "images/json", d.serveListImages)
	d.handle("GET", "images/{name}/json", d.serveInspectImage)
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

// getContainer returns the container matching idOrName by ID or by any of
// its Names, or nil if none match. Container names are stored Docker-style
// with a leading "/"; lookup accepts either form.
func (d *FakeDaemon) getContainer(idOrName string) *Container {
	d.mu.Lock()
	defer d.mu.Unlock()
	needle := strings.TrimPrefix(idOrName, "/")
	for _, c := range d.conts {
		if c.ID == idOrName {
			return c
		}
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == needle {
				return c
			}
		}
	}
	return nil
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

// AddVolume adds v to the set of volumes known to the fake daemon. The
// pointer is stored as-is; later mutations to *v are visible to the fake.
func (d *FakeDaemon) AddVolume(v *Volume) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.vols = append(d.vols, v)
}

// getVolume returns the volume matching name, or nil if none match.
func (d *FakeDaemon) getVolume(name string) *Volume {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, v := range d.vols {
		if v.Name == name {
			return v
		}
	}
	return nil
}

func (d *FakeDaemon) getVolumes(wantLabels []string) []*docker.VolumeInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	matched := make([]*docker.VolumeInfo, 0, len(d.vols))
	for _, v := range d.vols {
		if matchesAllLabels(v.Labels, wantLabels) {
			matched = append(matched, v.toInfo())
		}
	}
	return matched
}

// AddImage adds img to the set of images known to the fake daemon. The
// pointer is stored as-is; later mutations to *img are visible to the fake.
func (d *FakeDaemon) AddImage(img *Image) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.imgs = append(d.imgs, img)
}

func (d *FakeDaemon) getImages() []*docker.ImageListInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]*docker.ImageListInfo, 0, len(d.imgs))
	for _, img := range d.imgs {
		out = append(out, img.toListInfo())
	}
	return out
}

// getImage returns the image matching nameOrID by ID or by any of its
// RepoTags, or nil if none match.
func (d *FakeDaemon) getImage(nameOrID string) *Image {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, img := range d.imgs {
		if img.ID == nameOrID || slices.Contains(img.RepoTags, nameOrID) {
			return img
		}
	}
	return nil
}

func (d *FakeDaemon) servePing(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "OK")
}

func (d *FakeDaemon) serveVersion(w http.ResponseWriter, _ *http.Request) {
	d.writeJSON(w, d.getVersion())
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
	d.writeJSON(w, matched)
}

func (d *FakeDaemon) serveInspectNetwork(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	n := d.getNetwork(name)
	if n == nil {
		writeNotFound(w, fmt.Sprintf("network %s not found", name))
		return
	}
	d.writeJSON(w, n.toInfo())
}

func (d *FakeDaemon) serveInspectContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c := d.getContainer(id)
	if c == nil {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	d.writeJSON(w, c.toInfo())
}

func (d *FakeDaemon) serveInspectVolume(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	v := d.getVolume(name)
	if v == nil {
		writeNotFound(w, fmt.Sprintf("get %s: no such volume", name))
		return
	}
	d.writeJSON(w, v.toInfo())
}

func (d *FakeDaemon) serveListVolumes(w http.ResponseWriter, r *http.Request) {
	var filters map[string][]string
	if s := r.URL.Query().Get("filters"); s != "" {
		if err := json.Unmarshal([]byte(s), &filters); err != nil {
			http.Error(w, "invalid filters", http.StatusBadRequest)
			return
		}
	}
	matched := d.getVolumes(filters["label"])
	resp := struct {
		Volumes []*docker.VolumeInfo
	}{Volumes: matched}
	d.writeJSON(w, resp)
}

func (d *FakeDaemon) serveListImages(w http.ResponseWriter, _ *http.Request) {
	d.writeJSON(w, d.getImages())
}

func (d *FakeDaemon) serveInspectImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	img := d.getImage(name)
	if img == nil {
		writeNotFound(w, fmt.Sprintf("No such image: %s", name))
		return
	}
	d.writeJSON(w, img.toInfo())
}
