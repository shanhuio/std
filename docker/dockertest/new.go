package dockertest

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"shanhu.io/std/docker"
)

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
		data:     newDaemonData(),
	}

	d.handle("GET", "_ping", servePing)
	d.handle("HEAD", "_ping", servePing)
	d.handle("GET", "version", serveVersion)

	d.handle("GET", "containers/json", serveListContainers)
	d.handle("GET", "containers/{id}/json", serveInspectContainer)
	d.handle("POST", "containers/create", serveCreateContainer)
	d.handle("POST", "containers/{id}/rename", serveRenameContainer)
	d.handle("POST", "containers/{id}/start", serveStartContainer)
	d.handle("POST", "containers/{id}/stop", serveStopContainer)
	d.handle("POST", "containers/{id}/kill", serveKillContainer)
	d.handle("POST", "containers/{id}/wait", serveWaitContainer)
	d.handle("DELETE", "containers/{id}", serveRemoveContainer)
	d.handle("GET", "containers/{id}/logs", serveContainerLogs)
	d.handle("PUT", "containers/{id}/archive", serveCopyInArchive)
	d.handle("GET", "containers/{id}/archive", serveCopyOutArchive)
	d.handle("POST", "containers/{id}/exec", serveCreateExec)
	d.handle("POST", "exec/{id}/start", serveStartExec)
	d.handle("GET", "exec/{id}/json", serveInspectExec)

	d.handle("GET", "networks/{name}", serveInspectNetwork)
	d.handle("POST", "networks/create", serveCreateNetwork)
	d.handle("DELETE", "networks/{name}", serveRemoveNetwork)

	d.handle("GET", "volumes", serveListVolumes)
	d.handle("GET", "volumes/{name}", serveInspectVolume)
	d.handle("POST", "volumes/create", serveCreateVolume)
	d.handle("DELETE", "volumes/{name}", serveRemoveVolume)

	d.handle("GET", "images/json", serveListImages)
	d.handle("GET", "images/{name}/json", serveInspectImage)
	d.handle("POST", "images/{name}/tag", serveTagImage)
	d.handle("POST", "images/prune", servePruneImages)
	d.handle("DELETE", "images/{name}", serveRemoveImage)
	d.handle("POST", "images/create", servePullImage)
	d.handle("POST", "images/load", serveLoadImages)
	d.handle("GET", "images/get", serveSaveImages)
	d.handle("POST", "build", serveBuild)

	go func() { _ = srv.Serve(l) }()
	return d, nil
}
