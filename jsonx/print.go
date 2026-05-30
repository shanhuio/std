package jsonx

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strconv"
)

type printer struct {
	p *indentWriter
}

func newPrinter(w io.Writer) *printer {
	return &printer{p: newIndentWriter(w)}
}

func (p *printer) writeString(s string) {
	io.WriteString(p.p, strconv.Quote(s))
}

func (p *printer) write(v any) {
	switch v := v.(type) {
	case bool:
		io.WriteString(p.p, strconv.FormatBool(v))
	case json.Number:
		io.WriteString(p.p, v.String())
	case float64:
		s := strconv.FormatFloat(v, 'g', -1, 64)
		io.WriteString(p.p, s)
	case string:
		p.writeString(v)
	case []any:
		p.writeArray(v)
	case orderedObject:
		p.writeObject(v)
	case nil:
		io.WriteString(p.p, "null")
	}
}

func (p *printer) writeArrayItems(array []any) {
	p.p.Tab()
	defer p.p.ShiftTab()

	for _, item := range array {
		p.write(item)
		io.WriteString(p.p, ",\n")
	}
}

func (p *printer) writeArray(array []any) {
	if len(array) == 0 {
		io.WriteString(p.p, "[]")
		return
	}
	io.WriteString(p.p, "[\n")
	p.writeArrayItems(array)
	io.WriteString(p.p, "]")
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	if _, is := keywords[s]; is {
		return false
	}

	for i, r := range s {
		if r == '_' {
			continue
		}
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' && i > 0 {
			continue
		}
		return false
	}
	return true
}

func (p *printer) writeObjectItems(obj orderedObject) {
	// Members are kept in input order; only decide whether keys can be written
	// bare or must be quoted.
	identKeys := true
	for _, m := range obj {
		if !isIdent(m.key) {
			identKeys = false
			break
		}
	}

	p.p.Tab()
	defer p.p.ShiftTab()

	for _, m := range obj {
		if identKeys {
			io.WriteString(p.p, m.key)
		} else {
			p.writeString(m.key)
		}
		io.WriteString(p.p, ": ")
		p.write(m.value)
		io.WriteString(p.p, ",\n")
	}
}

func (p *printer) writeObject(obj orderedObject) {
	if len(obj) == 0 {
		io.WriteString(p.p, "{}")
		return
	}

	io.WriteString(p.p, "{\n")
	p.writeObjectItems(obj)
	io.WriteString(p.p, "}")
}

func (p *printer) err() error { return p.p.Err() }

// Fprint formats v in JSONX and prints it into w.
func Fprint(w io.Writer, v any) error {
	bs, err := json.Marshal(v)
	if err != nil {
		return err
	}

	g, err := decodeOrdered(bs)
	if err != nil {
		return err
	}

	p := newPrinter(w)
	p.write(g)
	if err := p.err(); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

// Print formats v in JSONX and prints it into stdout.
func Print(v any) error {
	return Fprint(os.Stdout, v)
}

// Sprint formats v in JSONX and returns the formatted string.
func Sprint(v any) (string, error) {
	buf := new(bytes.Buffer)
	if err := Fprint(buf, v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Marshal formats v in JSONX.
func Marshal(v any) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := Fprint(buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteFile writes a JSONX object into a file.
func WriteFile(p string, v any) error {
	bs, err := Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(p, bs, 0644)
}
