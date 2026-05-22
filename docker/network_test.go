package docker_test

import (
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
	"shanhu.io/std/errcode"
)

func TestInspectNetwork(t *testing.T) {
	t.Run("by name", func(t *testing.T) {
		d := newDaemon(t)
		d.AddNetwork(&dockertest.Network{
			ID:     "net1",
			Name:   "frontend",
			Driver: "bridge",
			IPAMConfig: []dockertest.NetworkIPAMConfig{
				{Subnet: "10.0.0.0/24", Gateway: "10.0.0.1"},
			},
		})

		got, err := docker.InspectNetwork(d.Client, "frontend")
		if err != nil {
			t.Fatalf("InspectNetwork: %v", err)
		}
		if got.ID != "net1" || got.Name != "frontend" || got.Driver != "bridge" {
			t.Errorf("network: got %+v", got)
		}
		if got.IPAM == nil || len(got.IPAM.Config) != 1 {
			t.Fatalf("IPAM: got %+v, want 1 config entry", got.IPAM)
		}
		cfg := got.IPAM.Config[0]
		if cfg.Subnet != "10.0.0.0/24" || cfg.Gateway != "10.0.0.1" {
			t.Errorf("IPAM config: got %+v", cfg)
		}
	})

	t.Run("by id", func(t *testing.T) {
		d := newDaemon(t)
		d.AddNetwork(&dockertest.Network{
			ID: "net1", Name: "frontend", Driver: "bridge",
		})

		got, err := docker.InspectNetwork(d.Client, "net1")
		if err != nil {
			t.Fatalf("InspectNetwork: %v", err)
		}
		if got.ID != "net1" {
			t.Errorf("ID: got %q, want %q", got.ID, "net1")
		}
	})

	t.Run("no IPAM when unconfigured", func(t *testing.T) {
		d := newDaemon(t)
		d.AddNetwork(&dockertest.Network{
			ID: "net1", Name: "frontend", Driver: "bridge",
		})

		got, err := docker.InspectNetwork(d.Client, "frontend")
		if err != nil {
			t.Fatalf("InspectNetwork: %v", err)
		}
		if got.IPAM != nil {
			t.Errorf("IPAM: got %+v, want nil", got.IPAM)
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.InspectNetwork(d.Client, "missing")
		if err == nil {
			t.Fatal("InspectNetwork: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}

func TestCreateNetwork(t *testing.T) {
	d := newDaemon(t)
	if err := docker.CreateNetwork(d.Client, "frontend"); err != nil {
		t.Fatalf("CreateNetwork: %v", err)
	}
	got, err := docker.InspectNetwork(d.Client, "frontend")
	if err != nil {
		t.Fatalf("InspectNetwork: %v", err)
	}
	if got.Name != "frontend" {
		t.Errorf("Name: got %q, want %q", got.Name, "frontend")
	}
	if got.ID == "" {
		t.Errorf("ID: empty, want non-empty")
	}
}

func TestRemoveNetwork(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		d := newDaemon(t)
		d.AddNetwork(&dockertest.Network{ID: "net1", Name: "frontend"})
		if err := docker.RemoveNetwork(d.Client, "frontend"); err != nil {
			t.Fatalf("RemoveNetwork: %v", err)
		}
		ok, err := docker.HasNetwork(d.Client, "frontend")
		if err != nil {
			t.Fatalf("HasNetwork: %v", err)
		}
		if ok {
			t.Errorf("HasNetwork: got true after Remove, want false")
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.RemoveNetwork(d.Client, "missing")
		if err == nil {
			t.Fatal("RemoveNetwork: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}
