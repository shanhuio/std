package docker_test

import (
	"archive/tar"
	"bytes"
	"io"
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
	"shanhu.io/std/errcode"
)

func TestContInspect(t *testing.T) {
	web := &dockertest.Container{
		ID:       "abc123",
		Names:    []string{"/web"},
		Image:    "nginx:latest",
		ImageID:  "sha256:nginxid",
		Labels:   map[string]string{"role": "frontend"},
		Hostname: "web-host",
		Running:  true,
		ExitCode: 0,
		Mounts: []dockertest.ContainerMount{
			{Type: "bind", Source: "/host/data", Target: "/data", ReadOnly: true},
		},
	}
	stopped := &dockertest.Container{
		ID:       "def456",
		Names:    []string{"/old"},
		Image:    "busybox",
		Running:  false,
		ExitCode: 137,
		Error:    "killed",
	}

	t.Run("by id", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(web)

		got, err := docker.NewCont(d.Client, "abc123").Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.ID != "abc123" || got.Image != "nginx:latest" {
			t.Errorf("top-level: got %+v", got)
		}
		if got.Config.Hostname != "web-host" || got.Config.Image != "nginx:latest" ||
			got.Config.Labels["role"] != "frontend" {
			t.Errorf("config: got %+v", got.Config)
		}
		if !got.State.Running || got.State.ExitCode != 0 {
			t.Errorf("state: got %+v", got.State)
		}
		if len(got.HostConfig.Mounts) != 1 {
			t.Fatalf("mounts: got %d, want 1", len(got.HostConfig.Mounts))
		}
		m := got.HostConfig.Mounts[0]
		if m.Type != "bind" || m.Source != "/host/data" || m.Target != "/data" || !m.ReadOnly {
			t.Errorf("mount: got %+v", m)
		}
	})

	t.Run("by name", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(web)

		got, err := docker.NewCont(d.Client, "/web").Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.ID != "abc123" {
			t.Errorf("ID: got %q, want %q", got.ID, "abc123")
		}
	})

	t.Run("stopped container state", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(stopped)

		got, err := docker.NewCont(d.Client, "def456").Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.State.Running {
			t.Errorf("State.Running: got true, want false")
		}
		if got.State.ExitCode != 137 {
			t.Errorf("State.ExitCode: got %d, want 137", got.State.ExitCode)
		}
		if got.State.Error != "killed" {
			t.Errorf("State.Error: got %q, want %q", got.State.Error, "killed")
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.NewCont(d.Client, "missing").Inspect()
		if err == nil {
			t.Fatal("Inspect: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}

func TestCreateCont(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		d := newDaemon(t)
		cont, err := docker.CreateCont(d.Client, "nginx:latest", &docker.ContConfig{
			Name:     "web",
			Hostname: "web-host",
			Labels:   map[string]string{"role": "frontend"},
		})
		if err != nil {
			t.Fatalf("CreateCont: %v", err)
		}
		if cont.ID() == "" {
			t.Errorf("ID(): empty, want non-empty")
		}

		got, err := docker.NewCont(d.Client, cont.ID()).Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.Image != "nginx:latest" || got.Config.Hostname != "web-host" ||
			got.Config.Labels["role"] != "frontend" {
			t.Errorf("inspect: got %+v", got)
		}

		got2, err := docker.NewCont(d.Client, "web").Inspect()
		if err != nil {
			t.Fatalf("Inspect by name: %v", err)
		}
		if got2.ID != cont.ID() {
			t.Errorf("ID by name: got %q, want %q", got2.ID, cont.ID())
		}
	})

	t.Run("no name", func(t *testing.T) {
		d := newDaemon(t)
		cont, err := docker.CreateCont(d.Client, "nginx:latest", nil)
		if err != nil {
			t.Fatalf("CreateCont: %v", err)
		}
		got, err := docker.NewCont(d.Client, cont.ID()).Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if got.Image != "nginx:latest" {
			t.Errorf("Image: got %q, want %q", got.Image, "nginx:latest")
		}
	})
}

func TestContLifecycle(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123"})
		c := docker.NewCont(d.Client, "abc123")
		if err := c.Start(); err != nil {
			t.Fatalf("Start: %v", err)
		}
		got, err := c.Inspect()
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if !got.State.Running {
			t.Errorf("State.Running: got false, want true")
		}
	})

	t.Run("start not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.NewCont(d.Client, "missing").Start()
		if err == nil {
			t.Fatal("Start: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("stop running", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123", Running: true})
		c := docker.NewCont(d.Client, "abc123")
		if err := c.Stop(); err != nil {
			t.Fatalf("Stop: %v", err)
		}
		got, _ := c.Inspect()
		if got.State.Running {
			t.Errorf("State.Running: got true, want false")
		}
	})

	t.Run("stop already stopped", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123", Running: false})
		// Stop returns nil (not an error) for an already-stopped container
		// because the dock client special-cases 304 Not Modified.
		if err := docker.NewCont(d.Client, "abc123").Stop(); err != nil {
			t.Fatalf("Stop already-stopped: %v", err)
		}
	})

	t.Run("stop not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.NewCont(d.Client, "missing").Stop()
		if err == nil {
			t.Fatal("Stop: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("send SIGINT", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123", Running: true})
		if err := docker.NewCont(d.Client, "abc123").SendSIGINT(); err != nil {
			t.Fatalf("SendSIGINT: %v", err)
		}
	})

	t.Run("send SIGINT not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.NewCont(d.Client, "missing").SendSIGINT()
		if err == nil {
			t.Fatal("SendSIGINT: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("wait returns exit code", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123", ExitCode: 42})
		code, err := docker.NewCont(d.Client, "abc123").Wait(docker.NotRunning)
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if code != 42 {
			t.Errorf("StatusCode: got %d, want 42", code)
		}
	})

	t.Run("wait not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.NewCont(d.Client, "missing").Wait(docker.NotRunning)
		if err == nil {
			t.Fatal("Wait: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("drop", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123", Running: true})
		c := docker.NewCont(d.Client, "abc123")
		if err := c.Drop(); err != nil {
			t.Fatalf("Drop: %v", err)
		}
		ok, err := c.Exists()
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if ok {
			t.Errorf("Exists: got true after Drop, want false")
		}
	})
}

