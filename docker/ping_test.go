package docker_test

import (
	"testing"

	"shanhu.io/std/docker"
)

func TestPing(t *testing.T) {
	d := newDaemon(t)
	if err := docker.Ping(d.Client); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
