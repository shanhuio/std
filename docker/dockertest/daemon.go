// Package dockertest provides a fake Docker daemon for testing docker.Client.
package dockertest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"

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

	data *daemonData
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
		data:     &daemonData{},
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
func (d *FakeDaemon) Err() error { return d.data.err() }

// SetVersion sets the version info returned by GET /version.
func (d *FakeDaemon) SetVersion(v docker.VersionInfo) { d.data.setVersion(v) }

// AddContainer adds c to the set of containers known to the fake daemon. The
// pointer is stored as-is; later mutations to *c are visible to the fake.
func (d *FakeDaemon) AddContainer(c *Container) { d.data.addContainer(c) }

// AddNetwork adds n to the set of networks known to the fake daemon. The
// pointer is stored as-is; later mutations to *n are visible to the fake.
func (d *FakeDaemon) AddNetwork(n *Network) { d.data.addNetwork(n) }

// AddVolume adds v to the set of volumes known to the fake daemon. The
// pointer is stored as-is; later mutations to *v are visible to the fake.
func (d *FakeDaemon) AddVolume(v *Volume) { d.data.addVolume(v) }

// AddImage adds img to the set of images known to the fake daemon. The
// pointer is stored as-is; later mutations to *img are visible to the fake.
func (d *FakeDaemon) AddImage(img *Image) { d.data.addImage(img) }

// writeJSON wraps the package-level writeJSON, recording any encode error
// via the data's error recorder.
func (d *FakeDaemon) writeJSON(w http.ResponseWriter, body any) {
	d.data.recordErr(writeJSON(w, body))
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
	d.handle("POST", "containers/create", d.serveCreateContainer)
	d.handle("POST", "containers/{id}/rename", d.serveRenameContainer)
	d.handle("POST", "containers/{id}/start", d.serveStartContainer)
	d.handle("POST", "containers/{id}/stop", d.serveStopContainer)
	d.handle("POST", "containers/{id}/kill", d.serveKillContainer)
	d.handle("POST", "containers/{id}/wait", d.serveWaitContainer)
	d.handle("DELETE", "containers/{id}", d.serveRemoveContainer)
	d.handle("GET", "networks/{name}", d.serveInspectNetwork)
	d.handle("POST", "networks/create", d.serveCreateNetwork)
	d.handle("DELETE", "networks/{name}", d.serveRemoveNetwork)
	d.handle("GET", "volumes", d.serveListVolumes)
	d.handle("GET", "volumes/{name}", d.serveInspectVolume)
	d.handle("POST", "volumes/create", d.serveCreateVolume)
	d.handle("DELETE", "volumes/{name}", d.serveRemoveVolume)
	d.handle("GET", "images/json", d.serveListImages)
	d.handle("GET", "images/{name}/json", d.serveInspectImage)
	d.handle("POST", "images/{name}/tag", d.serveTagImage)
	d.handle("POST", "images/prune", d.servePruneImages)
	d.handle("DELETE", "images/{name}", d.serveRemoveImage)
}

func (d *FakeDaemon) servePing(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "OK")
}

func (d *FakeDaemon) serveVersion(w http.ResponseWriter, _ *http.Request) {
	d.writeJSON(w, d.data.getVersion())
}

func (d *FakeDaemon) serveListContainers(w http.ResponseWriter, r *http.Request) {
	var filters map[string][]string
	if s := r.URL.Query().Get("filters"); s != "" {
		if err := json.Unmarshal([]byte(s), &filters); err != nil {
			http.Error(w, "invalid filters", http.StatusBadRequest)
			return
		}
	}
	d.writeJSON(w, d.data.getContainers(filters["label"]))
}

func (d *FakeDaemon) serveInspectContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c := d.data.getContainer(id)
	if c == nil {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	d.writeJSON(w, c.toInfo())
}

func (d *FakeDaemon) serveCreateContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Image    string
		Hostname string
		Labels   map[string]string
	}
	if !readJSON(w, r, &req) {
		return
	}
	name := r.URL.Query().Get("name")

	id := d.data.nextID("cont")
	c := &Container{
		ID:       id,
		Image:    req.Image,
		Hostname: req.Hostname,
		Labels:   req.Labels,
	}
	if name != "" {
		c.Names = []string{"/" + name}
	}
	d.data.addContainer(c)

	d.writeJSON(w, struct {
		ID string `json:"Id"`
	}{ID: id})
}

