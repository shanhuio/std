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

	// Inspect-only fields.
	Hostname string
	Running  bool
	ExitCode int
	Error    string
	Mounts   []ContainerMount

	// Files models the container's filesystem as absolute path ->
	// content. Populated by tests for CopyOutTar/ReadContFile reads, and
	// updated by CopyInTar requests.
	Files map[string][]byte

	// LogStdout and LogStderr are the multiplexed log content returned
	// from the /logs endpoint.
	LogStdout string
	LogStderr string
}

// ContainerMount is one mount on a Container, exposed under
// HostConfig.Mounts in the inspect response.
type ContainerMount struct {
	Type        string
	Target      string
	Source      string
	ReadOnly    bool
	Consistency string
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

func (c *Container) toInfo() *docker.ContInfo {
	info := &docker.ContInfo{ID: c.ID, Image: c.Image}
	info.Config.Image = c.Image
	info.Config.Hostname = c.Hostname
	info.Config.Labels = c.Labels
	info.State.Running = c.Running
	info.State.ExitCode = c.ExitCode
	info.State.Error = c.Error
	for _, m := range c.Mounts {
		info.HostConfig.Mounts = append(info.HostConfig.Mounts, &docker.ContMountInfo{
			Type:        m.Type,
			Target:      m.Target,
			Source:      m.Source,
			ReadOnly:    m.ReadOnly,
			Consistency: m.Consistency,
		})
	}
	return info
}
