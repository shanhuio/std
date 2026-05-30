// Package dockertest provides a fake Docker daemon for testing docker.Client.
package dockertest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"

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

// AllowPull marks image as pullable from the simulated registry. By
// default only "nginx" is pullable.
func (d *FakeDaemon) AllowPull(image string) { d.data.allowPull(image) }

// SetExecResponse configures the output and exit code returned by
// subsequent Cont.Exec calls.
func (d *FakeDaemon) SetExecResponse(resp ExecResponse) { d.data.setExecResponse(resp) }

// writeJSON wraps the package-level writeJSON, recording any encode error
// via the data's error recorder.
func (d *FakeDaemon) writeJSON(w http.ResponseWriter, body any) {
	d.data.recordErr(writeJSON(w, body))
}

// serveFunc is the signature shared by all dockertest HTTP handlers. The
// FakeDaemon is passed explicitly so handlers can live as free functions
// across multiple serve_*.go files.
type serveFunc func(d *FakeDaemon, w http.ResponseWriter, r *http.Request)

// handle binds a serveFunc into the mux at "METHOD <APIVersion>/<p>".
func (d *FakeDaemon) handle(method, p string, fn serveFunc) {
	d.mux.HandleFunc(
		method+" "+path.Join(docker.APIVersion, p),
		func(w http.ResponseWriter, r *http.Request) { fn(d, w, r) },
	)
}

func servePing(_ *FakeDaemon, w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "OK")
}

func serveVersion(d *FakeDaemon, w http.ResponseWriter, _ *http.Request) {
	d.writeJSON(w, d.data.getVersion())
}
