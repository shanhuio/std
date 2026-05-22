package dockertest

import "shanhu.io/std/docker"

// Image is the fake daemon's internal model of an image.
type Image struct {
	ID          string
	Parent      string
	VirtualSize int64
	RepoTags    []string
	RepoDigests []string
	Labels      map[string]string
}

func (img *Image) toListInfo() *docker.ImageListInfo {
	return &docker.ImageListInfo{
		ID:       img.ID,
		RepoTags: img.RepoTags,
		Labels:   img.Labels,
	}
}

func (img *Image) toInfo() *docker.ImageInfo {
	return &docker.ImageInfo{
		ID:          img.ID,
		Parent:      img.Parent,
		VirtualSize: img.VirtualSize,
		RepoTags:    img.RepoTags,
		RepoDigests: img.RepoDigests,
	}
}
