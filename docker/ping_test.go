package docker_test

import "testing"

func TestPing(t *testing.T) {
	d := newDaemon(t)
	if err := d.Client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
