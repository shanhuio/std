package dockertest

import "shanhu.io/std/docker"

// Container is the fake daemon's internal model of a container. The various
// Docker API responses (list, inspect) are derived from it.
type Container struct {
	ID      string
	Names   []string
	Image   string
	ImageID string
	Labels  map[string]string
}

func (c *Container) toListInfo() *docker.ContListInfo {
	return &docker.ContListInfo{
		ID:      c.ID,
		Names:   c.Names,
		Image:   c.Image,
		ImageID: c.ImageID,
		Labels:  c.Labels,
	}
}