func (d *FakeDaemon) serveRenameContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	newName := r.URL.Query().Get("name")
	if !d.data.renameContainer(id, newName) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *FakeDaemon) serveInspectNetwork(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	n := d.data.getNetwork(name)
	if n == nil {
		writeNotFound(w, fmt.Sprintf("network %s not found", name))
		return
	}
	d.writeJSON(w, n.toInfo())
}

func (d *FakeDaemon) serveCreateNetwork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string
	}
	if !readJSON(w, r, &req) {
		return
	}
	id := d.data.nextID("net")
	d.data.addNetwork(&Network{ID: id, Name: req.Name})

	d.writeJSON(w, struct {
		ID      string `json:"Id"`
		Warning string
	}{ID: id})
}

func (d *FakeDaemon) serveListVolumes(w http.ResponseWriter, r *http.Request) {
	var filters map[string][]string
	if s := r.URL.Query().Get("filters"); s != "" {
		if err := json.Unmarshal([]byte(s), &filters); err != nil {
			http.Error(w, "invalid filters", http.StatusBadRequest)
			return
		}
	}
	resp := struct {
		Volumes []*docker.VolumeInfo
	}{Volumes: d.data.getVolumes(filters["label"])}
	d.writeJSON(w, resp)
}

func (d *FakeDaemon) serveInspectVolume(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	v := d.data.getVolume(name)
	if v == nil {
		writeNotFound(w, fmt.Sprintf("get %s: no such volume", name))
		return
	}
	d.writeJSON(w, v.toInfo())
}

func (d *FakeDaemon) serveCreateVolume(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string
		Driver string
		Labels map[string]string
	}
	if !readJSON(w, r, &req) {
		return
	}
	driver := req.Driver
	if driver == "" {
		driver = "local"
	}
	v := &Volume{
		Name:       req.Name,
		Driver:     driver,
		Mountpoint: "/var/lib/docker/volumes/" + req.Name + "/_data",
		Labels:     req.Labels,
	}
	d.data.addVolume(v)

	d.writeJSON(w, v.toInfo())
}

func (d *FakeDaemon) serveListImages(w http.ResponseWriter, _ *http.Request) {
	d.writeJSON(w, d.data.getImages())
}

func (d *FakeDaemon) serveInspectImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	img := d.data.getImage(name)
	if img == nil {
		writeNotFound(w, fmt.Sprintf("No such image: %s", name))
		return
	}
	d.writeJSON(w, img.toInfo())
}

func (d *FakeDaemon) serveTagImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	repo := r.URL.Query().Get("repo")
	tag := r.URL.Query().Get("tag")
	if !d.data.addImageTag(name, repo+":"+tag) {
		writeNotFound(w, fmt.Sprintf("No such image: %s", name))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (d *FakeDaemon) servePruneImages(w http.ResponseWriter, _ *http.Request) {
	// Real Docker reports deleted images and reclaimed space; the dock
	// client decodes into struct{}, so an empty JSON object suffices.
	d.writeJSON(w, struct{}{})
}

func (d *FakeDaemon) serveStartContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.startContainer(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *FakeDaemon) serveStopContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	found, wasRunning := d.data.stopContainer(id)
	if !found {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	if !wasRunning {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *FakeDaemon) serveKillContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.containerExists(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *FakeDaemon) serveWaitContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	code, ok := d.data.containerExitCode(id)
	if !ok {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	d.writeJSON(w, struct {
		StatusCode int
	}{StatusCode: code})
}

func (d *FakeDaemon) serveRemoveContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.removeContainer(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *FakeDaemon) serveRemoveNetwork(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !d.data.removeNetwork(name) {
		writeNotFound(w, fmt.Sprintf("network %s not found", name))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *FakeDaemon) serveRemoveVolume(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !d.data.removeVolume(name) {
		writeNotFound(w, fmt.Sprintf("get %s: no such volume", name))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *FakeDaemon) serveRemoveImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !d.data.removeImage(name) {
		writeNotFound(w, fmt.Sprintf("No such image: %s", name))
		return
	}
	w.WriteHeader(http.StatusOK)
}
