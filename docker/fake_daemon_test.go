package docker_test

import (
	"testing"

	"shanhu.io/std/docker/dockertest"
)

// newDaemon starts a fake daemon and registers cleanup that closes the
// daemon and surfaces any internal errors it recorded.
func newDaemon(t *testing.T) *dockertest.FakeDaemon {
	t.Helper()
	d, err := dockertest.New()
	if err != nil {
		t.Fatalf("dockertest.New: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close fake daemon: %v", err)
		}
		if err := d.Err(); err != nil {
			t.Errorf("fake daemon: %v", err)
		}
	})
	return d
}
