package dockertest

import (
	"fmt"
	"net/http"
)

func serveInspectNetwork(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	n := d.data.getNetwork(name)
	if n == nil {
		writeNotFound(w, fmt.Sprintf("network %s not found", name))
		return
	}
	d.writeJSON(w, n.toInfo())
}

func serveCreateNetwork(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
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

func serveRemoveNetwork(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !d.data.removeNetwork(name) {
		writeNotFound(w, fmt.Sprintf("network %s not found", name))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
