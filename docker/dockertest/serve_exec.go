package dockertest

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
)

func serveCreateExec(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Cmd []string
	}
	if !readJSON(w, r, &req) {
		return
	}
	execID := d.data.createExec(id, req.Cmd)
	if execID == "" {
		writeNotFound(w, fmt.Sprintf("No such container: %s", id))
		return
	}
	d.writeJSON(w, struct {
		ID string `json:"Id"`
	}{ID: execID})
}

func serveStartExec(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Consume the body; the dock client posts an empty JSON object.
	var req struct{}
	if !readJSON(w, r, &req) {
		return
	}
	resp, ok := d.data.startExec(id)
	if !ok {
		writeNotFound(w, fmt.Sprintf("No such exec instance: %s", id))
		return
	}
	if err := writeLogFrame(w, streamStdout, []byte(resp.Stdout)); err != nil {
		d.data.recordErr(fmt.Errorf("write stdout frame: %w", err))
		return
	}
	if err := writeLogFrame(w, streamStderr, []byte(resp.Stderr)); err != nil {
		d.data.recordErr(fmt.Errorf("write stderr frame: %w", err))
		return
	}
}

func serveInspectExec(d *FakeDaemon, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	running, exitCode, ok := d.data.inspectExec(id)
	if !ok {
		writeNotFound(w, fmt.Sprintf("No such exec instance: %s", id))
		return
	}
	d.writeJSON(w, struct {
		ExitCode int
		Running  bool
	}{ExitCode: exitCode, Running: running})
}

// Stream identifiers in Docker's multiplexed log framing.
const (
	streamStdout = 1
	streamStderr = 2
)

// writeLogFrame writes a single chunk of Docker's multiplexed log stream:
// an 8-byte header (stream byte + 3 reserved zero bytes + 4-byte big-endian
// length) followed by the payload. Empty payloads emit no frame.
func writeLogFrame(w io.Writer, streamType byte, payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	var header [8]byte
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
