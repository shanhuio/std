package dockertest

import "encoding/json"

// Minimal protobuf wire-format encoder for control.StatusResponse, so the fake
// daemon can emit BuildKit-style (version=2) progress the way a real daemon
// does. It is the inverse of the decoder in the docker package.

func pbUvarint(x uint64) []byte {
	var b []byte
	for x >= 0x80 {
		b = append(b, byte(x)|0x80)
		x >>= 7
	}
	return append(b, byte(x))
}

// pbLenField encodes a length-delimited (wire type 2) field.
func pbLenField(field int, val []byte) []byte {
	out := pbUvarint(uint64(field)<<3 | 2)
	out = append(out, pbUvarint(uint64(len(val)))...)
	return append(out, val...)
}

// buildKitTrace builds a "moby.buildkit.trace" message announcing a build step
// with the given name, optionally carrying a line of log output. The aux value
// is the base64 of a StatusResponse proto, matching what dockerd returns.
func buildKitTrace(step, logMsg string) map[string]any {
	// Vertex{ digest=1, name=3 }
	vertex := append(
		pbLenField(1, []byte("sha256:"+step)),
		pbLenField(3, []byte(step))...,
	)
	// StatusResponse{ vertexes=1, logs=3 }
	status := pbLenField(1, vertex)
	if logMsg != "" {
		// VertexLog{ msg=4 }
		status = append(status, pbLenField(3, pbLenField(4, []byte(logMsg)))...)
	}
	aux, _ := json.Marshal(status) // []byte marshals to a base64 JSON string
	return map[string]any{
		"id":  "moby.buildkit.trace",
		"aux": json.RawMessage(aux),
	}
}
