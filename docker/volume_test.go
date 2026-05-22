package docker_test

import (
	"sort"
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
	"shanhu.io/std/errcode"
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
		d := newDaemon(t)
		got, err := docker.ListVolumesWithLabel(d.Client, "role=frontend")
		if err != nil {
			t.Fatalf("ListVolumesWithLabel: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d entries, want 0", len(got))
		}
	})

	t.Run("no match", func(t *testing.T) {
		d := newDaemon(t)
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
		d := newDaemon(t)
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
		d := newDaemon(t)
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
		d := newDaemon(t)
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

func TestInspectVolume(t *testing.T) {
	vol := &dockertest.Volume{
		Name:       "data",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/data/_data",
		Labels:     map[string]string{"role": "frontend"},
	}

	t.Run("found", func(t *testing.T) {
		d := newDaemon(t)
		d.AddVolume(vol)

		got, err := docker.InspectVolume(d.Client, "data")
		if err != nil {
			t.Fatalf("InspectVolume: %v", err)
		}
		if got.Name != vol.Name || got.Driver != vol.Driver ||
			got.Mountpoint != vol.Mountpoint || got.Labels["role"] != "frontend" {
			t.Errorf("payload mismatch: got %+v, want from %+v", got, vol)
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.InspectVolume(d.Client, "missing")
		if err == nil {
			t.Fatal("InspectVolume: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}

func TestCreateVolume(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		d := newDaemon(t)
		name, err := docker.CreateVolume(d.Client, "data", &docker.VolumeConfig{
			Labels: map[string]string{"role": "frontend"},
		})
		if err != nil {
			t.Fatalf("CreateVolume: %v", err)
		}
		if name != "data" {
			t.Errorf("name: got %q, want %q", name, "data")
		}
		got, err := docker.InspectVolume(d.Client, "data")
		if err != nil {
			t.Fatalf("InspectVolume: %v", err)
		}
		if got.Labels["role"] != "frontend" {
			t.Errorf("Labels: got %+v", got.Labels)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		d := newDaemon(t)
		name, err := docker.CreateVolume(d.Client, "data", nil)
		if err != nil {
			t.Fatalf("CreateVolume: %v", err)
		}
		if name != "data" {
			t.Errorf("name: got %q, want %q", name, "data")
		}
	})

	t.Run("CreateVolumeIfNotExist existing", func(t *testing.T) {
		d := newDaemon(t)
		d.AddVolume(&dockertest.Volume{Name: "data", Driver: "local"})
		name, err := docker.CreateVolumeIfNotExist(d.Client, "data", nil)
		if err != nil {
			t.Fatalf("CreateVolumeIfNotExist: %v", err)
		}
		if name != "data" {
			t.Errorf("name: got %q, want %q", name, "data")
		}
	})

	t.Run("CreateVolumeIfNotExist missing", func(t *testing.T) {
		d := newDaemon(t)
		name, err := docker.CreateVolumeIfNotExist(d.Client, "data", nil)
		if err != nil {
			t.Fatalf("CreateVolumeIfNotExist: %v", err)
		}
		if name != "data" {
			t.Errorf("name: got %q, want %q", name, "data")
		}
		if _, err := docker.InspectVolume(d.Client, "data"); err != nil {
			t.Errorf("InspectVolume after create-if-not-exist: %v", err)
		}
	})
}

func TestRemoveVolume(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		d := newDaemon(t)
		d.AddVolume(&dockertest.Volume{Name: "data", Driver: "local"})
		if err := docker.RemoveVolume(d.Client, "data"); err != nil {
			t.Fatalf("RemoveVolume: %v", err)
		}
		if _, err := docker.InspectVolume(d.Client, "data"); err == nil {
			t.Errorf("InspectVolume after Remove: expected error, got nil")
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.RemoveVolume(d.Client, "missing")
		if err == nil {
			t.Fatal("RemoveVolume: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}
