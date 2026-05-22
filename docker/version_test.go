package docker_test

import (
	"testing"

	"shanhu.io/std/docker"
)

func TestVersion(t *testing.T) {
	d := newDaemon(t)
	want := docker.VersionInfo{
		Version:       "24.0.7",
		APIVersion:    "1.43",
		OS:            "linux",
		KernelVersion: "6.1.0",
		GoVersion:     "go1.20.10",
		Arch:          "amd64",
	}
	d.SetVersion(want)

	got, err := docker.Version(d.Client)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if *got != want {
		t.Errorf("VersionInfo: got %+v, want %+v", *got, want)
	}
}
