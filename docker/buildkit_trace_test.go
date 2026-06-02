package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Minimal protobuf wire-format encoders, the inverse of pbScanner, used to
// build control.StatusResponse messages for the tests.

func pbUvarint(x uint64) []byte {
	var b []byte
	for x >= 0x80 {
		b = append(b, byte(x)|0x80)
		x >>= 7
	}
	return append(b, byte(x))
}

func pbTag(field, wire int) []byte {
	return pbUvarint(uint64(field)<<3 | uint64(wire))
}

func pbLenField(field int, val []byte) []byte {
	out := pbTag(field, 2)
	out = append(out, pbUvarint(uint64(len(val)))...)
	return append(out, val...)
}

func pbVarintField(field int, v uint64) []byte {
	return append(pbTag(field, 0), pbUvarint(v)...)
}

// bkVertexBytes encodes control.Vertex{ digest, cached, name }. The cached
// varint exercises the scanner's skip path.
func bkVertexBytes(digest, name string) []byte {
	b := pbLenField(1, []byte(digest))
	b = append(b, pbVarintField(4, 1)...)
	return append(b, pbLenField(3, []byte(name))...)
}

// bkLogBytes encodes control.VertexLog{ vertex, msg }; the vertex field is
// skipped by the decoder.
func bkLogBytes(msg string) []byte {
	b := pbLenField(1, []byte("sha256:x"))
	return append(b, pbLenField(4, []byte(msg))...)
}

// bkTraceMessage wraps a StatusResponse as a "moby.buildkit.trace" JSON
// message, base64-encoding the proto into the aux string the way the daemon does.
func bkTraceMessage(t *testing.T, status []byte) string {
	t.Helper()
	aux, err := json.Marshal(status) // []byte marshals to a base64 JSON string
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(`{"id":"moby.buildkit.trace","aux":%s}`, aux)
}

func TestPrintStreamMessageBuildKitTrace(t *testing.T) {
	status := pbLenField(1, bkVertexBytes("sha256:a", "[1/2] FROM scratch"))
	status = append(status, pbLenField(3, bkLogBytes("hello from run\n"))...)

	var out bytes.Buffer
	if err := printStreamMessage(
		io.NopCloser(strings.NewReader(bkTraceMessage(t, status))), &out,
	); err != nil {
		t.Fatalf("printStreamMessage: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "#1 [1/2] FROM scratch") {
		t.Errorf("missing step header in %q", s)
	}
	if !strings.Contains(s, "hello from run") {
		t.Errorf("missing log output in %q", s)
	}
}

func TestPrintStreamMessageBuildKitDedup(t *testing.T) {
	msg1 := bkTraceMessage(t, pbLenField(1, bkVertexBytes("sha256:a", "[1/2] FROM scratch")))

	status2 := pbLenField(1, bkVertexBytes("sha256:a", "[1/2] FROM scratch"))
	status2 = append(status2, pbLenField(1, bkVertexBytes("sha256:b", "[2/2] RUN echo"))...)
	msg2 := bkTraceMessage(t, status2)

	var out bytes.Buffer
	if err := printStreamMessage(
		io.NopCloser(strings.NewReader(msg1+msg2)), &out,
	); err != nil {
		t.Fatalf("printStreamMessage: %v", err)
	}
	s := out.String()
	// The repeated step is announced once; the new step gets the next number.
	if n := strings.Count(s, "[1/2] FROM scratch"); n != 1 {
		t.Errorf("step 1 announced %d times, want 1: %q", n, s)
	}
	if !strings.Contains(s, "#2 [2/2] RUN echo") {
		t.Errorf("missing '#2 [2/2] RUN echo' in %q", s)
	}
}

func TestPrintStreamMessageBuildKitErrorStillSurfaces(t *testing.T) {
	input := bkTraceMessage(t, pbLenField(1, bkVertexBytes("sha256:a", "[1/1] FROM scratch"))) +
		`{"error":"build failed"}`
	var out bytes.Buffer
	err := printStreamMessage(io.NopCloser(strings.NewReader(input)), &out)
	if err == nil || !strings.Contains(err.Error(), "build failed") {
		t.Fatalf("got %v, want error containing 'build failed'", err)
	}
}

func TestPrintStreamMessageBuildKitBadAux(t *testing.T) {
	// An aux that is not valid base64 is dropped silently (no error, no panic).
	input := `{"id":"moby.buildkit.trace","aux":"!!! not base64 !!!"}`
	var out bytes.Buffer
	if err := printStreamMessage(
		io.NopCloser(strings.NewReader(input)), &out,
	); err != nil {
		t.Errorf("got %v, want nil", err)
	}
	if out.Len() != 0 {
		t.Errorf("output = %q, want empty", out.String())
	}
}

func TestDecodeBKStatusTruncated(t *testing.T) {
	// A length-delimited field that claims more bytes than are present must
	// produce an error rather than panic.
	data := append(pbTag(1, 2), pbUvarint(10)...)
	if _, err := decodeBKStatus(data); err == nil {
		t.Errorf("got nil error, want truncation error")
	}
}
