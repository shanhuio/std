package docker_test

import (
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
	"shanhu.io/std/errcode"
)

func TestContInspect(t *testing.T) {
	web := &dockertest.Container{
		ID:       "abc123",
		Names:    []string{"/web"},
		Image:    "nginx:latest",
		ImageID:  "sha256:nginxid",
		Labels:   map[string]string{"role": "frontend"},
		Hostname: "web-host",
		Running:  true,
		ExitCode: 0,
		Mounts: []dockertest.ContainerMount{
			{Type: "bind", Source: "/host/data", Target: "/data", ReadOnly: true},
		},
	}
	stopped := &dockertest.Container{
		ID:       "def456",
		Names:    []string{"/old"},
		Image:    "busybox",
		Running:  false,
		ExitCode: 137,
		Error:    "killed",
	}

	t.Run("by id", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(web)

		got, err := docker.NewCont(d.Client, "abc123").Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.ID != "abc123" || got.Image != "nginx:latest" {
			t.Errorf("top-level: got %+v", got)
		}
		if got.Config.Hostname != "web-host" || got.Config.Image != "nginx:latest" ||
			got.Config.Labels["role"] != "frontend" {
			t.Errorf("config: got %+v", got.Config)
		}
		if !got.State.Running || got.State.ExitCode != 0 {
			t.Errorf("state: got %+v", got.State)
		}
		if len(got.HostConfig.Mounts) != 1 {
			t.Fatalf("mounts: got %d, want 1", len(got.HostConfig.Mounts))
		}
		m := got.HostConfig.Mounts[0]
		if m.Type != "bind" || m.Source != "/host/data" || m.Target != "/data" || !m.ReadOnly {
			t.Errorf("mount: got %+v", m)
		}
	})

	t.Run("by name", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(web)

		got, err := docker.NewCont(d.Client, "/web").Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.ID != "abc123" {
			t.Errorf("ID: got %q, want %q", got.ID, "abc123")
		}
	})

	t.Run("stopped container state", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(stopped)

		got, err := docker.NewCont(d.Client, "def456").Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.State.Running {
			t.Errorf("State.Running: got true, want false")
		}
		if got.State.ExitCode != 137 {
			t.Errorf("State.ExitCode: got %d, want 137", got.State.ExitCode)
		}
		if got.State.Error != "killed" {
			t.Errorf("State.Error: got %q, want %q", got.State.Error, "killed")
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.NewCont(d.Client, "missing").Inspect()
		if err == nil {
			t.Fatal("Inspect: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}