func TestContExec(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		d.SetExecResponse(dockertest.ExecResponse{
			Stdout:   "Hello\n",
			ExitCode: 0,
		})

		var stdout, stderr bytes.Buffer
		code, err := docker.NewCont(d.Client, "abc").ExecWithSetup(&docker.ExecSetup{
			Cmd:    []string{"echo", "Hello"},
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			t.Fatalf("ExecWithSetup: %v", err)
		}
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		if stdout.String() != "Hello\n" {
			t.Errorf("stdout: got %q, want %q", stdout.String(), "Hello\n")
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr: got %q, want empty", stderr.String())
		}
	})

	t.Run("non-zero exit and stderr", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		d.SetExecResponse(dockertest.ExecResponse{
			Stderr:   "Boom\n",
			ExitCode: 42,
		})

		var stdout, stderr bytes.Buffer
		code, err := docker.NewCont(d.Client, "abc").ExecWithSetup(&docker.ExecSetup{
			Cmd:    []string{"false"},
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			t.Fatalf("ExecWithSetup: %v", err)
		}
		if code != 42 {
			t.Errorf("exit code: got %d, want 42", code)
		}
		if stderr.String() != "Boom\n" {
			t.Errorf("stderr: got %q, want %q", stderr.String(), "Boom\n")
		}
	})

	t.Run("both streams", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		d.SetExecResponse(dockertest.ExecResponse{
			Stdout:   "out-msg",
			Stderr:   "err-msg",
			ExitCode: 0,
		})

		var stdout, stderr bytes.Buffer
		if _, err := docker.NewCont(d.Client, "abc").ExecWithSetup(&docker.ExecSetup{
			Cmd:    []string{"sh", "-c", "echo out; echo err >&2"},
			Stdout: &stdout,
			Stderr: &stderr,
		}); err != nil {
			t.Fatalf("ExecWithSetup: %v", err)
		}
		if stdout.String() != "out-msg" || stderr.String() != "err-msg" {
			t.Errorf("got stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	})

	t.Run("Cont.Exec splits command", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		code, err := docker.NewCont(d.Client, "abc").Exec("ls -la /tmp")
		if err != nil {
			t.Fatalf("Exec: %v", err)
		}
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
	})

	t.Run("container not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.NewCont(d.Client, "missing").Exec("echo hi")
		if err == nil {
			t.Fatal("Exec: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}

func TestContArchive(t *testing.T) {
	t.Run("copy out file", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{
			ID: "abc",
			Files: map[string][]byte{
				"/etc/hostname": []byte("myhost\n"),
			},
		})

		var buf bytes.Buffer
		if err := docker.NewCont(d.Client, "abc").CopyOutTar("/etc/hostname", &buf); err != nil {
			t.Fatalf("CopyOutTar: %v", err)
		}
		tr := tar.NewReader(&buf)
		hdr, err := tr.Next()
		if err != nil {
			t.Fatalf("read tar: %v", err)
		}
		if hdr.Name != "hostname" {
			t.Errorf("entry name: got %q, want %q", hdr.Name, "hostname")
		}
		got, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read tar body: %v", err)
		}
		if string(got) != "myhost\n" {
			t.Errorf("content: got %q, want %q", got, "myhost\n")
		}
	})

	t.Run("ReadContFile", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{
			ID: "abc",
			Files: map[string][]byte{
				"/etc/hostname": []byte("myhost\n"),
			},
		})

		bs, err := docker.ReadContFile(docker.NewCont(d.Client, "abc"), "/etc/hostname")
		if err != nil {
			t.Fatalf("ReadContFile: %v", err)
		}
		if string(bs) != "myhost\n" {
			t.Errorf("content: got %q, want %q", bs, "myhost\n")
		}
	})

	t.Run("ReadContFile missing", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		_, err := docker.ReadContFile(docker.NewCont(d.Client, "abc"), "/missing")
		if err == nil {
			t.Fatal("ReadContFile: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("copy in then read back", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})

		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		const payload = "hello world"
		if err := tw.WriteHeader(&tar.Header{
			Name:     "data.txt",
			Mode:     0644,
			Size:     int64(len(payload)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write([]byte(payload)); err != nil {
			t.Fatalf("tar write: %v", err)
		}
		if err := tw.Close(); err != nil {
			t.Fatalf("tar close: %v", err)
		}

		if err := docker.NewCont(d.Client, "abc").CopyInTar(&buf, "/opt"); err != nil {
			t.Fatalf("CopyInTar: %v", err)
		}

		bs, err := docker.ReadContFile(docker.NewCont(d.Client, "abc"), "/opt/data.txt")
		if err != nil {
			t.Fatalf("ReadContFile after CopyIn: %v", err)
		}
		if string(bs) != payload {
			t.Errorf("content: got %q, want %q", bs, payload)
		}
	})

	t.Run("copy in container not found", func(t *testing.T) {
		d := newDaemon(t)
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		_ = tw.Close()
		err := docker.NewCont(d.Client, "missing").CopyInTar(&buf, "/opt")
		if err == nil {
			t.Fatal("CopyInTar: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("copy out container not found", func(t *testing.T) {
		d := newDaemon(t)
		var buf bytes.Buffer
		err := docker.NewCont(d.Client, "missing").CopyOutTar("/etc/hostname", &buf)
		if err == nil {
			t.Fatal("CopyOutTar: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}

func TestContRemove(t *testing.T) {
	t.Run("remove", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123"})
		c := docker.NewCont(d.Client, "abc123")
		if err := c.Remove(); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		ok, _ := c.Exists()
		if ok {
			t.Errorf("Exists: got true after Remove, want false")
		}
	})

	t.Run("force remove", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc123", Running: true})
		c := docker.NewCont(d.Client, "abc123")
		if err := c.ForceRemove(); err != nil {
			t.Fatalf("ForceRemove: %v", err)
		}
		ok, _ := c.Exists()
		if ok {
			t.Errorf("Exists: got true after ForceRemove, want false")
		}
	})

	t.Run("remove not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.NewCont(d.Client, "missing").Remove()
		if err == nil {
			t.Fatal("Remove: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}

func TestRenameCont(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{
			ID:    "abc123",
			Names: []string{"/old"},
		})
		if err := docker.RenameCont(d.Client, "abc123", "new"); err != nil {
			t.Fatalf("RenameCont: %v", err)
		}
		got, err := docker.NewCont(d.Client, "new").Inspect()
		if err != nil {
			t.Fatalf("Inspect after rename: %v", err)
		}
		if got.ID != "abc123" {
			t.Errorf("ID: got %q, want %q", got.ID, "abc123")
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.RenameCont(d.Client, "missing", "new")
		if err == nil {
			t.Fatal("RenameCont: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}
