package docker_test

import (
	"testing"

	"shanhu.io/std/docker/dockertest"
)

func TestPing(t *testing.T) {
	d := dockertest.New(t)
	if err := d.Client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
