package dockertest

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
)

// ImageManifestEntry is a minimal subset of the Docker/OCI image manifest
// entry used by the fake daemon's SaveImages and LoadImages endpoints.
// Real Docker manifests carry more fields (Config, Layers, layer digests);
// only RepoTags is needed for the fake.
type ImageManifestEntry struct {
	Config   string   `json:",omitempty"`
	RepoTags []string `json:",omitempty"`
	Layers   []string `json:",omitempty"`
}

// WriteImageArchive writes a Docker-style image archive tar to w whose only
// entry is a manifest.json with the given entries.
func WriteImageArchive(w io.Writer, manifest []ImageManifestEntry) error {
	bs, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(&tar.Header{
		Name: "manifest.json",
		Mode: 0644,
		Size: int64(len(bs)),
	}); err != nil {
		return fmt.Errorf("write manifest header: %w", err)
	}
	if _, err := tw.Write(bs); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return tw.Close()
}

// ReadImageArchive reads a Docker-style image archive tar from r and
// returns its manifest.json entries. Entries other than manifest.json are
// ignored.
func ReadImageArchive(r io.Reader) ([]ImageManifestEntry, error) {
	tr := tar.NewReader(r)
	var manifest []ImageManifestEntry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if hdr.Name != "manifest.json" {
			continue
		}
		bs, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read manifest: %w", err)
		}
		if err := json.Unmarshal(bs, &manifest); err != nil {
			return nil, fmt.Errorf("decode manifest: %w", err)
		}
	}
	return manifest, nil
}
