package tarutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type zipEntry struct {
	name string
	body string
}

func writeTestZip(t *testing.T, path string, entries []zipEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatalf("zip create %q: %v", e.name, err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatalf("zip write %q: %v", e.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
}

func tarZipToBuf(t *testing.T, zipPath, dir string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := TarZipFile(tw, zipPath, dir); err != nil {
		t.Fatalf("TarZipFile: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	return buf.Bytes()
}

func TestTarZipFile(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "in.zip")
	writeTestZip(t, zipPath, []zipEntry{
		{name: "a.txt", body: "alpha"},
		{name: "b.txt", body: "bravo"},
	})

	entries := readTar(t, tarZipToBuf(t, zipPath, ""))
	if len(entries) != 2 {
		t.Fatalf("got %d tar entries, want 2", len(entries))
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].header.Name < entries[j].header.Name
	})

	want := map[string]string{"a.txt": "alpha", "b.txt": "bravo"}
	for _, e := range entries {
		w, ok := want[e.header.Name]
		if !ok {
			t.Errorf("unexpected entry %q", e.header.Name)
			continue
		}
		if string(e.body) != w {
			t.Errorf("body for %q: got %q, want %q", e.header.Name, e.body, w)
		}
		if e.header.Size != int64(len(w)) {
			t.Errorf("size for %q: got %d, want %d", e.header.Name, e.header.Size, len(w))
		}
	}
}

func TestTarZipFileWithDir(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "in.zip")
	writeTestZip(t, zipPath, []zipEntry{
		{name: "a.txt", body: "alpha"},
		{name: "sub/b.txt", body: "bravo"},
	})

	entries := readTar(t, tarZipToBuf(t, zipPath, "base"))
	gotNames := make([]string, 0, len(entries))
	for _, e := range entries {
		gotNames = append(gotNames, e.header.Name)
	}
	sort.Strings(gotNames)

	wantNames := []string{"base/a.txt", "base/sub/b.txt"}
	if len(gotNames) != len(wantNames) {
		t.Fatalf("got %v, want %v", gotNames, wantNames)
	}
	for i, n := range wantNames {
		if gotNames[i] != n {
			t.Errorf("entry %d: got %q, want %q", i, gotNames[i], n)
		}
	}
}

func TestTarZipFileEmpty(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "empty.zip")
	writeTestZip(t, zipPath, nil)

	entries := readTar(t, tarZipToBuf(t, zipPath, ""))
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestTarZipFileMissing(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()
	if err := TarZipFile(tw, filepath.Join(t.TempDir(), "nope.zip"), ""); err == nil {
		t.Errorf("TarZipFile on missing zip: expected error, got nil")
	}
}

func TestTarZipFileViaStream(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "in.zip")
	writeTestZip(t, zipPath, []zipEntry{
		{name: "f.txt", body: "hi"},
	})

	s := NewStream()
	s.AddZipFile("root", zipPath)

	var buf bytes.Buffer
	if _, err := s.WriteTo(&buf); err != nil {
		t.Fatalf("Stream.WriteTo: %v", err)
	}
	entries := readTar(t, buf.Bytes())
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].header.Name != "root/f.txt" {
		t.Errorf("name: got %q, want %q", entries[0].header.Name, "root/f.txt")
	}
	if string(entries[0].body) != "hi" {
		t.Errorf("body: got %q, want %q", entries[0].body, "hi")
	}
}
