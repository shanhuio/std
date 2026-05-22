package dockertest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"shanhu.io/std/docker"
)

func serveListVolumes(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
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

func serveInspectVolume(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	v := d.data.getVolume(name)
	if v == nil {
		writeNotFound(w, fmt.Sprintf("get %s: no such volume", name))
		return
	}
	d.writeJSON(w, v.toInfo())
}

func serveCreateVolume(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
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

func serveRemoveVolume(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !d.data.removeVolume(name) {
		writeNotFound(w, fmt.Sprintf("get %s: no such volume", name))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
