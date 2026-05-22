package dockertest

import "shanhu.io/std/docker"

// Volume is the fake daemon's internal model of a volume.
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
}

func (v *Volume) toInfo() *docker.VolumeInfo {
	return &docker.VolumeInfo{
		Name:       v.Name,
		Driver:     v.Driver,
		Mountpoint: v.Mountpoint,
		Labels:     v.Labels,
	}
}
