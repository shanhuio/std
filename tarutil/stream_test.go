package tarutil

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type tarEntry struct {
	header *tar.Header
	body   []byte
}

func readTar(t *testing.T, bs []byte) []tarEntry {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(bs))
	var got []tarEntry
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar header: %v", err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read tar body for %q: %v", h.Name, err)
		}
		got = append(got, tarEntry{header: h, body: body})
	}
	return got
}

func writeStream(t *testing.T, s *Stream) []byte {
	t.Helper()
	var buf bytes.Buffer
	n, err := s.WriteTo(&buf)
	if err != nil {
		t.Fatalf("Stream.WriteTo: %v", err)
	}
	if n != int64(buf.Len()) {
		t.Errorf("WriteTo returned n=%d, buf has %d bytes", n, buf.Len())
	}
	return buf.Bytes()
}

func TestStreamEmpty(t *testing.T) {
	s := NewStream()
	got := readTar(t, writeStream(t, s))
	if len(got) != 0 {
		t.Errorf("empty stream: got %d entries, want 0", len(got))
	}
}

func TestStreamAddString(t *testing.T) {
	s := NewStream()
	s.AddString("hello.txt", ModeMeta(0644), "hello world")
	entries := readTar(t, writeStream(t, s))
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.header.Name != "hello.txt" {
		t.Errorf("name: got %q, want %q", e.header.Name, "hello.txt")
	}
	if e.header.Mode != 0644 {
		t.Errorf("mode: got %o, want %o", e.header.Mode, 0644)
	}
	if string(e.body) != "hello world" {
		t.Errorf("body: got %q, want %q", string(e.body), "hello world")
	}
	if e.header.Size != int64(len("hello world")) {
		t.Errorf("size: got %d, want %d", e.header.Size, len("hello world"))
	}
}

func TestStreamAddBytesEmpty(t *testing.T) {
	s := NewStream()
	s.AddBytes("empty.bin", ModeMeta(0600), nil)
	entries := readTar(t, writeStream(t, s))
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if len(e.body) != 0 {
		t.Errorf("body: got %d bytes, want 0", len(e.body))
	}
	if e.header.Size != 0 {
		t.Errorf("size: got %d, want 0", e.header.Size)
	}
}

func TestStreamMetaPropagates(t *testing.T) {
	s := NewStream()
	s.AddString("perms.txt", &Meta{Mode: 0755, UserID: 1001, GroupID: 2002}, "x")
	entries := readTar(t, writeStream(t, s))
	e := entries[0]
	if e.header.Mode != 0755 {
		t.Errorf("mode: got %o, want 0755", e.header.Mode)
	}
	if e.header.Uid != 1001 {
		t.Errorf("uid: got %d, want 1001", e.header.Uid)
	}
	if e.header.Gid != 2002 {
		t.Errorf("gid: got %d, want 2002", e.header.Gid)
	}
}

func TestStreamAddFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "src.txt")
	want := []byte("file from disk")
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	s := NewStream()
	s.AddFile("dst.txt", &Meta{Mode: 0640, UserID: 7, GroupID: 8}, path)
	entries := readTar(t, writeStream(t, s))
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.header.Name != "dst.txt" {
		t.Errorf("name: got %q, want %q", e.header.Name, "dst.txt")
	}
	if e.header.Mode != 0640 {
		t.Errorf("mode: got %o, want 0640", e.header.Mode)
	}
	if e.header.Uid != 7 || e.header.Gid != 8 {
		t.Errorf("uid/gid: got %d/%d, want 7/8", e.header.Uid, e.header.Gid)
	}
	if e.header.Size != int64(len(want)) {
		t.Errorf("size: got %d, want %d", e.header.Size, len(want))
	}
	if !bytes.Equal(e.body, want) {
		t.Errorf("body: got %q, want %q", e.body, want)
	}
}

func TestStreamAddFileDefaultMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(path, []byte("x"), 0640); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := os.Chmod(path, 0640); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	s := NewStream()
	s.AddFile("dst.txt", &Meta{}, path)
	entries := readTar(t, writeStream(t, s))
	if entries[0].header.Mode != 0640 {
		t.Errorf("mode: got %o, want 0640 (from file stat)", entries[0].header.Mode)
	}
}

func TestStreamAddFileMissing(t *testing.T) {
	s := NewStream()
	s.AddFile("dst.txt", ModeMeta(0644), filepath.Join(t.TempDir(), "nope"))
	var buf bytes.Buffer
	if _, err := s.WriteTo(&buf); err == nil {
		t.Errorf("WriteTo on missing file: expected error, got nil")
	}
}

func TestStreamOrder(t *testing.T) {
	s := NewStream()
	s.AddString("a.txt", ModeMeta(0644), "AAA")
	s.AddString("b.txt", ModeMeta(0644), "BBB")
	s.AddString("c.txt", ModeMeta(0644), "CCC")
	entries := readTar(t, writeStream(t, s))
	gotNames := []string{entries[0].header.Name, entries[1].header.Name, entries[2].header.Name}
	wantNames := []string{"a.txt", "b.txt", "c.txt"}
	for i := range wantNames {
		if gotNames[i] != wantNames[i] {
			t.Errorf("entry %d: got %q, want %q", i, gotNames[i], wantNames[i])
		}
	}
}

func TestStreamModTime(t *testing.T) {
	s := NewStream()
	s.AddString("a.txt", ModeMeta(0644), "x")
	entries := readTar(t, writeStream(t, s))
	if entries[0].header.ModTime.Equal(time.Time{}) {
		t.Errorf("modTime should be set by NewStream, got zero")
	}
	if !entries[0].header.ModTime.Equal(s.modTime.Truncate(0)) &&
		entries[0].header.ModTime.Unix() != s.modTime.Unix() {
		t.Errorf("modTime: got %v, want %v", entries[0].header.ModTime, s.modTime)
	}
}
