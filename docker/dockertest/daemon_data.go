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

	version  docker.VersionInfo
	conts    []*Container
	nets     []*Network
	vols     []*Volume
	imgs     []*Image
	execs    []*execState
	pullable map[string]bool
	execResp ExecResponse

	idSeq int
	errs  []error
}

// newDaemonData returns a daemonData with default state: "nginx" is the
// only image pullable from the simulated registry.
func newDaemonData() *daemonData {
	return &daemonData{
		pullable: map[string]bool{"nginx": true},
	}
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

// startContainer marks the container as running. Returns false if not found.
func (d *daemonData) startContainer(idOrName string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	c := d.findContainer(idOrName)
	if c == nil {
		return false
	}
	c.Running = true
	return true
}

// stopContainer marks the container as not running. Returns (found,
// wasRunning).
func (d *daemonData) stopContainer(idOrName string) (found, wasRunning bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	c := d.findContainer(idOrName)
	if c == nil {
		return false, false
	}
	wasRunning = c.Running
	c.Running = false
	return true, wasRunning
}

// containerExists reports whether a container with the given ID or name
// exists.
func (d *daemonData) containerExists(idOrName string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.findContainer(idOrName) != nil
}

// containerExitCode returns the ExitCode of the matching container, or
// (0, false) if not found.
func (d *daemonData) containerExitCode(idOrName string) (int, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	c := d.findContainer(idOrName)
	if c == nil {
		return 0, false
	}
	return c.ExitCode, true
}

// removeContainer drops the matching container from the set. Returns false
// if not found.
func (d *daemonData) removeContainer(idOrName string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	needle := strings.TrimPrefix(idOrName, "/")
	for i, c := range d.conts {
		if c.ID == idOrName {
			d.conts = slices.Delete(d.conts, i, i+1)
			return true
		}
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == needle {
				d.conts = slices.Delete(d.conts, i, i+1)
				return true
			}
		}
	}
	return false
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

func (d *daemonData) removeNetwork(nameOrID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, n := range d.nets {
		if n.Name == nameOrID || n.ID == nameOrID {
			d.nets = slices.Delete(d.nets, i, i+1)
			return true
		}
	}
	return false
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

func (d *daemonData) removeVolume(name string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, v := range d.vols {
		if v.Name == name {
			d.vols = slices.Delete(d.vols, i, i+1)
			return true
		}
	}
	return false
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

// allowPull marks image as pullable from the simulated registry.
func (d *daemonData) allowPull(image string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pullable == nil {
		d.pullable = make(map[string]bool)
	}
	d.pullable[image] = true
}

// isPullable reports whether image can be pulled from the simulated
// registry.
func (d *daemonData) isPullable(image string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pullable[image]
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

// Exec ------------------------------------------------------------------

// setExecResponse sets the response that the fake daemon will produce for
// subsequent exec calls.
func (d *daemonData) setExecResponse(resp ExecResponse) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.execResp = resp
}

// createExec creates a new exec session attached to the container matching
// idOrName. Returns the new exec ID, or "" if no such container.
func (d *daemonData) createExec(idOrName string, cmd []string) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	c := d.findContainer(idOrName)
	if c == nil {
		return ""
	}
	d.idSeq++
	id := fmt.Sprintf("exec%d", d.idSeq)
	d.execs = append(d.execs, &execState{
		ID:          id,
		ContainerID: c.ID,
		Cmd:         cmd,
		Running:     false,
	})
	return id
}

// startExec marks the exec as finished and returns the configured response
// to write. Returns (resp, true) if found, (_, false) otherwise.
func (d *daemonData) startExec(execID string) (ExecResponse, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, e := range d.execs {
		if e.ID == execID {
			e.Running = false
			e.ExitCode = d.execResp.ExitCode
			return d.execResp, true
		}
	}
	return ExecResponse{}, false
}

// inspectExec returns the current Running/ExitCode of an exec, or ok=false
// if not found.
func (d *daemonData) inspectExec(execID string) (running bool, exitCode int, ok bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, e := range d.execs {
		if e.ID == execID {
			return e.Running, e.ExitCode, true
		}
	}
	return false, 0, false
}

func (d *daemonData) removeImage(nameOrID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, img := range d.imgs {
		if img.ID == nameOrID || slices.Contains(img.RepoTags, nameOrID) {
			d.imgs = slices.Delete(d.imgs, i, i+1)
			return true
		}
	}
	return false
}
