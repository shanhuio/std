package dockertest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func serveListImages(d *FakeDaemon, w http.ResponseWriter, _ *http.Request) {
	d.writeJSON(w, d.data.getImages())
}

func serveInspectImage(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	img := d.data.getImage(name)
	if img == nil {
		writeNotFound(w, fmt.Sprintf("No such image: %s", name))
		return
	}
	d.writeJSON(w, img.toInfo())
}

func serveTagImage(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	repo := r.URL.Query().Get("repo")
	tag := r.URL.Query().Get("tag")
	if !d.data.addImageTag(name, repo+":"+tag) {
		writeNotFound(w, fmt.Sprintf("No such image: %s", name))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func servePruneImages(d *FakeDaemon, w http.ResponseWriter, _ *http.Request) {
	// Real Docker reports deleted images and reclaimed space; the dock
	// client decodes into struct{}, so an empty JSON object suffices.
	d.writeJSON(w, struct{}{})
}

func serveRemoveImage(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !d.data.removeImage(name) {
		writeNotFound(w, fmt.Sprintf("No such image: %s", name))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func servePullImage(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	image := r.URL.Query().Get("fromImage")
	tag := r.URL.Query().Get("tag")
	ref := image
	if tag != "" {
		ref = image + ":" + tag
	}

	setJSONContentType(w)
	enc := json.NewEncoder(w)

	if !d.data.isPullable(image) {
		d.data.recordErr(enc.Encode(map[string]string{
			"error": "pull access denied for " + image +
				", repository does not exist or may require 'docker login'",
		}))
		return
	}

	id := d.data.nextID("img")
	d.data.addImage(&Image{ID: id, RepoTags: []string{ref}})

	d.data.recordErr(enc.Encode(map[string]string{
		"status": "Pulling from " + image,
	}))
	d.data.recordErr(enc.Encode(map[string]string{
		"status": "Status: Downloaded newer image for " + ref,
	}))
}

func serveLoadImages(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	manifest, err := ReadImageArchive(r.Body)
	if err != nil {
		d.data.recordErr(fmt.Errorf("load body: %w", err))
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	setJSONContentType(w)
	enc := json.NewEncoder(w)
	for _, m := range manifest {
		id := d.data.nextID("img")
		d.data.addImage(&Image{ID: id, RepoTags: m.RepoTags})
		for _, tag := range m.RepoTags {
			d.data.recordErr(enc.Encode(map[string]string{
				"stream": "Loaded image: " + tag + "\n",
			}))
		}
	}
}

func serveBuild(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("t")
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		d.data.recordErr(fmt.Errorf("read build body: %w", err))
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	id := d.data.nextID("img")
	var tags []string
	if tag != "" {
		tags = []string{tag}
	}
	d.data.addImage(&Image{ID: id, RepoTags: tags})

	setJSONContentType(w)
	enc := json.NewEncoder(w)
	d.data.recordErr(enc.Encode(map[string]string{
		"stream": "Step 1/1 : FROM scratch\n",
	}))
	d.data.recordErr(enc.Encode(map[string]string{
		"stream": "Successfully built " + id + "\n",
	}))
	if tag != "" {
		d.data.recordErr(enc.Encode(map[string]string{
			"stream": "Successfully tagged " + tag + "\n",
		}))
	}
}

func serveSaveImages(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	names := r.URL.Query()["names"]
	manifest := make([]ImageManifestEntry, 0, len(names))
	for _, name := range names {
		tags := []string{name}
		if img := d.data.getImage(name); img != nil {
			tags = img.RepoTags
		}
		manifest = append(manifest, ImageManifestEntry{RepoTags: tags})
	}
	d.data.recordErr(WriteImageArchive(w, manifest))
}
