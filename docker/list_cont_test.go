package docker_test

import (
	"sort"
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
)

func TestListContsWithLabel(t *testing.T) {
	frontend := &dockertest.Container{
		ID:      "abc123",
		Names:   []string{"/web"},
		Image:   "nginx:latest",
		ImageID: "sha256:deadbeef",
		Labels:  map[string]string{"role": "frontend", "tier": "edge"},
	}
	backend := &dockertest.Container{
		ID:     "def456",
		Names:  []string{"/api"},
		Image:  "myapi:1",
		Labels: map[string]string{"role": "backend", "tier": "edge"},
	}
	db := &dockertest.Container{
		ID:    "ghi789",
		Names: []string{"/db"},
		Image: "postgres:15",
	}

	ids := func(infos []*docker.ContListInfo) []string {
		out := make([]string, 0, len(infos))
		for _, i := range infos {
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
		d := dockertest.New(t)
		got, err := docker.ListContsWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListContsWithLabel: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d entries, want 0", len(got))
		}
	})

	t.Run("no match", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddContainer(db) // no labels at all
		got, err := docker.ListContsWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListContsWithLabel: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d entries, want 0", len(got))
		}
	})

	t.Run("key=value match", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddContainer(frontend)
		d.AddContainer(backend)
		d.AddContainer(db)
		got, err := docker.ListContsWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListContsWithLabel: %v", err)
		}
		if want := []string{"abc123"}; !eq(ids(got), want) {
			t.Errorf("ids: got %v, want %v", ids(got), want)
		}
	})

	t.Run("key-only match", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddContainer(frontend)
		d.AddContainer(backend)
		d.AddContainer(db)
		got, err := docker.ListContsWithLabel(d.Client, "role")
		if err != nil {
			t.Fatalf("ListContsWithLabel: %v", err)
		}
		if want := []string{"abc123", "def456"}; !eq(ids(got), want) {
			t.Errorf("ids: got %v, want %v", ids(got), want)
		}
	})

	t.Run("shared label matches all", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddContainer(frontend)
		d.AddContainer(backend)
		d.AddContainer(db)
		got, err := docker.ListContsWithLabel(d.Client, "tier=edge")
		if err != nil {
			t.Fatalf("ListContsWithLabel: %v", err)
		}
		if want := []string{"abc123", "def456"}; !eq(ids(got), want) {
			t.Errorf("ids: got %v, want %v", ids(got), want)
		}
	})

	t.Run("full payload preserved", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddContainer(frontend)
		got, err := docker.ListContsWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListContsWithLabel: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		c := got[0]
		if c.ID != frontend.ID || c.Image != frontend.Image ||
			c.ImageID != frontend.ImageID ||
			len(c.Names) != 1 || c.Names[0] != frontend.Names[0] ||
			c.Labels["role"] != "frontend" || c.Labels["tier"] != "edge" {
			t.Errorf("payload mismatch: got %+v, want from %+v", c, frontend)
		}
	})
}
