package docker

import (
	"encoding/json"
	"fmt"
	"io"
)

// The BuildKit builder (docker build with version=2) reports progress as JSON
// messages with id "moby.buildkit.trace" whose "aux" field is a base64 string
// holding a moby/buildkit control.StatusResponse protobuf. We decode just the
// pieces needed to stream step headers and command output, with a tiny
// hand-rolled wire reader, so we avoid depending on the (large) buildkit module.
const buildKitTraceID = "moby.buildkit.trace"

// pbScanner is a minimal protobuf wire-format reader. It handles the two wire
// types this decoder needs — varint (0) and length-delimited (2) — and can
// skip past the 64-bit (1) and 32-bit (5) types it does not.
type pbScanner struct {
	buf []byte
	pos int
	err error
}

func (s *pbScanner) more() bool { return s.err == nil && s.pos < len(s.buf) }

func (s *pbScanner) fail(err error) {
	if s.err == nil {
		s.err = err
	}
}

// uvarint reads a base-128 varint.
func (s *pbScanner) uvarint() uint64 {
	var x uint64
	for shift := uint(0); ; shift += 7 {
		if s.pos >= len(s.buf) {
			s.fail(io.ErrUnexpectedEOF)
			return 0
		}
		if shift >= 64 {
			s.fail(fmt.Errorf("varint overflow"))
			return 0
		}
		b := s.buf[s.pos]
		s.pos++
		x |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return x
		}
	}
}

// tag reads a field tag, returning its field number and wire type.
func (s *pbScanner) tag() (field, wire int) {
	t := s.uvarint()
	return int(t >> 3), int(t & 0x7)
}

// bytes reads a length-delimited value (wire type 2).
func (s *pbScanner) bytes() []byte {
	n := int(s.uvarint())
	if s.err != nil {
		return nil
	}
	if n < 0 || s.pos+n > len(s.buf) {
		s.fail(io.ErrUnexpectedEOF)
		return nil
	}
	b := s.buf[s.pos : s.pos+n]
	s.pos += n
	return b
}

// skip advances past a field value of the given wire type.
func (s *pbScanner) skip(wire int) {
	switch wire {
	case 0: // varint
		s.uvarint()
	case 1: // 64-bit
		s.advance(8)
	case 2: // length-delimited
		s.bytes()
	case 5: // 32-bit
		s.advance(4)
	default:
		s.fail(fmt.Errorf("unsupported wire type %d", wire))
	}
}

func (s *pbScanner) advance(n int) {
	if s.pos+n > len(s.buf) {
		s.fail(io.ErrUnexpectedEOF)
		return
	}
	s.pos += n
}

// bkVertex is one build step. Only the digest (its identity) and name are
// decoded from control.Vertex.
type bkVertex struct {
	digest string
	name   string
}

// bkStatus is the subset of control.StatusResponse we render.
type bkStatus struct {
	vertexes []bkVertex
	logs     [][]byte // control.VertexLog msg fields
}

// decodeTraceAux decodes a "moby.buildkit.trace" aux value (a JSON base64
// string) into a StatusResponse. encoding/json base64-decodes the string into
// the protobuf bytes.
func decodeTraceAux(aux json.RawMessage) (*bkStatus, error) {
	var data []byte
	if err := json.Unmarshal(aux, &data); err != nil {
		return nil, err
	}
	return decodeBKStatus(data)
}

// decodeBKStatus decodes control.StatusResponse{ vertexes=1, logs=3 }.
func decodeBKStatus(data []byte) (*bkStatus, error) {
	st := new(bkStatus)
	s := &pbScanner{buf: data}
	for s.more() {
		field, wire := s.tag()
		switch {
		case field == 1 && wire == 2:
			st.vertexes = append(st.vertexes, decodeBKVertex(s.bytes()))
		case field == 3 && wire == 2:
			st.logs = append(st.logs, decodeBKLogMsg(s.bytes()))
		default:
			s.skip(wire)
		}
	}
	return st, s.err
}

// decodeBKVertex decodes control.Vertex{ digest=1, name=3 }.
func decodeBKVertex(data []byte) bkVertex {
	var v bkVertex
	s := &pbScanner{buf: data}
	for s.more() {
		field, wire := s.tag()
		switch {
		case field == 1 && wire == 2:
			v.digest = string(s.bytes())
		case field == 3 && wire == 2:
			v.name = string(s.bytes())
		default:
			s.skip(wire)
		}
	}
	return v
}

// decodeBKLogMsg decodes the msg field (4) of control.VertexLog.
func decodeBKLogMsg(data []byte) []byte {
	s := &pbScanner{buf: data}
	var msg []byte
	for s.more() {
		field, wire := s.tag()
		if field == 4 && wire == 2 {
			msg = s.bytes()
			continue
		}
		s.skip(wire)
	}
	return msg
}

// printBuildKitTrace renders a decoded trace message: a numbered header for
// each newly seen named step, followed by any command output. seen carries the
// already-announced vertices across messages so each header prints once.
func printBuildKitTrace(out io.Writer, aux json.RawMessage, seen map[string]bool) {
	st, err := decodeTraceAux(aux)
	if err != nil {
		return // progress that cannot be decoded is dropped; errors still surface
	}
	for _, v := range st.vertexes {
		if v.name == "" || seen[v.digest] {
			continue
		}
		seen[v.digest] = true
		fmt.Fprintf(out, "#%d %s\n", len(seen), v.name)
	}
	for _, msg := range st.logs {
		out.Write(msg)
	}
}
