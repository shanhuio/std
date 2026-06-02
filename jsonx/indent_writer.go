package jsonx

import (
	"bytes"
	"io"
)

// indentWriter is a write filter that supports shift tabbing.
type indentWriter struct {
	out     io.Writer
	e       error
	midLine bool

	indent    int
	indentStr string
}

func newIndentWriter(out io.Writer) *indentWriter {
	return &indentWriter{out: out, indentStr: "  "}
}

func (p *indentWriter) write(buf []byte) {
	if p.e != nil {
		return
	}
	_, p.e = p.out.Write(buf)
}

func (p *indentWriter) writeBytes(buf []byte) {
	if len(buf) == 0 {
		return
	}
	if !p.midLine {
		for j := 0; j < p.indent; j++ {
			p.write([]byte(p.indentStr))
		}
	}
	p.midLine = true
	p.write(buf)
}

func (p *indentWriter) writeEndl() {
	p.write([]byte("\n"))
	p.midLine = false
}

// Write writes buf, prepending the current indent at the start of each line.
func (p *indentWriter) Write(buf []byte) (int, error) {
	lines := bytes.Split(buf, []byte("\n"))
	for i, line := range lines {
		if i > 0 {
			p.writeEndl()
		}
		p.writeBytes(line)
	}
	return len(buf), nil
}

// Tab indents one level deeper.
func (p *indentWriter) Tab() { p.indent++ }

// ShiftTab indents one level shallower.
func (p *indentWriter) ShiftTab() {
	if p.indent > 0 {
		p.indent--
	}
}

// Err returns the first write error.
func (p *indentWriter) Err() error { return p.e }
