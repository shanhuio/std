package docker_test

import (
	"sort"
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
)

func TestListVolumesWithLabel(t *testing.T) {
	web := &dockertest.Volume{
		Name:       "web-data",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/web-data/_data",
		Labels:     map[string]string{"role": "frontend", "tier": "edge"},
	}
	db := &dockertest.Volume{
		Name:       "db-data",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/db-data/_data",
		Labels:     map[string]string{"role": "backend"},
	}
	cache := &dockertest.Volume{
		Name:       "cache-data",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/cache-data/_data",
	}

	names := func(vols []*docker.VolumeInfo) []string {
		out := make([]string, 0, len(vols))
		for _, v := range vols {
			out = append(out, v.Name)
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
		got, err := docker.ListVolumesWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListVolumesWithLabel: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d entries, want 0", len(got))
		}
	})

	t.Run("no match", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddVolume(cache) // no labels
		got, err := docker.ListVolumesWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListVolumesWithLabel: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d entries, want 0", len(got))
		}
	})

	t.Run("key=value match", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddVolume(web)
		d.AddVolume(db)
		d.AddVolume(cache)
		got, err := docker.ListVolumesWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListVolumesWithLabel: %v", err)
		}
		if want := []string{"web-data"}; !eq(names(got), want) {
			t.Errorf("names: got %v, want %v", names(got), want)
		}
	})

	t.Run("key-only match", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddVolume(web)
		d.AddVolume(db)
		d.AddVolume(cache)
		got, err := docker.ListVolumesWithLabel(d.Client, "role")
		if err != nil {
			t.Fatalf("ListVolumesWithLabel: %v", err)
		}
		if want := []string{"db-data", "web-data"}; !eq(names(got), want) {
			t.Errorf("names: got %v, want %v", names(got), want)
		}
	})

	t.Run("full payload preserved", func(t *testing.T) {
		d := dockertest.New(t)
		d.AddVolume(web)
		got, err := docker.ListVolumesWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListVolumesWithLabel: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		v := got[0]
		if v.Name != web.Name || v.Driver != web.Driver ||
			v.Mountpoint != web.Mountpoint ||
			v.Labels["role"] != "frontend" || v.Labels["tier"] != "edge" {
			t.Errorf("payload mismatch: got %+v, want from %+v", v, web)
		}
	})
}
