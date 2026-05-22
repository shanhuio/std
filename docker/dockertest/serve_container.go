package dockertest

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
)

func serveListContainers(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	var filters map[string][]string
	if s := r.URL.Query().Get("filters"); s != "" {
		if err := json.Unmarshal([]byte(s), &filters); err != nil {
			http.Error(w, "invalid filters", http.StatusBadRequest)
			return
		}
	}
	d.writeJSON(w, d.data.getContainers(filters["label"]))
}

func serveInspectContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c := d.data.getContainer(id)
	if c == nil {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	d.writeJSON(w, c.toInfo())
}

func serveCreateContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
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

func serveRenameContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	newName := r.URL.Query().Get("name")
	if !d.data.renameContainer(id, newName) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveStartContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.startContainer(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveStopContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
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

func serveKillContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.containerExists(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveWaitContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
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

func serveRemoveContainer(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.removeContainer(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveContainerLogs(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stdout, stderr, ok := d.data.containerLogs(id)
	if !ok {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	if err := writeLogFrame(w, streamStdout, []byte(stdout)); err != nil {
		d.data.recordErr(fmt.Errorf("write stdout frame: %w", err))
		return
	}
	if err := writeLogFrame(w, streamStderr, []byte(stderr)); err != nil {
		d.data.recordErr(fmt.Errorf("write stderr frame: %w", err))
		return
	}
}

func serveCopyInArchive(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.containerExists(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	basePath := r.URL.Query().Get("path")
	tr := tar.NewReader(r.Body)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "read tar: "+err.Error(), http.StatusBadRequest)
			return
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		bs, err := io.ReadAll(tr)
		if err != nil {
			http.Error(w, "read entry: "+err.Error(), http.StatusBadRequest)
			return
		}
		d.data.containerWriteFile(id, path.Join(basePath, hdr.Name), bs)
	}
	w.WriteHeader(http.StatusOK)
}

func serveCopyOutArchive(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.data.containerExists(id) {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	p := r.URL.Query().Get("path")
	content, ok := d.data.containerReadFile(id, p)
	if !ok {
		writeNotFound(w, fmt.Sprintf("Could not find the file %s in container %s", p, id))
		return
	}
	tw := tar.NewWriter(w)
	defer func() { d.data.recordErr(tw.Close()) }()
	if err := tw.WriteHeader(&tar.Header{
		Name:     path.Base(p),
		Mode:     0644,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		d.data.recordErr(fmt.Errorf("write tar header: %w", err))
		return
	}
	if _, err := tw.Write(content); err != nil {
		d.data.recordErr(fmt.Errorf("write tar entry: %w", err))
	}
}
