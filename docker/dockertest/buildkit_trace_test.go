package dockertest

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBuildKitTrace(t *testing.T) {
	m := buildKitTrace("[1/2] FROM scratch", "hello world\n")

	if m["id"] != "moby.buildkit.trace" {
		t.Errorf("id = %v, want moby.buildkit.trace", m["id"])
	}

	aux, ok := m["aux"].(json.RawMessage)
	if !ok {
		t.Fatalf("aux is %T, want json.RawMessage", m["aux"])
	}

	// aux is a base64 JSON string; decoding it yields the StatusResponse proto
	// bytes, which embed the step name and log message verbatim.
	var proto []byte
	if err := json.Unmarshal(aux, &proto); err != nil {
		t.Fatalf("aux is not a base64 string: %v", err)
	}
	if !bytes.Contains(proto, []byte("[1/2] FROM scratch")) {
		t.Errorf("proto does not contain the step name: %q", proto)
	}
	if !bytes.Contains(proto, []byte("hello world")) {
		t.Errorf("proto does not contain the log message: %q", proto)
	}
}

func TestBuildKitTraceNoLog(t *testing.T) {
	// With no log message, the message is still well-formed base64-aux JSON.
	m := buildKitTrace("exporting", "")
	aux := m["aux"].(json.RawMessage)
	var proto []byte
	if err := json.Unmarshal(aux, &proto); err != nil {
		t.Fatalf("aux is not a base64 string: %v", err)
	}
	if !bytes.Contains(proto, []byte("exporting")) {
		t.Errorf("proto does not contain the step name: %q", proto)
	}
}
