package dockertest

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"shanhu.io/std/docker"
)

// daemonData holds the fake daemon's state behind a single mutex. All
// state reads and writes go through methods on this type so the rest of
// the package never touches d.mu directly.
type daemonData struct {
	mu sync.Mutex

	version docker.VersionInfo
	conts   []*Container
	nets    []*Network
	vols    []*Volume
	imgs    []*Image

	idSeq int
	errs  []error
}

// nextID returns an incrementing identifier with the given prefix.
func (d *daemonData) nextID(prefix string) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.idSeq++
	return fmt.Sprintf("%s%d", prefix, d.idSeq)
}

func (d *daemonData) recordErr(err error) {
	if err == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.errs = append(d.errs, err)
}

func (d *daemonData) err() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.errs) == 0 {
		return nil
	}
	return errors.Join(d.errs...)
}

// Version ---------------------------------------------------------------

func (d *daemonData) setVersion(v docker.VersionInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.version = v
}

func (d *daemonData) getVersion() docker.VersionInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.version
}

// Containers ------------------------------------------------------------

func (d *daemonData) addContainer(c *Container) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.conts = append(d.conts, c)
}

// findContainer matches a container by ID or by any of its Names. Names
// are stored Docker-style with a leading "/"; lookup accepts either form.
// Caller must hold d.mu.
func (d *daemonData) findContainer(idOrName string) *Container {
	needle := strings.TrimPrefix(idOrName, "/")
	for _, c := range d.conts {
		if c.ID == idOrName {
			return c
		}
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == needle {
				return c
			}
		}
	}
	return nil
}

func (d *daemonData) getContainer(idOrName string) *Container {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.findContainer(idOrName)
}

func (d *daemonData) getContainers(wantLabels []string) []*docker.ContListInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	matched := make([]*docker.ContListInfo, 0, len(d.conts))
	for _, c := range d.conts {
		if matchesAllLabels(c.Labels, wantLabels) {
			matched = append(matched, c.toListInfo())
		}
	}
	return matched
}

// renameContainer renames an existing container; returns false if no such
// container exists.
func (d *daemonData) renameContainer(idOrName, newName string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	c := d.findContainer(idOrName)
	if c == nil {
		return false
	}
	c.Names = []string{"/" + newName}
	return true
}

// Networks --------------------------------------------------------------

func (d *daemonData) addNetwork(n *Network) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nets = append(d.nets, n)
}

func (d *daemonData) getNetwork(nameOrID string) *Network {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, n := range d.nets {
		if n.Name == nameOrID || n.ID == nameOrID {
			return n
		}
	}
	return nil
}

// Volumes ---------------------------------------------------------------

func (d *daemonData) addVolume(v *Volume) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.vols = append(d.vols, v)
}

func (d *daemonData) getVolume(name string) *Volume {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, v := range d.vols {
		if v.Name == name {
			return v
		}
	}
	return nil
}

func (d *daemonData) getVolumes(wantLabels []string) []*docker.VolumeInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	matched := make([]*docker.VolumeInfo, 0, len(d.vols))
	for _, v := range d.vols {
		if matchesAllLabels(v.Labels, wantLabels) {
			matched = append(matched, v.toInfo())
		}
	}
	return matched
}

// Images ----------------------------------------------------------------

func (d *daemonData) addImage(img *Image) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.imgs = append(d.imgs, img)
}

// findImage matches an image by ID or by any of its RepoTags. Caller must
// hold d.mu.
func (d *daemonData) findImage(nameOrID string) *Image {
	for _, img := range d.imgs {
		if img.ID == nameOrID || slices.Contains(img.RepoTags, nameOrID) {
			return img
		}
	}
	return nil
}

func (d *daemonData) getImage(nameOrID string) *Image {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.findImage(nameOrID)
}

func (d *daemonData) getImages() []*docker.ImageListInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]*docker.ImageListInfo, 0, len(d.imgs))
	for _, img := range d.imgs {
		out = append(out, img.toListInfo())
	}
	return out
}

// addImageTag appends repoTag to the matching image's RepoTags. Returns
// false if no such image exists.
func (d *daemonData) addImageTag(nameOrID, repoTag string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	img := d.findImage(nameOrID)
	if img == nil {
		return false
	}
	img.RepoTags = append(img.RepoTags, repoTag)
	return true
}
