package docker_test

import (
	"archive/tar"
	"bytes"
	"sort"
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
	"shanhu.io/std/errcode"
)

func TestListImages(t *testing.T) {
	nginx := &dockertest.Image{
		ID:       "sha256:nginxid",
		RepoTags: []string{"nginx:latest", "nginx:1.25"},
		Labels:   map[string]string{"maintainer": "nginx"},
	}
	postgres := &dockertest.Image{
		ID:       "sha256:postgresid",
		RepoTags: []string{"postgres:15"},
	}

	ids := func(imgs []*docker.ImageListInfo) []string {
		out := make([]string, 0, len(imgs))
		for _, i := range imgs {
			out = append(out, i.ID)
		}
		sort.Strings(out)
		return out
	}
	eq := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	t.Run("empty daemon", func(t *testing.T) {
		d := newDaemon(t)
		got, err := docker.ListImages(d.Client)
		if err != nil {
			t.Fatalf("ListImages: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d entries, want 0", len(got))
		}
	})

	t.Run("multiple images", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)
		d.AddImage(postgres)

		got, err := docker.ListImages(d.Client)
		if err != nil {
			t.Fatalf("ListImages: %v", err)
		}
		want := []string{"sha256:nginxid", "sha256:postgresid"}
		if !eq(ids(got), want) {
			t.Errorf("ids: got %v, want %v", ids(got), want)
		}
	})

	t.Run("full payload preserved", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)

		got, err := docker.ListImages(d.Client)
		if err != nil {
			t.Fatalf("ListImages: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		img := got[0]
		if img.ID != nginx.ID || len(img.RepoTags) != 2 ||
			img.RepoTags[0] != "nginx:latest" || img.RepoTags[1] != "nginx:1.25" ||
			img.Labels["maintainer"] != "nginx" {
			t.Errorf("payload mismatch: got %+v, want from %+v", img, nginx)
		}
	})
}

