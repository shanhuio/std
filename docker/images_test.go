package docker_test

import (
	"sort"
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
	"shanhu.io/std/errcode"
)

func TestListImages(t *testing.T) {
	nginx := &dockertest.Image{
		ID:       "sha256:nginxid",
		RepoTags: []string{"nginx:latest", "nginx:1.25"},
		Labels:   map[string]string{"maintainer": "nginx"},
	}
	postgres := &dockertest.Image{
		ID:       "sha256:postgresid",
		RepoTags: []string{"postgres:15"},
	}

	ids := func(imgs []*docker.ImageListInfo) []string {
		out := make([]string, 0, len(imgs))
		for _, i := range imgs {
			out = append(out, i.ID)
		}
		sort.Strings(out)
		return out
	}
	eq := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	t.Run("empty daemon", func(t *testing.T) {
		d := newDaemon(t)
		got, err := docker.ListImages(d.Client)
		if err != nil {
			t.Fatalf("ListImages: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d entries, want 0", len(got))
		}
	})

	t.Run("multiple images", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)
		d.AddImage(postgres)

		got, err := docker.ListImages(d.Client)
		if err != nil {
			t.Fatalf("ListImages: %v", err)
		}
		want := []string{"sha256:nginxid", "sha256:postgresid"}
		if !eq(ids(got), want) {
			t.Errorf("ids: got %v, want %v", ids(got), want)
		}
	})

	t.Run("full payload preserved", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)

		got, err := docker.ListImages(d.Client)
		if err != nil {
			t.Fatalf("ListImages: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		img := got[0]
		if img.ID != nginx.ID || len(img.RepoTags) != 2 ||
			img.RepoTags[0] != "nginx:latest" || img.RepoTags[1] != "nginx:1.25" ||
			img.Labels["maintainer"] != "nginx" {
			t.Errorf("payload mismatch: got %+v, want from %+v", img, nginx)
		}
	})
}

func TestInspectImage(t *testing.T) {
	nginx := &dockertest.Image{
		ID:          "sha256:nginxid",
		Parent:      "sha256:parentid",
		VirtualSize: 12345678,
		RepoTags:    []string{"nginx:latest", "nginx:1.25"},
		RepoDigests: []string{"nginx@sha256:digestid"},
		Labels:      map[string]string{"maintainer": "nginx"},
	}

	t.Run("by id", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)

		got, err := docker.InspectImage(d.Client, "sha256:nginxid")
		if err != nil {
			t.Fatalf("InspectImage: %v", err)
		}
		if got.ID != nginx.ID || got.Parent != nginx.Parent ||
			got.VirtualSize != nginx.VirtualSize ||
			len(got.RepoTags) != 2 || len(got.RepoDigests) != 1 {
			t.Errorf("inspect: got %+v", got)
		}
	})

	t.Run("by tag", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)

		got, err := docker.InspectImage(d.Client, "nginx:latest")
		if err != nil {
			t.Fatalf("InspectImage: %v", err)
		}
		if got.ID != nginx.ID {
			t.Errorf("ID: got %q, want %q", got.ID, nginx.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.InspectImage(d.Client, "missing:tag")
		if err == nil {
			t.Fatal("InspectImage: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("HasImage true", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)
		ok, err := docker.HasImage(d.Client, "nginx:1.25")
		if err != nil {
			t.Fatalf("HasImage: %v", err)
		}
		if !ok {
			t.Errorf("HasImage: got false, want true")
		}
	})

	t.Run("HasImage false", func(t *testing.T) {
		d := newDaemon(t)
		ok, err := docker.HasImage(d.Client, "missing:tag")
		if err != nil {
			t.Fatalf("HasImage: %v", err)
		}
		if ok {
			t.Errorf("HasImage: got true, want false")
		}
	})
}