func TestInspectImage(t *testing.T) {
	nginx := &dockertest.Image{
		ID:          "sha256:nginxid",
		Parent:      "sha256:parentid",
		VirtualSize: 12345678,
		RepoTags:    []string{"nginx:latest", "nginx:1.25"},
		RepoDigests: []string{"nginx@sha256:digestid"},
		Labels:      map[string]string{"maintainer": "nginx"},
	}

	t.Run("by id", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)

		got, err := docker.InspectImage(d.Client, "sha256:nginxid")
		if err != nil {
			t.Fatalf("InspectImage: %v", err)
		}
		if got.ID != nginx.ID || got.Parent != nginx.Parent ||
			got.VirtualSize != nginx.VirtualSize ||
			len(got.RepoTags) != 2 || len(got.RepoDigests) != 1 {
			t.Errorf("inspect: got %+v", got)
		}
	})

	t.Run("by tag", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)

		got, err := docker.InspectImage(d.Client, "nginx:latest")
		if err != nil {
			t.Fatalf("InspectImage: %v", err)
		}
		if got.ID != nginx.ID {
			t.Errorf("ID: got %q, want %q", got.ID, nginx.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		_, err := docker.InspectImage(d.Client, "missing:tag")
		if err == nil {
			t.Fatal("InspectImage: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("HasImage true", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(nginx)
		ok, err := docker.HasImage(d.Client, "nginx:1.25")
		if err != nil {
			t.Fatalf("HasImage: %v", err)
		}
		if !ok {
			t.Errorf("HasImage: got false, want true")
		}
	})

	t.Run("HasImage false", func(t *testing.T) {
		d := newDaemon(t)
		ok, err := docker.HasImage(d.Client, "missing:tag")
		if err != nil {
			t.Fatalf("HasImage: %v", err)
		}
		if ok {
			t.Errorf("HasImage: got true, want false")
		}
	})
}

func TestTagImage(t *testing.T) {
	t.Run("by id", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(&dockertest.Image{
			ID:       "sha256:nginxid",
			RepoTags: []string{"nginx:latest"},
		})
		if err := docker.TagImage(d.Client, "sha256:nginxid", "nginx", "1.25"); err != nil {
			t.Fatalf("TagImage: %v", err)
		}
		got, err := docker.InspectImage(d.Client, "nginx:1.25")
		if err != nil {
			t.Fatalf("InspectImage by new tag: %v", err)
		}
		if got.ID != "sha256:nginxid" {
			t.Errorf("ID: got %q, want %q", got.ID, "sha256:nginxid")
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.TagImage(d.Client, "missing", "nginx", "1.25")
		if err == nil {
			t.Fatal("TagImage: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("empty repo", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.TagImage(d.Client, "nginx:latest", "", "1.25")
		if err == nil {
			t.Fatal("TagImage with empty repo: expected error, got nil")
		}
	})

	t.Run("empty tag", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.TagImage(d.Client, "nginx:latest", "nginx", "")
		if err == nil {
			t.Fatal("TagImage with empty tag: expected error, got nil")
		}
	})
}

func TestPruneImages(t *testing.T) {
	d := newDaemon(t)
	if err := docker.PruneImages(d.Client, &docker.PruneImagesOption{}); err != nil {
		t.Fatalf("PruneImages: %v", err)
	}
	if err := docker.PruneImages(d.Client, &docker.PruneImagesOption{Unused: true}); err != nil {
		t.Fatalf("PruneImages(Unused): %v", err)
	}
}

func TestPullImage(t *testing.T) {
	d := newDaemon(t)
	if err := docker.PullImage(d.Client, "nginx", "latest"); err != nil {
		t.Fatalf("PullImage: %v", err)
	}
	got, err := docker.InspectImage(d.Client, "nginx:latest")
	if err != nil {
		t.Fatalf("InspectImage after pull: %v", err)
	}
	if len(got.RepoTags) != 1 || got.RepoTags[0] != "nginx:latest" {
		t.Errorf("RepoTags: got %v, want [nginx:latest]", got.RepoTags)
	}
}

func TestPullImageNoTag(t *testing.T) {
	d := newDaemon(t)
	if err := docker.PullImage(d.Client, "nginx", ""); err != nil {
		t.Fatalf("PullImage: %v", err)
	}
	if _, err := docker.InspectImage(d.Client, "nginx"); err != nil {
		t.Errorf("InspectImage(nginx): %v", err)
	}
}

func TestPullImageNotAllowed(t *testing.T) {
	d := newDaemon(t)
	err := docker.PullImage(d.Client, "notreal", "latest")
	if err == nil {
		t.Fatal("PullImage: expected error, got nil")
	}
	if !errcode.IsInternal(err) {
		t.Errorf("expected Internal, got %v", err)
	}
	// Image should not have been recorded.
	if _, ierr := docker.InspectImage(d.Client, "notreal:latest"); ierr == nil {
		t.Errorf("InspectImage(notreal:latest): expected error, got nil")
	}
}

func TestAllowPull(t *testing.T) {
	d := newDaemon(t)
	d.AllowPull("busybox")
	if err := docker.PullImage(d.Client, "busybox", "1"); err != nil {
		t.Fatalf("PullImage: %v", err)
	}
	if _, err := docker.InspectImage(d.Client, "busybox:1"); err != nil {
		t.Errorf("InspectImage(busybox:1): %v", err)
	}
}

func TestLoadImages(t *testing.T) {
	var buf bytes.Buffer
	if err := dockertest.WriteImageArchive(&buf, []dockertest.ImageManifestEntry{
		{RepoTags: []string{"nginx:latest"}},
	}); err != nil {
		t.Fatalf("WriteImageArchive: %v", err)
	}

	d := newDaemon(t)
	if err := docker.LoadImages(d.Client, &buf); err != nil {
		t.Fatalf("LoadImages: %v", err)
	}
	got, err := docker.InspectImage(d.Client, "nginx:latest")
	if err != nil {
		t.Fatalf("InspectImage after load: %v", err)
	}
	if len(got.RepoTags) != 1 || got.RepoTags[0] != "nginx:latest" {
		t.Errorf("RepoTags: got %v, want [nginx:latest]", got.RepoTags)
	}
}

func TestSaveImages(t *testing.T) {
	d := newDaemon(t)
	d.AddImage(&dockertest.Image{
		ID:       "sha256:nginxid",
		RepoTags: []string{"nginx:latest"},
	})
	d.AddImage(&dockertest.Image{
		ID:       "sha256:postgresid",
		RepoTags: []string{"postgres:15"},
	})

	var out bytes.Buffer
	if err := docker.SaveImages(d.Client, []string{"nginx:latest", "postgres:15"}, &out); err != nil {
		t.Fatalf("SaveImages: %v", err)
	}
	manifest, err := dockertest.ReadImageArchive(&out)
	if err != nil {
		t.Fatalf("ReadImageArchive: %v", err)
	}
	if len(manifest) != 2 {
		t.Fatalf("manifest: got %d entries, want 2", len(manifest))
	}
	wantTags := [][]string{{"nginx:latest"}, {"postgres:15"}}
	for i, e := range manifest {
		if len(e.RepoTags) != len(wantTags[i]) || e.RepoTags[0] != wantTags[i][0] {
			t.Errorf("entry %d RepoTags: got %v, want %v", i, e.RepoTags, wantTags[i])
		}
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	d1 := newDaemon(t)
	d1.AddImage(&dockertest.Image{ID: "img1", RepoTags: []string{"nginx:latest"}})

	var buf bytes.Buffer
	if err := docker.SaveImages(d1.Client, []string{"nginx:latest"}, &buf); err != nil {
		t.Fatalf("SaveImages: %v", err)
	}

	d2 := newDaemon(t)
	if err := docker.LoadImages(d2.Client, &buf); err != nil {
		t.Fatalf("LoadImages: %v", err)
	}
	infos, err := docker.ListImages(d2.Client)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(infos) != 1 || len(infos[0].RepoTags) != 1 || infos[0].RepoTags[0] != "nginx:latest" {
		t.Errorf("ListImages: got %+v, want one nginx:latest", infos)
	}
}

func TestBuildImage(t *testing.T) {
	t.Run("from tarball", func(t *testing.T) {
		d := newDaemon(t)

		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		const dockerfile = "FROM scratch\n"
		if err := tw.WriteHeader(&tar.Header{
			Name:     "Dockerfile",
			Mode:     0600,
			Size:     int64(len(dockerfile)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write([]byte(dockerfile)); err != nil {
			t.Fatalf("tar write: %v", err)
		}
		if err := tw.Close(); err != nil {
			t.Fatalf("tar close: %v", err)
		}

		if err := docker.BuildImage(d.Client, "myimage:latest", &buf); err != nil {
			t.Fatalf("BuildImage: %v", err)
		}
		if _, err := docker.InspectImage(d.Client, "myimage:latest"); err != nil {
			t.Errorf("InspectImage after build: %v", err)
		}
	})

	t.Run("from tarutil.Stream", func(t *testing.T) {
		d := newDaemon(t)
		ts := docker.NewTarStream("FROM scratch\nCMD [\"true\"]\n")
		if err := docker.BuildImageStream(d.Client, "mystream:v1", ts); err != nil {
			t.Fatalf("BuildImageStream: %v", err)
		}
		if _, err := docker.InspectImage(d.Client, "mystream:v1"); err != nil {
			t.Errorf("InspectImage after build: %v", err)
		}
	})

	t.Run("with build args and no-cache", func(t *testing.T) {
		d := newDaemon(t)
		ts := docker.NewTarStream("FROM scratch\n")
		err := docker.BuildImageConfig(d.Client, "args:1", &docker.BuildConfig{
			Files: ts,
			Args:  map[string]string{"FOO": "bar"},
		})
		if err != nil {
			t.Fatalf("BuildImageConfig: %v", err)
		}
		if _, err := docker.InspectImage(d.Client, "args:1"); err != nil {
			t.Errorf("InspectImage after build: %v", err)
		}
	})

	t.Run("no input", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.BuildImageConfig(d.Client, "tag", &docker.BuildConfig{})
		if err == nil {
			t.Fatal("BuildImageConfig: expected error, got nil")
		}
		if !errcode.IsInvalidArg(err) {
			t.Errorf("expected InvalidArg, got %v", err)
		}
	})
}

func TestRemoveImage(t *testing.T) {
	t.Run("by tag", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(&dockertest.Image{
			ID:       "sha256:nginxid",
			RepoTags: []string{"nginx:latest"},
		})
		if err := docker.RemoveImage(d.Client, "nginx:latest", &docker.RemoveImageOptions{}); err != nil {
			t.Fatalf("RemoveImage: %v", err)
		}
		if _, err := docker.InspectImage(d.Client, "nginx:latest"); err == nil {
			t.Errorf("InspectImage after Remove: expected error, got nil")
		}
	})

	t.Run("force", func(t *testing.T) {
		d := newDaemon(t)
		d.AddImage(&dockertest.Image{
			ID:       "sha256:nginxid",
			RepoTags: []string{"nginx:latest"},
		})
		if err := docker.RemoveImage(d.Client, "nginx:latest", &docker.RemoveImageOptions{Force: true, NoPrune: true}); err != nil {
			t.Fatalf("RemoveImage(Force): %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.RemoveImage(d.Client, "missing", &docker.RemoveImageOptions{})
		if err == nil {
			t.Fatal("RemoveImage: expected error, got nil")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}
